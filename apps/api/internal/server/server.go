package server

import (
	"context"
	"database/sql"
	"encoding/json"
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
	"github.com/reddybashanaguru/opsowl/internal/alerts"
	"github.com/reddybashanaguru/opsowl/internal/entity_graph"
	"github.com/reddybashanaguru/opsowl/internal/closure"
	"github.com/reddybashanaguru/opsowl/internal/findings"
	"github.com/reddybashanaguru/opsowl/internal/narrative"
	"github.com/reddybashanaguru/opsowl/internal/transactions"
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
	"github.com/reddybashanaguru/opsowl/internal/scanner"
	"github.com/reddybashanaguru/opsowl/internal/insights"
	"github.com/reddybashanaguru/opsowl/internal/middleware"
	"github.com/reddybashanaguru/opsowl/internal/notes"
	"github.com/reddybashanaguru/opsowl/internal/notifications"
	"github.com/reddybashanaguru/opsowl/internal/response"
	"github.com/reddybashanaguru/opsowl/internal/search"
	"github.com/reddybashanaguru/opsowl/internal/sla"
	"github.com/reddybashanaguru/opsowl/internal/storage"
	"github.com/reddybashanaguru/opsowl/internal/subjects"
	"github.com/reddybashanaguru/opsowl/internal/tasks"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
	"github.com/reddybashanaguru/opsowl/internal/casetypes"
	"github.com/reddybashanaguru/opsowl/internal/csvimport"
	"github.com/reddybashanaguru/opsowl/internal/playbooks"
	scheduled_reports "github.com/reddybashanaguru/opsowl/internal/scheduled_reports"
	"github.com/reddybashanaguru/opsowl/internal/templates"
	"github.com/reddybashanaguru/opsowl/internal/users"
	"github.com/reddybashanaguru/opsowl/internal/webhooks"
)

type Server struct {
	router          *chi.Mux
	logger          *zap.Logger
	jwtMgr          *auth.JWTManager
	queries         *db.Queries
	sqlDB           *sql.DB
	store           storage.Client
	aiServiceURL    string
	aiServiceSecret string
	ldapCfg         *auth.LDAPConfig
	siemFwd         *siem.Forwarder
	chain           *auditchain.ChainWriter
	mailer          *mail.Sender
	maxSessions     int
	pwdPolicy       auth.PasswordPolicy
	docsSvc         *documents.Service
}

func New(ctx context.Context, logger *zap.Logger, jwtMgr *auth.JWTManager, queries *db.Queries, sqlDB *sql.DB, store storage.Client, aiServiceURL, aiServiceSecret string, ldapCfg *auth.LDAPConfig, siemURL, siemFmt string, mailCfg mail.Config, maxSessions int, pwdPolicy auth.PasswordPolicy) *Server {
	siemFwd := siem.New(siemURL, siemFmt, logger)
	chain := auditchain.NewChainWriter(ctx, queries, logger).WithSIEM(siemFwd)

	if maxSessions <= 0 {
		maxSessions = 5
	}
	s := &Server{
		router:          chi.NewRouter(),
		logger:          logger,
		jwtMgr:          jwtMgr,
		queries:         queries,
		sqlDB:           sqlDB,
		store:           store,
		aiServiceURL:    aiServiceURL,
		aiServiceSecret: aiServiceSecret,
		ldapCfg:         ldapCfg,
		siemFwd:         siemFwd,
		chain:           chain,
		mailer:          mail.New(mailCfg),
		maxSessions:     maxSessions,
		pwdPolicy:       pwdPolicy,
	}
	s.setupMiddleware()
	s.setupRoutes(ctx)
	go retention.NewScheduler(sqlDB, queries, logger).Start(ctx)
	return s
}

// DrainAuditChain flushes any buffered audit events before the process exits.
// Call this after the HTTP server has stopped accepting requests.
func (s *Server) DrainAuditChain() {
	s.chain.Drain()
}

