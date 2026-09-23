package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"vps-billing/internal/config"
	"vps-billing/internal/database"
	domainCommerce "vps-billing/internal/domain/commerce"
	domainIdentity "vps-billing/internal/domain/identity"
	domainInfrastructure "vps-billing/internal/domain/infrastructure"
	"vps-billing/internal/logger"
	"vps-billing/internal/provider"
	"vps-billing/internal/provider/mock"
	backendRedis "vps-billing/internal/redis"
	"vps-billing/internal/repository"
	"vps-billing/internal/server"
	serviceAudit "vps-billing/internal/service/audit"
	serviceCommerce "vps-billing/internal/service/commerce"
	serviceFulfillment "vps-billing/internal/service/fulfillment"
	serviceIdentity "vps-billing/internal/service/identity"
	serviceOperation "vps-billing/internal/service/operation"
	"vps-billing/internal/service/scheduler"
	"vps-billing/internal/service/session"
	serviceSubscription "vps-billing/internal/service/subscription"
	"vps-billing/internal/workflow"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel)

	slog.Info("starting vps billing server",
		slog.String("env", cfg.AppEnv),
		slog.String("port", cfg.HTTPPort),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to PostgreSQL
	var dbPool *pgxpool.Pool
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Warn("could not connect to postgresql at startup (readiness will report unhealthy)",
			slog.String("error", err.Error()),
		)
	} else {
		dbPool = pool
		defer dbPool.Close()
		slog.Info("connected to postgresql successfully")

		if cfg.AutoMigrate {
			if err := database.RunMigrations(ctx, dbPool); err != nil {
				slog.Error("failed to run database migrations", slog.String("error", err.Error()))
			}
		}
	}

	// Connect to Redis
	var redisClient *redis.Client
	rClient, err := backendRedis.Connect(ctx, cfg.RedisURL)
	if err != nil {
		slog.Warn("could not connect to redis at startup (readiness will report unhealthy)",
			slog.String("error", err.Error()),
		)
	} else {
		redisClient = rClient
		defer redisClient.Close()
		slog.Info("connected to redis successfully")
	}

	// Session Store
	var sessionStore session.Store
	if redisClient != nil {
		sessionStore = session.NewRedisSessionStore(redisClient)
	} else {
		slog.Warn("using in-memory session store (sessions will not persist across restarts)")
		sessionStore = session.NewMemorySessionStore()
	}
	sessMgr := session.NewManager(sessionStore, cfg.AppEnv == "production")

	// Repositories & Services
	var userSvc *serviceIdentity.UserService
	var adminSvc *serviceIdentity.AdminService
	var auditSvc *serviceAudit.Service
	var productSvc *serviceCommerce.ProductService
	var orderSvc *serviceCommerce.OrderService
	var paymentSvc *serviceCommerce.PaymentService
	var walletSvc *serviceCommerce.WalletService
	var subSvc *serviceSubscription.SubscriptionService
	var opSvc *serviceOperation.Service
	var infraRepo domainInfrastructure.InfrastructureRepository

	if dbPool != nil {
		queries := repository.New(dbPool)
		userRepo := repository.NewPostgresUserRepository(queries)
		adminRepo := repository.NewPostgresAdminRepository(queries)
		rbacRepo := repository.NewPostgresRBACRepository(queries)
		auditRepo := repository.NewPostgresAuditRepository(queries)
		commerceRepo := repository.NewPostgresCommerceRepository(dbPool, queries)
		subRepo := repository.NewPostgresSubscriptionRepository(queries)
		opRepo := repository.NewPostgresOperationRepository(queries)
		infraRepo = repository.NewPostgresInfrastructureRepository(dbPool, queries)

		// Seed initial roles & permissions
		if err := rbacRepo.SeedInitialRolesAndPermissions(ctx); err != nil {
			slog.Error("failed to seed rbac roles and permissions", slog.String("error", err.Error()))
		}

		auditSvc = serviceAudit.NewService(auditRepo)
		userSvc = serviceIdentity.NewUserService(userRepo, auditSvc, sessMgr)
		adminSvc = serviceIdentity.NewAdminService(adminRepo, rbacRepo, auditSvc, sessMgr)

		fakeGateway := serviceCommerce.NewFakePaymentGateway("")
		productSvc = serviceCommerce.NewProductService(commerceRepo)
		orderSvc = serviceCommerce.NewOrderService(commerceRepo, commerceRepo)
		paymentSvc = serviceCommerce.NewPaymentService(commerceRepo, commerceRepo, fakeGateway)
		walletSvc = serviceCommerce.NewWalletService(commerceRepo, commerceRepo)
		subSvc = serviceSubscription.NewSubscriptionService(subRepo, commerceRepo)
		opSvc = serviceOperation.NewService(opRepo, redisClient)

		mockProv := mock.NewMockProvider("mock-default")
		providersMap := map[string]provider.Provider{
			"default": mockProv,
		}
		sched := scheduler.NewScheduler(infraRepo)
		provWf := workflow.NewProvisionWorkflow(infraRepo, subRepo, commerceRepo, sched, providersMap, opSvc)
		fulfillSvc := serviceFulfillment.NewFulfillmentService(commerceRepo, subRepo, commerceRepo, opSvc, provWf)
		paymentSvc.SetFulfiller(fulfillSvc)

		// Seed sample provider and node if none exist
		if existingNodes, err := infraRepo.ListActiveNodes(ctx); err == nil && len(existingNodes) == 0 {
			provID := uuid.New()
			_, _ = infraRepo.CreateProvider(ctx, &domainInfrastructure.Provider{
				ID:           provID,
				Name:         "Default Mock Provider",
				ProviderType: "mock",
				Status:       "active",
				Config:       []byte("{}"),
				Capabilities: []byte(`{"create_instance":true}`),
			})
			providersMap[provID.String()] = mockProv
			_, _ = infraRepo.CreateNode(ctx, &domainInfrastructure.Node{
				ID:            uuid.New(),
				ProviderID:    provID,
				Name:          "node-us-west-1",
				Region:        "us-west",
				Status:        "active",
				CPUTotal:      64,
				MemoryTotalMB: 262144,
				DiskTotalGB:   4000,
				Weight:        100,
				Capabilities:  []byte("{}"),
			})
			slog.Info("seeded initial sample provider and node")
		}

		// Seed sample product and plans if none exist
		if existingProds, err := productSvc.ListActiveProducts(ctx); err == nil && len(existingProds) == 0 {
			prodID := uuid.New()
			nameI18n, _ := json.Marshal(map[string]string{"en-US": "Standard Cloud VPS", "zh-CN": "标准云服务器"})
			descI18n, _ := json.Marshal(map[string]string{"en-US": "High performance cloud VPS with NVMe storage", "zh-CN": "高性能 NVMe 云服务器"})
			_, err := productSvc.CreateProduct(ctx, &domainCommerce.Product{
				ID:              prodID,
				Slug:            "standard-vps",
				NameI18n:        nameI18n,
				DescriptionI18n: descI18n,
				Status:          "active",
				SortOrder:       1,
			})
			if err == nil {
				traffic1 := int64(1000)
				bw1 := 100
				plan1Name, _ := json.Marshal(map[string]string{"en-US": "Starter VPS (1C/1G)", "zh-CN": "入门型 (1核/1G)"})
				_, _ = productSvc.CreatePlan(ctx, &domainCommerce.Plan{
					ID:             uuid.New(),
					ProductID:      prodID,
					Slug:           "starter",
					NameI18n:       plan1Name,
					Status:         "active",
					CPUCores:       1,
					MemoryMB:       1024,
					DiskGB:         25,
					TrafficGB:      &traffic1,
					BandwidthMbps:  &bw1,
					IPv4Count:      1,
					IPv6Count:      1,
					NatPortCount:   0,
					Virtualization: "kvm",
					BillingCycle:   "monthly",
					PriceMinor:     500, // $5.00
					Currency:       "USD",
					StockMode:      "automatic",
				})

				traffic2 := int64(2000)
				bw2 := 200
				plan2Name, _ := json.Marshal(map[string]string{"en-US": "Pro VPS (2C/2G)", "zh-CN": "专业型 (2核/2G)"})
				_, _ = productSvc.CreatePlan(ctx, &domainCommerce.Plan{
					ID:             uuid.New(),
					ProductID:      prodID,
					Slug:           "pro",
					NameI18n:       plan2Name,
					Status:         "active",
					CPUCores:       2,
					MemoryMB:       2048,
					DiskGB:         50,
					TrafficGB:      &traffic2,
					BandwidthMbps:  &bw2,
					IPv4Count:      1,
					IPv6Count:      1,
					NatPortCount:   0,
					Virtualization: "kvm",
					BillingCycle:   "monthly",
					PriceMinor:     1000, // $10.00
					Currency:       "USD",
					StockMode:      "automatic",
				})
				slog.Info("seeded initial sample product and plans")
			}
		}

		// Seed default super admin if none exists
		initAdminEmail := cfg.InitialAdminEmail
		initAdminPass := cfg.InitialAdminPassword
		if admin, err := adminSvc.CreateInitialAdmin(ctx, initAdminEmail, initAdminPass, "Super Admin", domainIdentity.RoleSuperAdmin); err == nil && admin != nil {
			slog.Info("default super admin initialized", slog.String("email", initAdminEmail))
		}
	}

	// Setup Router
	router := server.NewRouterWithDeps(server.RouterDeps{
		Config:          cfg,
		DB:              dbPool,
		Redis:           redisClient,
		UserSvc:         userSvc,
		AdminSvc:        adminSvc,
		AuditSvc:        auditSvc,
		SessionMgr:      sessMgr,
		ProductSvc:      productSvc,
		OrderSvc:        orderSvc,
		PaymentSvc:      paymentSvc,
		WalletSvc:       walletSvc,
		SubscriptionSvc: subSvc,
		OperationSvc:    opSvc,
		InfraRepo:       infraRepo,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown channel
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("server listening", slog.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	sig := <-shutdownChan
	slog.Info("received shutdown signal, terminating gracefully", slog.String("signal", sig.String()))

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", slog.String("error", err.Error()))
	}

	slog.Info("server exited cleanly")
}
