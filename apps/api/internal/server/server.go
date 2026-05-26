package server

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/reddybashanaguru/opsowl/internal/ai"
	"github.com/reddybashanaguru/opsowl/internal/analytics"
	"github.com/reddybashanaguru/opsowl/internal/apikeys"
	"github.com/reddybashanaguru/opsowl/internal/assignment"
	"github.com/reddybashanaguru/opsowl/internal/audit"
	"github.com/reddybashanaguru/opsowl/internal/auditchain"
	"github.com/reddybashanaguru/opsowl/internal/report"
	"github.com/reddybashanaguru/opsowl/internal/str"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/mail"
	"github.com/reddybashanaguru/opsowl/internal/retention"
	"github.com/reddybashanaguru/opsowl/internal/siem"
	"github.com/reddybashanaguru/opsowl/internal/cases"
	"github.com/reddybashanaguru/opsowl/internal/comms"
	"github.com/reddybashanaguru/opsowl/internal/config"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/documents"
	"github.com/reddybashanaguru/opsowl/internal/inbound"
	"github.com/reddybashanaguru/opsowl/internal/insights"
	"github.com/reddybashanaguru/opsowl/internal/middleware"
	"github.com/reddybashanaguru/opsowl/internal/notes"
	"github.com/reddybashanaguru/opsowl/internal/notifications"
	"github.com/reddybashanaguru/opsowl/internal/response"
	"github.com/reddybashanaguru/opsowl/internal/search"
	"github.com/reddybashanaguru/opsowl/internal/sla"
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
	sqlDB        *sql.DB
	store        storage.Client
	aiServiceURL string
	ldapCfg      *auth.LDAPConfig
	siemFwd      *siem.Forwarder
	chain        *auditchain.ChainWriter
	mailer       *mail.Sender
	maxSessions  int
	pwdPolicy    auth.PasswordPolicy
}

func New(ctx context.Context, logger *zap.Logger, jwtMgr *auth.JWTManager, queries *db.Queries, sqlDB *sql.DB, store storage.Client, aiServiceURL string, ldapCfg *auth.LDAPConfig, siemURL, siemFmt string, mailCfg mail.Config, maxSessions int, pwdPolicy auth.PasswordPolicy) *Server {
	siemFwd := siem.New(siemURL, siemFmt, logger)
	chain := auditchain.NewChainWriter(ctx, queries, logger).WithSIEM(siemFwd)

	if maxSessions <= 0 {
		maxSessions = 5
	}
	s := &Server{
		router:       chi.NewRouter(),
		logger:       logger,
		jwtMgr:       jwtMgr,
		queries:      queries,
		sqlDB:        sqlDB,
		store:        store,
		aiServiceURL: aiServiceURL,
		ldapCfg:      ldapCfg,
		siemFwd:      siemFwd,
		chain:        chain,
		mailer:       mail.New(mailCfg),
		maxSessions:  maxSessions,
		pwdPolicy:    pwdPolicy,
	}
	s.setupMiddleware()
	s.setupRoutes(ctx)
	go sla.NewScheduler(queries, logger).Start(ctx)
	go retention.NewScheduler(sqlDB, queries, logger).Start(ctx)
	return s
}

// sseAwareTimeout applies a 30-second request timeout to all routes EXCEPT
// SSE stream endpoints. SSE connections are long-lived by design; a 30-second
// global timeout would silently kill them and break real-time updates.
func sseAwareTimeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		withTimeout := chimw.Timeout(d)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/stream") {
				next.ServeHTTP(w, r)
				return
			}
			withTimeout.ServeHTTP(w, r)
		})
	}
}

func (s *Server) setupMiddleware() {
	rateLimit := 100
	if v, err := strconv.Atoi(os.Getenv("MAX_REQUESTS_PER_MIN")); err == nil && v > 0 {
		rateLimit = v
	}
	rl := middleware.NewRateLimiter(rateLimit, s.jwtMgr)

	s.router.Use(middleware.RequestID) // stamps X-Request-Id; must be first
	s.router.Use(chimw.RealIP)
	s.router.Use(chimw.Recoverer)
	s.router.Use(chimw.CleanPath)
	s.router.Use(chimw.StripSlashes)
	s.router.Use(middleware.CORS)
	s.router.Use(middleware.Metrics)
	s.router.Use(chimw.Compress(5))
	s.router.Use(middleware.AccessLog(s.logger, s.jwtMgr)) // structured access log with user_id + request_id
	s.router.Use(sseAwareTimeout(30 * time.Second))
	s.router.Use(apikeys.APIKeyMiddleware(s.sqlDB))
	s.router.Use(rl.Middleware)
	s.router.Use(middleware.Idempotency(s.sqlDB, s.logger)) // POST idempotency (opt-in via Idempotency-Key header)
	s.router.Use(middleware.AuditLog(s.queries, s.jwtMgr, s.logger, s.chain))
}

