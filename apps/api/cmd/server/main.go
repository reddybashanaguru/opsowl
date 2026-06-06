package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/config"
	"github.com/reddybashanaguru/opsowl/internal/db"
	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/mail"
	"github.com/reddybashanaguru/opsowl/internal/middleware"
	"github.com/reddybashanaguru/opsowl/internal/server"
	"github.com/reddybashanaguru/opsowl/internal/storage"
	"github.com/reddybashanaguru/opsowl/internal/worker"
)

func main() {
	var logger *zap.Logger
	if os.Getenv("ENVIRONMENT") == "development" {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}
	defer logger.Sync() //nolint:errcheck

	worker.Init(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}
	logger.Info("config loaded", zap.String("env", cfg.Environment), zap.String("port", cfg.Port))

	ctx := context.Background()
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()
	logger.Info("database connected")

	// Circuit breaker: pings the pool every 15s; after 5 consecutive failures
	// the circuit opens and all non-health requests receive 503 immediately.
	cb := db.NewCircuitBreaker(5, 30*time.Second)
	go func() {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				cb.Probe(pool.Ping(context.Background()))
			case <-ctx.Done():
				return
			}
		}
	}()

	sqlDB := stdlib.OpenDBFromPool(pool)

	// Auto-run pending migrations at startup (idempotent — goose tracks applied versions).
	// The binary ships with the migrations/ directory (see Dockerfile COPY step).
	// This replaces the manual `make migrate` step and prevents version skew between
	// binary and schema.
	goose.SetLogger(goose.NopLogger())
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		logger.Fatal("database migration failed — check migrations/ directory", zap.Error(err))
	}
	logger.Info("database migrations up-to-date")

	seedAdminUser(ctx, sqlDB, logger)

	queries := dbgen.New(sqlDB)

	// AT revocation checker — checked by JWTManager.Verify on every authenticated request.
	revoker := auth.NewRevokeChecker(sqlDB)

	// Daily cleanup goroutines.
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			middleware.IdempotencyCleanup(sqlDB, logger)
			if err := revoker.Cleanup(ctx); err != nil {
				logger.Warn("revoked token cleanup failed", zap.Error(err))
			}
		}
	}()

	accessTTL, err := time.ParseDuration(cfg.JWTAccessExpiry)
	if err != nil {
		accessTTL = time.Hour
	}
	jwtMgr, err := auth.NewJWTManager(cfg.JWTPrivateKeyPath, cfg.JWTPublicKeyPath, accessTTL)
	if err != nil {
		logger.Fatal("failed to init JWT manager", zap.Error(err))
	}
	// Wire AT revocation — Verify() will reject explicitly-revoked tokens.
	jwtMgr.WithRevoker(revoker)

	minioClient, err := storage.NewMinIO(cfg.StorageEndpoint, cfg.StorageAccessKey, cfg.StorageSecretKey, cfg.StorageBucket, cfg.StorageUseSSL)
	if err != nil {
		logger.Fatal("failed to init MinIO client", zap.Error(err))
	}
	if err := minioClient.EnsureBucket(ctx); err != nil {
		logger.Warn("failed to ensure MinIO bucket (will retry on first upload)", zap.Error(err))
	}
	logger.Info("storage client initialized")

	ldapCfg := &auth.LDAPConfig{
		URL:          cfg.LDAPUrl,
		BaseDN:       cfg.LDAPBaseDN,
		UserAttr:     cfg.LDAPUserAttr,
		BindDN:       cfg.LDAPBindDN,
		BindPassword: cfg.LDAPBindPassword,
	}
	if ldapCfg.Enabled() {
		logger.Info("LDAP authentication enabled", zap.String("url", cfg.LDAPUrl))
	}

	mailCfg := mail.Config{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		User:     cfg.SMTPUser,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		TLS:      cfg.SMTPTLS,
	}
	pwdPolicy := auth.PasswordPolicy{
		MinLength:    cfg.PasswordMinLength,
		ExpiryDays:   cfg.PasswordExpiryDays,
		HistoryCount: cfg.PasswordHistoryN,
		RequireUpper: cfg.PasswordUppercase,
		RequireNumber: cfg.PasswordNumber,
	}
	if pwdPolicy.MinLength == 0 {
		pwdPolicy.MinLength = 8
	}

	srv := server.New(
		ctx, logger, jwtMgr, queries, sqlDB, minioClient,
		cfg.AIServiceURL, cfg.AIServiceSecret,
		ldapCfg, cfg.SIEMWebhookURL, cfg.SIEMFormat,
		mailCfg, cfg.MaxSessionsPerUser, pwdPolicy,
	)

	// Wrap the router with the circuit breaker guard. /health and /ready bypass
	// the check so load-balancer probes still work during a DB outage.
	cbHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cb.IsOpen() && r.URL.Path != "/health" && r.URL.Path != "/ready" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"code":"SERVICE_UNAVAILABLE","message":"database unavailable, retry shortly"}}`))
			return
		}
		srv.ServeHTTP(w, r)
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      cbHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", zap.String("addr", httpServer.Addr))
		serverErr <- httpServer.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	case sig := <-quit:
		logger.Info("shutdown signal received", zap.String("signal", sig.String()))
	}

	logger.Info("shutting down gracefully...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
		os.Exit(1)
	}
	// Drain buffered audit events so no log entries are lost on clean shutdown.
	srv.DrainAuditChain()
	// Wait up to 15 seconds for in-flight document processing goroutines.
	srv.DrainDocumentProcessing()
	// Drain background worker pool after all other drains so in-flight jobs finish.
	worker.Background.Stop()
	logger.Info("server stopped cleanly")
}
