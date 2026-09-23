package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"vps-billing/internal/config"
	domainIdentity "vps-billing/internal/domain/identity"
	domainInfrastructure "vps-billing/internal/domain/infrastructure"
	"vps-billing/internal/handler"
	"vps-billing/internal/middleware"
	serviceAudit "vps-billing/internal/service/audit"
	serviceCommerce "vps-billing/internal/service/commerce"
	serviceIdentity "vps-billing/internal/service/identity"
	serviceOperation "vps-billing/internal/service/operation"
	"vps-billing/internal/service/session"
	serviceSubscription "vps-billing/internal/service/subscription"
	serviceTicket "vps-billing/internal/service/ticket"
)

type RouterDeps struct {
	Config     *config.Config
	DB         *pgxpool.Pool
	Redis      *redis.Client
	UserSvc    *serviceIdentity.UserService
	AdminSvc   *serviceIdentity.AdminService
	AuditSvc   *serviceAudit.Service
	SessionMgr *session.Manager

	// Commerce & Infra services
	ProductSvc      *serviceCommerce.ProductService
	OrderSvc        *serviceCommerce.OrderService
	PaymentSvc      *serviceCommerce.PaymentService
	WalletSvc       *serviceCommerce.WalletService
	SubscriptionSvc *serviceSubscription.SubscriptionService
	OperationSvc    *serviceOperation.Service
	InfraRepo       domainInfrastructure.InfrastructureRepository
	TicketSvc       *serviceTicket.Service
}