func (s *Server) setupRoutes(ctx context.Context) {
	s.router.Get("/health", s.handleHealth)
	s.router.Get("/ready", s.handleReady)
	s.router.Handle("/metrics", promhttp.Handler())

	s.router.Route("/api/v1", func(r chi.Router) {
		// Auth module — chain writer wired so auth audit logs join the hash chain
		authRepo := auth.NewRepository(s.queries).WithChain(s.chain)
		authSvc := auth.NewService(authRepo, s.jwtMgr).
			WithLDAP(s.ldapCfg).
			WithMaxSessions(s.maxSessions).
			WithPasswordPolicy(s.pwdPolicy)
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

		// Assignment rules engine
		assignEng := assignment.NewEngine(s.sqlDB)
		assignHandler := assignment.NewHandler(s.sqlDB)
		assignment.RegisterRoutes(r, assignHandler, s.jwtMgr)

		// Cases module
		casesRepo := cases.NewRepository(s.queries)
		casesSvc := cases.NewService(casesRepo, s.sqlDB, timelineEmitter).
			WithAssignmentEngine(assignEngineAdapter{assignEng})
		casesHandler := cases.NewHandler(casesSvc)
		cases.RegisterRoutes(r, casesHandler, s.jwtMgr)

		// Tasks module
		tasksRepo := tasks.NewRepository(s.queries)
		tasksSvc := tasks.NewService(tasksRepo, timelineEmitter)
		tasksHandler := tasks.NewHandler(tasksSvc)
		tasks.RegisterRoutes(r, tasksHandler, s.jwtMgr)

		// Documents module
		docsRepo := documents.NewRepository(s.queries, s.sqlDB)
		docsSvc := documents.NewService(docsRepo, s.store, timelineEmitter, s.logger, s.aiServiceURL)
		go docsSvc.RecoverStuckDocuments(ctx)
		docsHandler := documents.NewHandler(docsSvc)
		documents.RegisterRoutes(r, docsHandler, s.jwtMgr)

		// AI Copilot + AI Logging module (Feature 4 + 12)
		aiRepo := ai.NewRepository(s.queries)
		aiSvc := ai.NewService(aiRepo, timelineEmitter, s.logger, s.aiServiceURL)
		aiHandler := ai.NewHandler(aiSvc)
		ai.RegisterRoutes(r, aiHandler, s.jwtMgr)

		// Notes & Collaboration module (Feature 6)
		notesBroker := notes.NewBroker()
		notesSvc := notes.NewService(s.queries, notesBroker)
		notesHandler := notes.NewHandler(notesSvc, notesBroker, s.jwtMgr)
		notes.RegisterRoutes(r, notesHandler, s.jwtMgr)

		// Notifications module (Feature 9) — wire up AFTER other modules to inject back
		notifBroker := notifications.NewBroker()
		notifSvc := notifications.NewService(s.queries, notifBroker).WithMailer(s.mailer)
		notifHandler := notifications.NewHandler(notifSvc, notifBroker, s.jwtMgr)
		notifications.RegisterRoutes(r, notifHandler, s.jwtMgr)

		// Inject notification service into modules that need it
		casesSvc.WithNotifications(notifSvc)
		tasksSvc.WithNotifications(notifSvc)

		// Search module (Feature 11)
		searchSvc := search.NewService(s.queries)
		searchHandler := search.NewHandler(searchSvc)
		search.RegisterRoutes(r, searchHandler, s.jwtMgr)

		// Audit Logging UI module (Feature 10)
		auditHandler := audit.NewHandler(s.queries)
		audit.RegisterRoutes(r, auditHandler, s.jwtMgr)

		// AI Insights module (P2-2)
		insightsHandler := insights.NewHandler(s.queries, s.aiServiceURL)
		insights.RegisterRoutes(r, insightsHandler, s.jwtMgr)

		// Communication Logs module (P2-4)
		commsSvc := comms.NewService(s.queries, s.aiServiceURL, s.logger)
		commsHandler := comms.NewHandler(commsSvc)
		comms.RegisterRoutes(r, commsHandler, s.jwtMgr)

		// SLA Policy Engine (P2-3)
		slaHandler := sla.NewHandler(s.queries)
		sla.RegisterRoutes(r, slaHandler, s.jwtMgr)

		// Analytics & Workload (P2-5)
		analyticsHandler := analytics.NewHandler(s.queries)
		analytics.RegisterRoutes(r, analyticsHandler, s.jwtMgr)

		// PDF Case Report (P2-1)
		reportHandler := report.NewHandler(s.queries, s.aiServiceURL, s.logger)
		report.RegisterRoutes(r, reportHandler, s.jwtMgr)

		// STR/SAR/CTR dual-authorization workflow + FINnet 2.0 XML export
		str.RegisterRoutes(r, s.sqlDB, s.jwtMgr)

		// Dynamic config (industry labels)
		configHandler := config.NewHandler(s.sqlDB)
		config.RegisterRoutes(r, configHandler, s.jwtMgr)

		// Inbound webhook receiver
		inboundSvc := inbound.NewService(s.sqlDB, casesSvc, s.logger)
		inboundHandler := inbound.NewHandler(inboundSvc)
		inbound.RegisterRoutes(r, inboundHandler, s.jwtMgr)

		// API Key management
		apiKeysHandler := apikeys.NewHandler(s.sqlDB)
		apikeys.RegisterRoutes(r, apiKeysHandler, s.jwtMgr)

		// Data retention admin
		r.With(auth.JWTValidate(s.jwtMgr), auth.RequireRole("admin", "super_admin")).
			Get("/retention/policies", func(w http.ResponseWriter, req *http.Request) {
				policies, err := retention.ListPolicies(req.Context(), s.sqlDB)
				if err != nil {
					response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load retention policies")
					return
				}
				response.JSON(w, http.StatusOK, map[string]any{"policies": policies})
			})
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

// assignEngineAdapter bridges assignment.Engine to cases.AssignmentEngine.
type assignEngineAdapter struct {
	eng *assignment.Engine
}

func (a assignEngineAdapter) Evaluate(ctx context.Context, input cases.AssignmentInput) (uuid.UUID, error) {
	return a.eng.Evaluate(ctx, assignment.MatchInput{
		Priority: input.Priority,
		SourceID: input.SourceID,
	})
}
