package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/config"
	"github.com/reddybashanaguru/opsowl/internal/db"
	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/server"
	"github.com/reddybashanaguru/opsowl/internal/storage"
)

func main() {
	var logger *zap.Logger
	if os.Getenv("ENVIRONMENT") == "development" {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}
	defer logger.Sync() //nolint:errcheck

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

	sqlDB := stdlib.OpenDBFromPool(pool)
	queries := dbgen.New(sqlDB)

	accessTTL, err := time.ParseDuration(cfg.JWTAccessExpiry)
	if err != nil {
		accessTTL = time.Hour
	}
	jwtMgr, err := auth.NewJWTManager(cfg.JWTPrivateKeyPath, cfg.JWTPublicKeyPath, accessTTL)
	if err != nil {
		logger.Fatal("failed to init JWT manager", zap.Error(err))
	}

	minioClient, err := storage.NewMinIO(cfg.StorageEndpoint, cfg.StorageAccessKey, cfg.StorageSecretKey, cfg.StorageBucket, cfg.StorageUseSSL)
	if err != nil {
		logger.Fatal("failed to init MinIO client", zap.Error(err))
	}
	if err := minioClient.EnsureBucket(ctx); err != nil {
		logger.Warn("failed to ensure MinIO bucket (will retry on first upload)", zap.Error(err))
	}
	logger.Info("storage client initialized")

	srv := server.New(logger, jwtMgr, queries, minioClient, cfg.AIServiceURL)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      srv,
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
	logger.Info("server stopped cleanly")
}