// DrainDocumentProcessing waits up to 15 seconds for in-flight document
// processing goroutines to finish before the process exits.
func (s *Server) DrainDocumentProcessing() {
	if s.docsSvc != nil {
		s.docsSvc.DrainProcessing(15 * time.Second)
	}
}

// sseAwareTimeout applies a 30-second request timeout to all routes EXCEPT
// SSE stream endpoints. SSE connections are long-lived by design; a 30-second
// global timeout would silently kill them and break real-time updates.
func sseAwareTimeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		withTimeout := chimw.Timeout(d)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/stream") || strings.HasSuffix(r.URL.Path, "/agent") {
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

	s.router.Use(middleware.RequireTLS) // HTTPS enforcement (RBI T2); no-op unless REQUIRE_TLS=true
	s.router.Use(middleware.RequestID) // stamps X-Request-Id; must be first
	s.router.Use(chimw.RealIP)
	s.router.Use(chimw.Recoverer)
	s.router.Use(chimw.CleanPath)
	s.router.Use(chimw.StripSlashes)
	s.router.Use(middleware.CORS)
	s.router.Use(middleware.Metrics)
	s.router.Use(chimw.Compress(5))
	s.router.Use(middleware.AccessLog(s.logger, s.jwtMgr))
	s.router.Use(sseAwareTimeout(30 * time.Second))
	s.router.Use(apikeys.APIKeyMiddleware(s.sqlDB))
	s.router.Use(rl.Middleware)
	s.router.Use(middleware.Idempotency(s.sqlDB, s.logger))
	s.router.Use(middleware.AuditLog(s.queries, s.jwtMgr, s.logger, s.chain))
}

