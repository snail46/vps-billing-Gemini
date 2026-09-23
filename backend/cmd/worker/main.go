package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"vps-billing/internal/config"
	"vps-billing/internal/database"
	domainOperation "vps-billing/internal/domain/operation"
	"vps-billing/internal/logger"
	"vps-billing/internal/provider"
	"vps-billing/internal/provider/mock"
	"vps-billing/internal/reconciler"
	backendRedis "vps-billing/internal/redis"
	"vps-billing/internal/repository"
	serviceOperation "vps-billing/internal/service/operation"
	"vps-billing/internal/service/scheduler"
	"vps-billing/internal/worker"
	"vps-billing/internal/workflow"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel)

	slog.Info("starting vps billing worker",
		slog.String("env", cfg.AppEnv),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to PostgreSQL with resilient retry (up to 25 attempts, 2s interval)
	var dbPool *pgxpool.Pool
	pool, err := database.ConnectWithRetry(ctx, cfg.DatabaseURL, 25, 2*time.Second)
	if err != nil {
		slog.Error("worker could not connect to postgresql after startup retries",
			slog.String("error", err.Error()),
		)
	} else {
		dbPool = pool
		defer dbPool.Close()
		slog.Info("worker connected to postgresql")
	}

	// Connect to Redis with resilient retry (up to 25 attempts, 2s interval)
	var redisClient *redis.Client
	rClient, err := backendRedis.ConnectWithRetry(ctx, cfg.RedisURL, 25, 2*time.Second)
	if err != nil {
		slog.Error("worker could not connect to redis after startup retries",
			slog.String("error", err.Error()),
		)
	} else {
		redisClient = rClient
		defer redisClient.Close()
		slog.Info("worker connected to redis")
	}

	var opRepo domainOperation.OperationRepository
	var opSvc *serviceOperation.Service
	registry := worker.NewWorkflowRegistry()

	if dbPool != nil {
		queries := repository.New(dbPool)
		opRepo = repository.NewPostgresOperationRepository(queries)
		opSvc = serviceOperation.NewService(opRepo, redisClient)
		infraRepo := repository.NewPostgresInfrastructureRepository(dbPool, queries)
		subRepo := repository.NewPostgresSubscriptionRepository(queries)
		commerceRepo := repository.NewPostgresCommerceRepository(dbPool, queries)
		sched := scheduler.NewScheduler(infraRepo)
		mockProv := mock.NewMockProvider("mock-default")
		providersMap := map[string]provider.Provider{
			"default": mockProv,
		}
		provWf := workflow.NewProvisionWorkflow(infraRepo, subRepo, commerceRepo, sched, providersMap, opSvc)
		registry.Register(provWf)

		registry.Register(workflow.NewInstanceLifecycleWorkflow("start_instance", infraRepo, providersMap, opSvc))
		registry.Register(workflow.NewInstanceLifecycleWorkflow("stop_instance", infraRepo, providersMap, opSvc))
		registry.Register(workflow.NewInstanceLifecycleWorkflow("restart_instance", infraRepo, providersMap, opSvc))
		registry.Register(workflow.NewInstanceLifecycleWorkflow("reinstall_instance", infraRepo, providersMap, opSvc))
		registry.Register(workflow.NewInstanceLifecycleWorkflow("reset_password", infraRepo, providersMap, opSvc))
		registry.Register(workflow.NewInstanceLifecycleWorkflow("delete_instance", infraRepo, providersMap, opSvc))

		rec := reconciler.NewReconciler(infraRepo, opRepo, opSvc, providersMap, 3*time.Minute, 1*time.Minute)
		go func() {
			recTicker := time.NewTicker(30 * time.Second)
			defer recTicker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-recTicker.C:
					rec.RunOnce(ctx)
				}
			}
		}()
	}

	w := worker.New(cfg, dbPool, redisClient, opRepo, opSvc, registry)

	// Graceful shutdown handling
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-shutdownChan
		slog.Info("worker received shutdown signal", slog.String("signal", sig.String()))
		cancel()
	}()

	if err := w.Start(ctx); err != nil {
		slog.Error("worker terminated with error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("worker exited cleanly")
}
