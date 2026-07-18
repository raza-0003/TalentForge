// Command worker runs the background job processor (Asynq/Redis): it delivers
// queued emails and interview reminders, and periodically sweeps the outbox for
// stuck notifications.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/faizan/ats/internal/config"
	"github.com/faizan/ats/internal/database"
	"github.com/faizan/ats/internal/logger"
	"github.com/faizan/ats/internal/mailer"
	"github.com/faizan/ats/internal/repository"
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

	db, err := database.NewPostgres(context.Background(), cfg.Postgres.DSN())
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer db.Close()
	repos := repository.New(db)

	// Choose the mailer: real SMTP when configured, otherwise log-only so the
	// worker runs without a mail server.
	var m mailer.Mailer
	if cfg.SMTP.Host == "" {
		m = mailer.NewLogMailer(log)
		log.Warn("SMTP host not configured — emails will be logged, not sent")
	} else {
		m = mailer.NewSMTPMailer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
		log.Info("SMTP mailer configured", zap.String("host", cfg.SMTP.Host))
	}

	store, err := storage.New(context.Background(), cfg.Storage.Driver, cfg.Storage.Dir,
		cfg.Storage.Bucket, cfg.Storage.Region, cfg.Storage.Endpoint)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}

	sweepAfter := time.Duration(cfg.Worker.SweepAfterMin) * time.Minute
	handlers := tasks.NewHandlers(m, repos.Notifications, repos.Interviews, repos.Users, repos.Resumes, store, sweepAfter, log)

	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	// Server: processes tasks off the queue.
	srv := asynq.NewServer(redisOpt, asynq.Config{Concurrency: cfg.Worker.Concurrency})
	mux := asynq.NewServeMux()
	handlers.Register(mux)

	// Scheduler: enqueues the sweep task on a cron interval. A fixed TaskID makes
	// each run a singleton, so overlapping sweeps can't pile up.
	scheduler := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{})
	if _, err := scheduler.Register(cfg.Worker.SweepCron, tasks.NewSweepPendingTask(),
		asynq.TaskID("sweep-pending")); err != nil {
		return fmt.Errorf("register sweep schedule: %w", err)
	}

	if err := srv.Start(mux); err != nil {
		return fmt.Errorf("start worker server: %w", err)
	}
	if err := scheduler.Start(); err != nil {
		srv.Shutdown()
		return fmt.Errorf("start scheduler: %w", err)
	}
	log.Info("worker running",
		zap.Int("concurrency", cfg.Worker.Concurrency),
		zap.String("sweep_cron", cfg.Worker.SweepCron),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("shutdown signal received", zap.String("signal", sig.String()))

	scheduler.Shutdown()
	srv.Shutdown()
	log.Info("worker stopped cleanly")
	return nil
}
