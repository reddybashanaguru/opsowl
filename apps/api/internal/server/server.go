package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/reddybashanaguru/opsowl/internal/middleware"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Server struct {
	router *chi.Mux
	logger *zap.Logger
}

func New(logger *zap.Logger) *Server {
	s := &Server{
		router: chi.NewRouter(),
		logger: logger,
	}
	s.setupMiddleware()
	s.setupRoutes()
	return s
}

func (s *Server) setupMiddleware() {
	s.router.Use(chimw.RequestID)
	s.router.Use(chimw.RealIP)
	s.router.Use(chimw.Recoverer)
	s.router.Use(chimw.CleanPath)
	s.router.Use(chimw.StripSlashes)
	s.router.Use(middleware.CORS)
	s.router.Use(chimw.Compress(5))
}

func (s *Server) setupRoutes() {
	// Health — no auth required
	s.router.Get("/health", s.handleHealth)
	s.router.Get("/ready", s.handleReady)

	// All API routes — versioned
	s.router.Route("/api/v1", func(r chi.Router) {
		// Feature modules register their routes here via RegisterRoutes(r)
		// Added in order: auth → cases → timeline → tasks → documents → ai → notes → notifications → search → dashboard → audit
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "opsowl-api",
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Router exposes the chi mux so feature modules can register routes.
func (s *Server) Router() chi.Router {
	return s.router
}
