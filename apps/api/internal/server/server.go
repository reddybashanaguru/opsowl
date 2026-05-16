package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/middleware"
	"github.com/reddybashanaguru/opsowl/internal/response"
	"github.com/reddybashanaguru/opsowl/internal/users"
)

type Server struct {
	router  *chi.Mux
	logger  *zap.Logger
	jwtMgr  *auth.JWTManager
	queries *db.Queries
}

func New(logger *zap.Logger, jwtMgr *auth.JWTManager, queries *db.Queries) *Server {
	s := &Server{
		router:  chi.NewRouter(),
		logger:  logger,
		jwtMgr:  jwtMgr,
		queries: queries,
	}
	s.setupMiddleware()
	s.setupRoutes()
	return s
}

func (s *Server) setupMiddleware() {
	rl := middleware.NewRateLimiter(100)

	s.router.Use(chimw.RequestID)
	s.router.Use(chimw.RealIP)
	s.router.Use(chimw.Recoverer)
	s.router.Use(chimw.CleanPath)
	s.router.Use(chimw.StripSlashes)
	s.router.Use(middleware.CORS)
	s.router.Use(chimw.Compress(5))
	s.router.Use(chimw.Timeout(30 * time.Second))
	s.router.Use(rl.Middleware)
}

func (s *Server) setupRoutes() {
	s.router.Get("/health", s.handleHealth)
	s.router.Get("/ready", s.handleReady)

	s.router.Route("/api/v1", func(r chi.Router) {
		// Auth module
		authRepo := auth.NewRepository(s.queries)
		authSvc := auth.NewService(authRepo, s.jwtMgr)
		authHandler := auth.NewHandler(authSvc)
		auth.RegisterRoutes(r, authHandler, s.jwtMgr)

		// Users module
		usersRepo := users.NewRepository(s.queries)
		usersSvc := users.NewService(usersRepo, authSvc)
		usersHandler := users.NewHandler(usersSvc)
		users.RegisterRoutes(r, usersHandler, s.jwtMgr)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "opsowl-api",
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) Router() chi.Router {
	return s.router
}
