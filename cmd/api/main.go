// Command api is the ATS HTTP API server.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/faizan/ats/internal/auth"
	"github.com/faizan/ats/internal/config"
	"github.com/faizan/ats/internal/database"
	"github.com/faizan/ats/internal/handler"
	"github.com/faizan/ats/internal/logger"
	"github.com/faizan/ats/internal/middleware"
	"github.com/faizan/ats/internal/repository"
	"github.com/faizan/ats/internal/service"
	"github.com/faizan/ats/internal/storage"
	"github.com/faizan/ats/internal/tasks"
)
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log, err := logger.New(cfg.Env, cfg.Log.Level)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer func() { _ = log.Sync() }()

	ctx := context.Background()

	db, err := database.NewPostgres(ctx, cfg.Postgres.DSN())
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer db.Close()
	log.Info("connected to postgres")

	rdb, err := database.NewRedis(ctx, cfg.Redis.Addr(), cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return fmt.Errorf("connect redis: %w", err)
	}
	defer func() { _ = rdb.Close() }()
	log.Info("connected to redis")

	// --- Wiring: repositories -> services -> handlers ---
	repos := repository.New(db)

	tm := auth.NewTokenManager(
		cfg.JWT.Secret,
		time.Duration(cfg.JWT.AccessTTLMin)*time.Minute,
		time.Duration(cfg.JWT.RefreshTTLHours)*time.Hour,
	)
	// The enqueuer persists notifications and pushes email/reminder tasks onto
	// Redis for the worker; it satisfies both Notifier and ReminderScheduler.
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer func() { _ = asynqClient.Close() }()
	enqueuer := tasks.NewEnqueuer(asynqClient, repos.Notifications, log)

	store, err := storage.New(ctx, cfg.Storage.Driver, cfg.Storage.Dir,
		cfg.Storage.Bucket, cfg.Storage.Region, cfg.Storage.Endpoint)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}

	authSvc := service.NewAuthService(repos.Users, repos.RefreshTokens, tm)
	candSvc := service.NewCandidateService(repos.Profiles)
	jobSvc := service.NewJobService(repos.Jobs)
	appSvc := service.NewApplicationService(repos.Applications, repos.Jobs, repos.Users, repos.Resumes, enqueuer)
	ivSvc := service.NewInterviewService(repos.Interviews, repos.Feedback, repos.Applications, repos.Users, enqueuer, enqueuer)
	resumeSvc := service.NewResumeService(repos.Resumes, store, enqueuer)
	offerSvc := service.NewOfferService(repos.Offers, repos.Applications, repos.Users, repos.Jobs, store, enqueuer)
	adminSvc := service.NewAdminService(repos.Users, repos.Jobs, repos.Applications)

	handlers := &handler.Handlers{
		Health:      handler.NewHealthHandler(db, rdb),
		Docs:        handler.NewDocsHandler(),
		Auth:        handler.NewAuthHandler(authSvc),
		Candidate:   handler.NewCandidateHandler(candSvc, appSvc),
		Job:         handler.NewJobHandler(jobSvc),
		Application: handler.NewApplicationHandler(appSvc),
		Interview:   handler.NewInterviewHandler(ivSvc),
		Resume:      handler.NewResumeHandler(resumeSvc),
		Search:      handler.NewSearchHandler(candSvc),
		Offer:       handler.NewOfferHandler(offerSvc),
		Admin:       handler.NewAdminHandler(adminSvc),
	}

	if cfg.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(middleware.Logger(log), middleware.Recovery(log))
	router.Use(middleware.CORS(os.Getenv("ATS_CORS_ORIGINS")))
	handlers.Register(router, tm)

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		log.Info("shutdown signal received", zap.String("signal", sig.String()))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	log.Info("server stopped cleanly")
	return nil
}
