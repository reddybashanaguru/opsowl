package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/reddybashanaguru/opsowl/internal/ai"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/cases"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/documents"
	"github.com/reddybashanaguru/opsowl/internal/middleware"
	"github.com/reddybashanaguru/opsowl/internal/response"
	"github.com/reddybashanaguru/opsowl/internal/storage"
	"github.com/reddybashanaguru/opsowl/internal/tasks"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
	"github.com/reddybashanaguru/opsowl/internal/users"
)

type Server struct {
	router       *chi.Mux
	logger       *zap.Logger
	jwtMgr       *auth.JWTManager
	queries      *db.Queries
	store        storage.Client
	aiServiceURL string
}

func New(logger *zap.Logger, jwtMgr *auth.JWTManager, queries *db.Queries, store storage.Client, aiServiceURL string) *Server {
	s := &Server{
		router:       chi.NewRouter(),
		logger:       logger,
		jwtMgr:       jwtMgr,
		queries:      queries,
		store:        store,
		aiServiceURL: aiServiceURL,
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
	s.router.Use(middleware.AuditLog(s.queries, s.jwtMgr, s.logger))
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

		// Dashboard stats
		r.With(auth.JWTValidate(s.jwtMgr)).Get("/dashboard/stats", s.handleDashboardStats)

		// Users module
		usersRepo := users.NewRepository(s.queries)
		usersSvc := users.NewService(usersRepo, authSvc)
		usersHandler := users.NewHandler(usersSvc)
		users.RegisterRoutes(r, usersHandler, s.jwtMgr)

		// Timeline module (hub + emitter shared across modules)
		timelineHub := timeline.NewHub()
		timelineRepo := timeline.NewRepository(s.queries)
		timelineEmitter := timeline.NewEmitter(s.queries, timelineHub)
		timelineSvc := timeline.NewService(timelineRepo, timelineEmitter)
		timelineHandler := timeline.NewHandler(timelineSvc, timelineHub, s.jwtMgr)
		timeline.RegisterRoutes(r, timelineHandler, s.jwtMgr)

		// Cases module
		casesRepo := cases.NewRepository(s.queries)
		casesSvc := cases.NewService(casesRepo, timelineEmitter)
		casesHandler := cases.NewHandler(casesSvc)
		cases.RegisterRoutes(r, casesHandler, s.jwtMgr)

		// Tasks module
		tasksRepo := tasks.NewRepository(s.queries)
		tasksSvc := tasks.NewService(tasksRepo, timelineEmitter)
		tasksHandler := tasks.NewHandler(tasksSvc)
		tasks.RegisterRoutes(r, tasksHandler, s.jwtMgr)

		// Documents module
		docsRepo := documents.NewRepository(s.queries)
		docsSvc := documents.NewService(docsRepo, s.store, timelineEmitter, s.logger, s.aiServiceURL)
		docsHandler := documents.NewHandler(docsSvc)
		documents.RegisterRoutes(r, docsHandler, s.jwtMgr)

		// AI Copilot + AI Logging module (Feature 4 + 12)
		aiRepo := ai.NewRepository(s.queries)
		aiSvc := ai.NewService(aiRepo, timelineEmitter, s.logger, s.aiServiceURL)
		aiHandler := ai.NewHandler(aiSvc)
		ai.RegisterRoutes(r, aiHandler, s.jwtMgr)
	})
}

func (s *Server) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.queries.GetDashboardStats(r.Context())
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load dashboard stats")
		return
	}
	response.JSON(w, http.StatusOK, stats)
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
