package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CaioWing/Harbor/internal/api"
	"github.com/CaioWing/Harbor/internal/auth"
	"github.com/CaioWing/Harbor/internal/config"
	"github.com/CaioWing/Harbor/internal/repository/postgres"
	"github.com/CaioWing/Harbor/internal/service"
	"github.com/CaioWing/Harbor/internal/storage/local"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log.Info("starting Harbor",
		"listen", cfg.ListenAddr(),
		"db_host", cfg.DB.Host,
		"storage", cfg.Storage.Path,
	)

	// Run migrations
	log.Info("running database migrations")
	if err := postgres.RunMigrations(cfg.DB.DSN()); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	log.Info("migrations completed")

	// Database connection pool
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DB.DSN())
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	log.Info("database connected")

	// File storage
	store, err := local.New(cfg.Storage.Path)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	log.Info("storage initialized", "path", cfg.Storage.Path)

	// Repositories
	deviceRepo := postgres.NewDeviceRepo(pool)
	artifactRepo := postgres.NewArtifactRepo(pool)
	deploymentRepo := postgres.NewDeploymentRepo(pool)

	// Services
	deviceSvc := service.NewDeviceService(deviceRepo, log)
	artifactSvc := service.NewArtifactService(artifactRepo, store, log)
	deploymentSvc := service.NewDeploymentService(deploymentRepo, deviceRepo, artifactRepo, log)

	// Auth
	jwtMgr := auth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry)

	// Router
	router := api.NewRouter(api.RouterDeps{
		DeviceSvc:     deviceSvc,
		ArtifactSvc:   artifactSvc,
		DeploymentSvc: deploymentSvc,
		JWTManager:    jwtMgr,
		CORSOrigins:   cfg.CORS.AllowedOrigins,
		Logger:        log,
	})

	// HTTP Server
	srv := &http.Server{
		Addr:         cfg.ListenAddr(),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		log.Info("server listening", "addr", cfg.ListenAddr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info("shutting down", "signal", sig)
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	log.Info("server stopped")
	return nil
}