func NewRouterWithDeps(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	// Global middlewares
	r.Use(middleware.RequestID)
	r.Use(TrackRequestMetric)
	r.Use(middleware.Logging)
	r.Use(middleware.Recovery)
	r.Use(middleware.CORS([]string{
		deps.Config.UserWebOrigin,
		deps.Config.AdminWebOrigin,
		"http://localhost:3000",
		"http://localhost:3001",
	}))
	r.Use(chiMiddleware.RealIP)

	healthHandler := NewHealthHandler(deps.DB, deps.Redis)
	metricsHandler := NewMetricsHandler(deps.DB, deps.Redis)

	// Direct health and metrics endpoints
	r.Get("/health/live", healthHandler.Live)
	r.Get("/health/ready", healthHandler.Ready)
	r.Get("/metrics", metricsHandler.Metrics)

	// Rate limiter (Redis if available, memory fallback)
	var rateLimiter middleware.RateLimiter
	if deps.Redis != nil {
		rateLimiter = middleware.NewRedisRateLimiter(deps.Redis)
	} else {
		rateLimiter = middleware.NewMemoryRateLimiter()
	}

	// Handlers initialization
	var (
		userAuthHandler      *handler.UserAuthHandler
		adminAuthHandler     *handler.AdminAuthHandler
		adminAuditHandler    *handler.AdminAuditHandler
		productHandler       *handler.ProductHandler
		orderHandler         *handler.OrderHandler
		paymentHandler       *handler.PaymentHandler
		walletInvoiceHandler *handler.WalletInvoiceHandler
		adminCommerceHandler *handler.AdminCommerceHandler
		subHandler           *handler.SubscriptionHandler
		operationHandler     *handler.OperationHandler
		instanceHandler      *handler.InstanceHandler
		adminInfraHandler    *handler.AdminInfrastructureHandler
		ticketHandler        *handler.TicketHandler
	)

	if deps.TicketSvc != nil {
		ticketHandler = handler.NewTicketHandler(deps.TicketSvc)
	}

	if deps.OperationSvc != nil {
		operationHandler = handler.NewOperationHandler(deps.OperationSvc)
	}

	if deps.InfraRepo != nil && deps.OperationSvc != nil {
		adminInfraHandler = handler.NewAdminInfrastructureHandler(deps.InfraRepo, deps.OperationSvc, deps.UserSvc, deps.AdminSvc)
	}

	if deps.InfraRepo != nil && deps.SubscriptionSvc != nil && deps.OperationSvc != nil {
		instanceHandler = handler.NewInstanceHandler(deps.InfraRepo, deps.SubscriptionSvc, deps.OperationSvc)
	}

	if deps.SubscriptionSvc != nil {
		subHandler = handler.NewSubscriptionHandler(deps.SubscriptionSvc)
	}

	if deps.UserSvc != nil && deps.SessionMgr != nil {
		userAuthHandler = handler.NewUserAuthHandler(deps.UserSvc, deps.SessionMgr)
	}
	if deps.AdminSvc != nil && deps.SessionMgr != nil {
		adminAuthHandler = handler.NewAdminAuthHandler(deps.AdminSvc, deps.SessionMgr)
	}
	if deps.AuditSvc != nil {
		adminAuditHandler = handler.NewAdminAuditHandler(deps.AuditSvc)
	}

	if deps.ProductSvc != nil {
		productHandler = handler.NewProductHandler(deps.ProductSvc)
	}
	if deps.OrderSvc != nil && deps.PaymentSvc != nil {
		orderHandler = handler.NewOrderHandler(deps.OrderSvc, deps.PaymentSvc)
	}
	if deps.PaymentSvc != nil {
		paymentHandler = handler.NewPaymentHandler(deps.PaymentSvc)
	}
	if deps.WalletSvc != nil {
		walletInvoiceHandler = handler.NewWalletInvoiceHandler(deps.WalletSvc)
	}
	if deps.ProductSvc != nil && deps.OrderSvc != nil && deps.WalletSvc != nil {
		adminCommerceHandler = handler.NewAdminCommerceHandler(deps.ProductSvc, deps.OrderSvc, deps.WalletSvc)
	}

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health/live", healthHandler.Live)
		r.Get("/health/ready", healthHandler.Ready)
		r.Get("/metrics", metricsHandler.Metrics)

		// Operations (SSE & Polling)
		if operationHandler != nil {
			r.Get("/operations/{id}", operationHandler.Get)
			r.Get("/operations/{id}/events", operationHandler.Events)
		}

		// Public Products
		if productHandler != nil {
			r.Get("/products", productHandler.List)
			r.Get("/products/{id}", productHandler.Get)
		}

		// Public Payment Webhooks & Simulation
		if paymentHandler != nil {
			r.Post("/payments/webhook/{gateway}", paymentHandler.HandleWebhook)
			r.Post("/payments/fake/simulate", paymentHandler.SimulateFakePayment)
		}

		// Public User Auth (Rate-limited)
		if userAuthHandler != nil {
			r.With(middleware.RateLimit(rateLimiter, 10, time.Minute)).Post("/auth/register", userAuthHandler.Register)
			r.With(middleware.RateLimit(rateLimiter, 10, time.Minute)).Post("/auth/login", userAuthHandler.Login)
		}

		// Protected User Area
		if deps.SessionMgr != nil && deps.UserSvc != nil {
			r.Group(func(r chi.Router) {
				r.Use(middleware.UserAuth(deps.SessionMgr, deps.UserSvc))
				r.Use(middleware.CSRF)

				if userAuthHandler != nil {
					r.Get("/auth/me", userAuthHandler.GetMe)
					r.Post("/auth/logout", userAuthHandler.Logout)
				}

				// Orders
				if orderHandler != nil {
					r.Post("/orders", orderHandler.Create)
					r.Get("/orders", orderHandler.List)
					r.Get("/orders/{id}", orderHandler.Get)
					r.Post("/orders/{id}/pay", orderHandler.Pay)
					r.Post("/orders/{id}/cancel", orderHandler.Cancel)
				}

				// Wallet & Invoices
				if walletInvoiceHandler != nil {
					r.Get("/wallet", walletInvoiceHandler.GetWallet)
					r.Post("/wallet/deposit", walletInvoiceHandler.Deposit)
					r.Get("/wallet/ledger", walletInvoiceHandler.ListUserLedger)
					r.Get("/invoices", walletInvoiceHandler.ListInvoices)
					r.Get("/invoices/{id}", walletInvoiceHandler.GetInvoice)
				}

				// Tickets
				if ticketHandler != nil {
					r.Get("/tickets", ticketHandler.ListMyTickets)
					r.Post("/tickets", ticketHandler.CreateTicket)
					r.Get("/tickets/{id}", ticketHandler.GetTicket)
					r.Post("/tickets/{id}/reply", ticketHandler.UserReplyTicket)
				}

				// Subscriptions
				if subHandler != nil {
					r.Get("/subscriptions", subHandler.List)
					r.Get("/subscriptions/{id}", subHandler.Get)
					r.Post("/subscriptions/{id}/cancel", subHandler.Cancel)
				}

				// Instances
				if instanceHandler != nil {
					r.Get("/instances", instanceHandler.List)
					r.Get("/instances/{id}", instanceHandler.Get)
					r.Post("/instances/{id}/start", instanceHandler.Start)
					r.Post("/instances/{id}/stop", instanceHandler.Stop)
					r.Post("/instances/{id}/restart", instanceHandler.Restart)
					r.Post("/instances/{id}/reinstall", instanceHandler.Reinstall)
				}
			})
		}

		// Admin Area
		r.Route("/admin", func(r chi.Router) {
			if adminAuthHandler != nil {
				// Public Admin Login (Rate-limited)
				r.With(middleware.RateLimit(rateLimiter, 5, time.Minute)).Post("/auth/login", adminAuthHandler.Login)

				// Protected Admin Area
				r.Group(func(r chi.Router) {
					r.Use(middleware.AdminAuth(deps.SessionMgr, deps.AdminSvc))
					r.Use(middleware.CSRF)

					r.Get("/auth/me", adminAuthHandler.GetMe)
					r.Post("/auth/logout", adminAuthHandler.Logout)
					r.Get("/auth/2fa/setup", adminAuthHandler.Setup2FA)
					r.Post("/auth/2fa/enable", adminAuthHandler.Enable2FA)
					r.Post("/auth/2fa/disable", adminAuthHandler.Disable2FA)

					if adminAuditHandler != nil {
						r.With(middleware.RequirePermission(deps.AdminSvc, domainIdentity.PermAuditRead)).
							Get("/audit", adminAuditHandler.List)
					}

					if adminCommerceHandler != nil {
						r.Get("/products", adminCommerceHandler.ListProducts)
						r.Post("/products", adminCommerceHandler.CreateProduct)
						r.Post("/plans", adminCommerceHandler.CreatePlan)
						r.Get("/orders", adminCommerceHandler.ListOrders)
						r.Get("/invoices", adminCommerceHandler.ListInvoices)
						r.Get("/ledger", adminCommerceHandler.ListLedger)
					}

					if subHandler != nil {
						r.Get("/subscriptions", subHandler.AdminList)
					}

					if adminInfraHandler != nil {
						r.Get("/providers", adminInfraHandler.ListProviders)
						r.Post("/providers", adminInfraHandler.CreateProvider)
						r.Get("/nodes", adminInfraHandler.ListNodes)
						r.Post("/nodes", adminInfraHandler.CreateNode)
						r.Get("/instances", adminInfraHandler.ListInstances)
						r.Get("/operations", adminInfraHandler.ListOperations)
						r.Get("/operations/{id}", adminInfraHandler.GetOperation)
						r.Get("/users", adminInfraHandler.ListUsers)
						r.Get("/admins", adminInfraHandler.ListAdmins)
					}

					if ticketHandler != nil {
						r.Get("/tickets", ticketHandler.AdminListTickets)
						r.Get("/tickets/{id}", ticketHandler.GetTicket)
						r.Post("/tickets/{id}/reply", ticketHandler.AdminReplyTicket)
						r.Post("/tickets/{id}/status", ticketHandler.AdminUpdateTicketStatus)
					}

				})
			}
		})

		// 404 handler within api/v1
		r.NotFound(func(w http.ResponseWriter, req *http.Request) {
			Error(w, req, http.StatusNotFound, "NOT_FOUND", "errors.not_found", "endpoint not found")
		})
	})

	// Fallback 404
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		Error(w, req, http.StatusNotFound, "NOT_FOUND", "errors.not_found", "resource not found")
	})

	return r
}

func NewRouter(cfg *config.Config, db *pgxpool.Pool, redisClient *redis.Client) http.Handler {
	return NewRouterWithDeps(RouterDeps{
		Config: cfg,
		DB:     db,
		Redis:  redisClient,
	})
}