// authLoginRateLimit returns a per-IP rate limiter function for pre-auth endpoints.
// 10 req/min prevents credential stuffing while allowing legitimate login retries.
func authLoginRateLimit() func(http.Handler) http.Handler {
	rl := middleware.NewIPRateLimiter(10)
	return rl.Middleware
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
		// Per-IP limiter (10 req/min) applied only to login/refresh/MFA challenge.
		auth.RegisterRoutes(r, authHandler, s.jwtMgr, authLoginRateLimit())

		// Dashboard stats
		r.With(auth.JWTValidate(s.jwtMgr)).Get("/dashboard/stats", s.handleDashboardStats)

		// AI mode — proxies /config from the Python AI service
		r.With(auth.JWTValidate(s.jwtMgr)).Get("/system/ai-mode", s.handleAIMode)

		// Users module
		usersRepo := users.NewRepository(s.queries).WithDB(s.sqlDB)
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

		// Outbound webhooks (P8-1) — must be wired before modules that emit events
		webhookSvc := webhooks.NewService(s.queries, s.logger)
		timelineEmitter.WithWebhooks(webhookSvc)
		webhookHandler := webhooks.NewHandler(webhookSvc)
		webhooks.RegisterRoutes(r, webhookHandler, s.jwtMgr)
		go webhookSvc.StartWorker(ctx)

		// Assignment rules engine
		assignEng := assignment.NewEngine(s.sqlDB)
		assignHandler := assignment.NewHandler(s.sqlDB)
		assignment.RegisterRoutes(r, assignHandler, s.jwtMgr)

		// Cases module
		casesRepo := cases.NewRepository(s.queries).WithDB(s.sqlDB)
		casesSvc := cases.NewService(casesRepo, s.sqlDB, timelineEmitter).
			WithAssignmentEngine(assignEngineAdapter{assignEng})
		casesHandler := cases.NewHandler(casesSvc)
		cases.RegisterRoutes(r, casesHandler, s.jwtMgr)

		// Tasks module
		tasksRepo := tasks.NewRepository(s.queries)
		tasksSvc := tasks.NewService(tasksRepo, timelineEmitter).WithDB(s.sqlDB)
		tasksHandler := tasks.NewHandler(tasksSvc)
		tasks.RegisterRoutes(r, tasksHandler, s.jwtMgr)

		// Documents module
		docsRepo := documents.NewRepository(s.queries, s.sqlDB)
		docsSvc := documents.NewService(docsRepo, s.store, timelineEmitter, s.logger, s.aiServiceURL).
			WithScanner(scanner.FromEnv()).
			WithServiceSecret(s.aiServiceSecret)
		docsSvc.RecoverStuckDocuments(ctx)
		docsSvc.StartPeriodicRecovery(ctx, 10*time.Minute)
		s.docsSvc = docsSvc
		docsHandler := documents.NewHandler(docsSvc)
		documents.RegisterRoutes(r, docsHandler, s.jwtMgr)

		// AI Copilot + AI Logging module (Feature 4 + 12)
		aiRepo := ai.NewRepository(s.queries)
		aiSvc := ai.NewService(aiRepo, timelineEmitter, s.logger, s.aiServiceURL).
			WithServiceSecret(s.aiServiceSecret)
		aiHandler := ai.NewHandler(aiSvc)
		ai.RegisterRoutes(r, aiHandler, s.jwtMgr)

		// Notes & Collaboration module (Feature 6)
		notesBroker := notes.NewBroker()
		notesSvc := notes.NewService(s.queries, notesBroker, timelineEmitter)
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

		// SLA breach scheduler — started here (after notifSvc) so it can push per-assignee alerts
		go sla.NewScheduler(s.queries, s.logger).
			WithNotifications(s.sqlDB, notifSvc).
			Start(ctx)

		// Search module (Feature 11)
		searchSvc := search.NewService(s.queries).WithDB(s.sqlDB)
		searchHandler := search.NewHandler(searchSvc)
		search.RegisterRoutes(r, searchHandler, s.jwtMgr)

		// Audit Logging UI module (Feature 10)
		auditHandler := audit.NewHandler(s.queries)
		audit.RegisterRoutes(r, auditHandler, s.jwtMgr)

		// AI Insights module (P2-2)
		insightsHandler := insights.NewHandler(s.queries, s.aiServiceURL, timelineEmitter)
		insights.RegisterRoutes(r, insightsHandler, s.jwtMgr)

		// Communication Logs module (P2-4)
		commsSvc := comms.NewService(s.queries, s.aiServiceURL, s.logger, timelineEmitter)
		commsHandler := comms.NewHandler(commsSvc)
		comms.RegisterRoutes(r, commsHandler, s.jwtMgr)

		// SLA Policy Engine (P2-3)
		slaHandler := sla.NewHandler(s.queries)
		sla.RegisterRoutes(r, slaHandler, s.jwtMgr)

		// Analytics & Workload (P2-5)
		analyticsHandler := analytics.NewHandlerWithDB(s.queries, s.sqlDB)
		analytics.RegisterRoutes(r, analyticsHandler, s.jwtMgr)

		// PDF Case Report (P2-1)
		reportHandler := report.NewHandler(s.queries, s.aiServiceURL, s.logger)
		report.RegisterRoutes(r, reportHandler, s.jwtMgr)

		// STR/SAR/CTR dual-authorization workflow + FINnet 2.0 XML export
		str.RegisterRoutes(r, s.sqlDB, s.jwtMgr, timelineEmitter)

		// Dynamic config (industry labels)
		configHandler := config.NewHandler(s.sqlDB)
		config.RegisterRoutes(r, configHandler, s.jwtMgr)

		// Inbound webhook receiver
		inboundSvc := inbound.NewService(s.sqlDB, casesSvc, s.logger)
		inboundHandler := inbound.NewHandler(inboundSvc)
		inbound.RegisterRoutes(r, inboundHandler, s.jwtMgr)

		// CSV bulk import (P8-2)
		csvImportSvc := csvimport.NewService(s.queries, casesSvc, s.logger)
		csvImportHandler := csvimport.NewHandler(csvImportSvc)
		csvimport.RegisterRoutes(r, csvImportHandler, s.jwtMgr)

		// Scheduled email reports (P8-3)
		reportSchedSvc := scheduled_reports.NewService(s.queries, s.mailer, s.logger)
		reportSchedHandler := scheduled_reports.NewHandler(reportSchedSvc)
		scheduled_reports.RegisterRoutes(r, reportSchedHandler, s.jwtMgr)
		go scheduled_reports.NewScheduler(reportSchedSvc, s.logger).Start(ctx)

		// Case templates (P8-4)
		templateSvc := templates.NewService(s.queries, tasksSvc)
		templateHandler := templates.NewHandler(templateSvc)
		templates.RegisterRoutes(r, templateHandler, s.jwtMgr)

		// Case types registry
		caseTypesHandler := casetypes.NewHandler(s.sqlDB)
		casetypes.RegisterRoutes(r, caseTypesHandler, s.jwtMgr)

		// Investigation Playbooks — ordered task sequences auto-applied on case create
		playbooksRepo := playbooks.NewRepository(s.sqlDB)
		playbooksSvc := playbooks.NewService(playbooksRepo).WithTaskCreator(tasksSvc)
		playbooksHandler := playbooks.NewHandler(playbooksSvc)
		playbooks.RegisterRoutes(r, playbooksHandler, s.jwtMgr)
		casesSvc.WithPlaybookApplier(playbooksSvc)

		// Subject 360 — entity identity directory (P9-3)
		subjectsRepo := subjects.NewRepository(s.sqlDB)
		subjectsSvc := subjects.NewService(subjectsRepo)
		subjectsHandler := subjects.NewHandler(subjectsSvc)
		subjects.RegisterRoutes(r, subjectsHandler, s.jwtMgr)

		// Alert ingestion + routing automation (P9-2)
		sysUserID := resolveSystemUser(ctx, s.sqlDB)
		alertsRepo := alerts.NewRepository(s.sqlDB)
		alertsSvc := alerts.NewService(alertsRepo).
			WithCaseCreator(alertCaseCreatorAdapter{casesSvc, subjectsSvc, sysUserID, assignEng}).
			WithSubjectResolver(alertSubjectResolverAdapter{subjectsSvc, sysUserID}).
			WithInsightsTrigger(alertInsightsTriggerAdapter{insightsHandler, sysUserID})
		alertsHandler := alerts.NewHandler(alertsSvc)
		alerts.RegisterRoutes(r, alertsHandler, s.jwtMgr)

		// Connected entity graph (P9-1)
		entity_graph.RegisterRoutes(r, s.sqlDB, s.jwtMgr)

		// Findings + Evidence Linking (Tier-1 requirement)
		findingsRepo := findings.NewRepository(s.sqlDB)
		findingsSvc := findings.NewService(findingsRepo, timelineEmitter)
		findingsHandler := findings.NewHandler(findingsSvc)
		findings.RegisterRoutes(r, findingsHandler, s.jwtMgr)

		// Structured Case Closure (Tier-1 requirement)
		closureRepo := closure.NewRepository(s.sqlDB)
		closureSvc := closure.NewService(closureRepo, timelineEmitter)
		closureHandler := closure.NewHandler(closureSvc)
		closure.RegisterRoutes(r, closureHandler, s.jwtMgr)

		// AI Investigation Narrative (Sprint 4)
		narrativeRepo := narrative.NewRepository(s.sqlDB)
		narrativeSvc := narrative.NewService(narrativeRepo, s.queries, timelineEmitter, s.aiServiceURL).
			WithServiceSecret(s.aiServiceSecret)
		narrativeHandler := narrative.NewHandler(narrativeSvc)
		narrative.RegisterRoutes(r, narrativeHandler, s.jwtMgr)

		// Transaction Timeline + Flow-of-Funds (Sprint 3)
		txnRepo := transactions.NewRepository(s.sqlDB)
		txnSvc := transactions.NewService(txnRepo, timelineEmitter)
		txnHandler := transactions.NewHandler(txnSvc)
		transactions.RegisterRoutes(r, txnHandler, s.jwtMgr)

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

func (s *Server) handleAIMode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.aiServiceURL+"/config", nil)
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]string{"provider": "unknown", "mode": "unknown"})
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]string{"provider": "unknown", "mode": "unknown"})
		return
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		response.JSON(w, http.StatusOK, map[string]string{"provider": "unknown", "mode": "unknown"})
		return
	}
	response.JSON(w, http.StatusOK, result)
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
		CaseType: input.CaseType,
		SourceID: input.SourceID,
	})
}

