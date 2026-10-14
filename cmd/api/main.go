package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/haryfun/task-management-service/internal/config"
	"github.com/haryfun/task-management-service/internal/domain"
	"github.com/haryfun/task-management-service/internal/platform/jwt"
	"github.com/haryfun/task-management-service/internal/platform/logger"
	"github.com/haryfun/task-management-service/internal/platform/notifier"
	"github.com/haryfun/task-management-service/internal/platform/password"
	"github.com/haryfun/task-management-service/internal/repository/postgres"
	"github.com/haryfun/task-management-service/internal/transport/http/dto"
	"github.com/haryfun/task-management-service/internal/transport/http/handler"
	"github.com/haryfun/task-management-service/internal/transport/http/router"
	"github.com/haryfun/task-management-service/internal/usecase"
)

// @title           Task Management API
// @version         1.0
// @description     Multi-user task management API: JWT auth, per-user tasks, custom statuses, idempotent create, and assignment with an audit trail.
// @host            localhost:8080
// @BasePath        /v1
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Bootstrap logger so even early failures (e.g. config) are structured JSON.
	slog.SetDefault(logger.New("info"))

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(cfg.LogLevel)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, postgres.PoolConfig{
		MaxConns:        cfg.DBMaxConns,
		MinConns:        cfg.DBMinConns,
		MaxConnLifetime: cfg.DBMaxConnLifetime,
		MaxConnIdleTime: cfg.DBMaxConnIdleTime,
	})
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Info("database connected",
		"max_conns", cfg.DBMaxConns,
		"min_conns", cfg.DBMinConns,
		"max_conn_lifetime", cfg.DBMaxConnLifetime.String(),
		"max_conn_idle_time", cfg.DBMaxConnIdleTime.String(),
	)

	db := postgres.NewDB(pool, cfg.DBLockTimeout)
	userRepo := postgres.NewUserRepository(db)
	statusRepo := postgres.NewTaskStatusRepository(db)
	taskRepo := postgres.NewTaskRepository(db)
	teamRepo := postgres.NewTeamRepository(db)
	taskLogRepo := postgres.NewTaskLogRepository(db)
	idempotencyRepo := postgres.NewIdempotencyRepository(db)

	hasher := password.NewBcryptHasher()
	tokenIssuer := jwt.NewIssuer(cfg.JWTSecret, cfg.JWTExpiry)
	authUsecase := usecase.NewAuthUsecase(userRepo, statusRepo, db, hasher, tokenIssuer)
	statusUsecase := usecase.NewStatusUsecase(statusRepo, db)

	// The response encoder keeps the wire shape in the transport layer while
	// letting the use case persist a replayable response for idempotency.
	encodeTask := func(task *domain.Task) (int, []byte, error) {
		body, err := json.Marshal(dto.ToTaskResponse(task))
		return http.StatusCreated, body, err
	}
	taskUsecase := usecase.NewTaskUsecase(taskRepo, statusRepo, teamRepo, idempotencyRepo, taskLogRepo, notifier.NewMock(log), db, encodeTask)

	srv := &http.Server{
		Addr: ":" + cfg.AppPort,
		Handler: router.New(router.Deps{
			Logger:   log,
			Health:   handler.NewHealthHandler(pool),
			Auth:     handler.NewAuthHandler(authUsecase),
			Status:   handler.NewStatusHandler(statusUsecase),
			Tasks:    handler.NewTaskHandler(taskUsecase),
			Verifier: tokenIssuer,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", srv.Addr, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	log.Info("server stopped gracefully")
	return nil
}