// resolveSystemUser returns the UUID of the first super_admin/admin user in the DB.
// Used as the "created_by" actor for system-generated cases (alert automation).
// Falls back to a zero UUID if no admin exists yet; FK violation will be caught at runtime.
func resolveSystemUser(ctx context.Context, db *sql.DB) uuid.UUID {
	var id uuid.UUID
	row := db.QueryRowContext(ctx,
		`SELECT id FROM users WHERE role IN ('super_admin','admin') ORDER BY created_at LIMIT 1`)
	if err := row.Scan(&id); err != nil {
		return uuid.Nil
	}
	return id
}

// alertCaseCreatorAdapter bridges alerts.CaseCreator to cases.Service.CreateAuto.
type alertCaseCreatorAdapter struct {
	svc           *cases.Service
	subjectSvc    *subjects.Service
	systemCreator uuid.UUID
	assignEng     *assignment.Engine
}

func (a alertCaseCreatorAdapter) CreateFromAlert(ctx context.Context, p alerts.AlertCaseParams) (uuid.UUID, string, error) {
	req := cases.CreateCaseRequest{
		Title:       p.Title,
		Description: p.Description,
		Priority:    db.CasePriority(p.Priority),
		Tags:        p.Tags,
		AssignedTo:  p.AssignTo,
		CaseType:    p.CaseType,
	}
	if req.Priority == "" {
		req.Priority = db.CasePriorityMedium
	}
	// When the alert rule has no specific assignee, use the assignment engine
	// to find the least-loaded analyst/manager rather than defaulting to admin.
	if req.AssignedTo == nil && a.assignEng != nil {
		if userID, err := a.assignEng.Evaluate(ctx, assignment.MatchInput{
			Priority: string(req.Priority),
			CaseType: req.CaseType,
		}); err == nil && userID != uuid.Nil {
			req.AssignedTo = &userID
		}
	}
	c, err := a.svc.CreateAuto(ctx, a.systemCreator, req)
	if err != nil {
		return uuid.Nil, "", err
	}
	// Link subject to the auto-created case (best-effort)
	if p.SubjectID != nil {
		claims := &auth.Claims{UserID: a.systemCreator}
		_ = a.subjectSvc.LinkToCase(ctx, claims, c.ID, *p.SubjectID, "primary")
	}
	return c.ID, c.CaseNumber, nil
}

// alertInsightsTriggerAdapter wraps insights.Handler so the alerts package
// can trigger AI analysis without importing the insights package (avoids cycles).
type alertInsightsTriggerAdapter struct {
	handler       *insights.Handler
	systemCreator uuid.UUID
}

func (a alertInsightsTriggerAdapter) TriggerAnalysis(caseID uuid.UUID) {
	a.handler.TriggerAnalysis(caseID, a.systemCreator)
}

// alertSubjectResolverAdapter bridges alerts.SubjectResolver to subjects.Service.
type alertSubjectResolverAdapter struct {
	svc           *subjects.Service
	systemCreator uuid.UUID
}

func (a alertSubjectResolverAdapter) ResolveOrCreate(ctx context.Context, idType, idNumber, name string) (uuid.UUID, error) {
	claims := &auth.Claims{UserID: a.systemCreator}
	sub, _, err := a.svc.ResolveOrCreate(ctx, claims, subjects.CreateSubjectRequest{
		CanonicalName: name,
		IDType:        idType,
		IDNumber:      idNumber,
		EntityType:    "individual",
	})
	if err != nil {
		return uuid.Nil, err
	}
	return sub.ID, nil
}
