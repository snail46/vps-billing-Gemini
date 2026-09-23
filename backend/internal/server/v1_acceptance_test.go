package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"vps-billing/internal/config"
	domainCommerce "vps-billing/internal/domain/commerce"
	domainIdentity "vps-billing/internal/domain/identity"
	domainInfrastructure "vps-billing/internal/domain/infrastructure"
	domainOperation "vps-billing/internal/domain/operation"
	domainSubscription "vps-billing/internal/domain/subscription"
	"vps-billing/internal/reconciler"
	"vps-billing/internal/server"
	serviceAudit "vps-billing/internal/service/audit"
	serviceCommerce "vps-billing/internal/service/commerce"
	serviceIdentity "vps-billing/internal/service/identity"
	serviceOperation "vps-billing/internal/service/operation"
	"vps-billing/internal/service/session"
	serviceSubscription "vps-billing/internal/service/subscription"
)

// Acceptance Test 1: 100 Duplicate Payment Callbacks
// Invariant: Exactly 1 payment success, 1 order status update, 0 duplicate ledger charges.
func TestAcceptance100DuplicatePaymentWebhooks(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		UserWebOrigin:  "http://localhost:3000",
		AdminWebOrigin: "http://localhost:3001",
	}

	repo := newTestCommerceRepo()
	fakeGateway := serviceCommerce.NewFakePaymentGateway("secret_test")
	productSvc := serviceCommerce.NewProductService(repo)
	orderSvc := serviceCommerce.NewOrderService(repo, repo)
	paymentSvc := serviceCommerce.NewPaymentService(repo, repo, fakeGateway)
	walletSvc := serviceCommerce.NewWalletService(repo, repo)

	userID := uuid.New()
	prodID := uuid.New()
	_, err := productSvc.CreateProduct(ctx, &domainCommerce.Product{
		ID:        prodID,
		Slug:      "vps-cloud-acceptance",
		NameI18n:  json.RawMessage(`{"en-US":"Cloud VPS"}`),
		Status:    "active",
		SortOrder: 1,
	})
	require.NoError(t, err)

	planID := uuid.New()
	_, err = productSvc.CreatePlan(ctx, &domainCommerce.Plan{
		ID:             planID,
		ProductID:      prodID,
		Slug:           "1c1g-acc",
		NameI18n:       json.RawMessage(`{"en-US":"1 CPU 1 GB"}`),
		Status:         "active",
		CPUCores:       1,
		MemoryMB:       1024,
		DiskGB:         25,
		PriceMinor:     1200,
		Currency:       "USD",
		BillingCycle:   "monthly",
		Virtualization: "kvm",
	})
	require.NoError(t, err)

	order, err := orderSvc.CreateOrder(ctx, userID, []serviceCommerce.CreateOrderItemInput{
		{PlanID: planID, Quantity: 1},
	})
	require.NoError(t, err)

	payRes, err := paymentSvc.InitiatePayment(ctx, order.ID, "fake")
	require.NoError(t, err)
	paymentNo := payRes.Payment.PaymentNo

	router := server.NewRouterWithDeps(server.RouterDeps{
		Config:     cfg,
		ProductSvc: productSvc,
		OrderSvc:   orderSvc,
		PaymentSvc: paymentSvc,
		WalletSvc:  walletSvc,
	})

	body, sig, err := fakeGateway.GenerateSimulatedWebhook(paymentNo, order.ID, 1200, "USD")
	require.NoError(t, err)

	const concurrency = 100
	var wg sync.WaitGroup
	var successCount int
	var mu sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/api/v1/payments/webhook/fake", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Signature", sig)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code == http.StatusOK {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, concurrency, successCount, "all 100 requests must return 200 OK")

	// Order must be paid
	updatedOrder, err := repo.GetOrderByID(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, domainCommerce.OrderStatusPaid, updatedOrder.Status)

	// Ensure exactly 1 ledger transaction and 2 balanced entries
	repo.mu.Lock()
	txCount := len(repo.ledgerTxs)
	entCount := len(repo.ledgerEnts)
	repo.mu.Unlock()

	assert.Equal(t, 1, txCount, "expected exactly 1 ledger transaction (0 duplicate charges)")
	assert.Equal(t, 2, entCount, "expected exactly 2 balanced ledger entries (1 debit, 1 credit)")
}

// Acceptance Test 2: 50 Concurrent Reinstall Requests
// Invariant: Exactly 1 reinstall operation accepted (HTTP 202), 49 rejected with HTTP 409 Conflict.
func TestAcceptance50ConcurrentReinstallConflict(t *testing.T) {
	cfg := &config.Config{
		UserWebOrigin:  "http://localhost:3000",
		AdminWebOrigin: "http://localhost:3001",
	}

	userRepo := newTestUserRepo()
	adminRepo := newTestAdminRepo()
	auditRepo := newTestAuditRepo()
	rbacRepo := &testRBACRepo{}
	sessStore := session.NewMemorySessionStore()
	sessMgr := session.NewManager(sessStore, false)
	auditSvc := serviceAudit.NewService(auditRepo)
	userSvc := serviceIdentity.NewUserService(userRepo, auditSvc, sessMgr)
	adminSvc := serviceIdentity.NewAdminService(adminRepo, rbacRepo, auditSvc, sessMgr)

	subRepo := newTestSubRepo()
	infraRepo := newTestInfraRepo()
	opRepo := newTestOpRepo()
	commerceRepo := newTestCommerceRepo()
	subSvc := serviceSubscription.NewSubscriptionService(subRepo, commerceRepo)
	opSvc := serviceOperation.NewService(opRepo, nil)

	r := server.NewRouterWithDeps(server.RouterDeps{
		Config:          cfg,
		UserSvc:         userSvc,
		AdminSvc:        adminSvc,
		AuditSvc:        auditSvc,
		SessionMgr:      sessMgr,
		SubscriptionSvc: subSvc,
		OperationSvc:    opSvc,
		InfraRepo:       infraRepo,
	})

	userID := uuid.New()
	user := &domainIdentity.User{
		ID:     userID,
		Email:  "user-50@example.com",
		Status: domainIdentity.UserStatusActive,
	}
	_ = userRepo.Create(context.Background(), user)

	sess, err := sessMgr.CreateUserSession(context.Background(), user.ID, user.Email)
	require.NoError(t, err)

	subID := uuid.New()
	subRepo.subs[subID] = &domainSubscription.Subscription{
		ID:     subID,
		UserID: user.ID,
		Status: domainSubscription.StatusActive,
	}

	instID := uuid.New()
	infraRepo.instances[instID] = &domainInfrastructure.Instance{
		ID:             instID,
		SubscriptionID: subID,
		DesiredState:   "running",
		ObservedState:  "running",
	}

	var wg sync.WaitGroup
	statusCodes := make([]int, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			reqBody := bytes.NewBufferString(`{"image":"ubuntu-24.04"}`)
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/instances/%s/reinstall", instID), reqBody)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", sess.CSRFToken)
			req.AddCookie(&http.Cookie{
				Name:  session.UserSessionCookieName,
				Value: sess.Token,
			})

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			statusCodes[idx] = rec.Code
		}(i)
	}

	wg.Wait()

	countAccepted := 0
	countConflict := 0
	for _, code := range statusCodes {
		if code == http.StatusAccepted {
			countAccepted++
		} else if code == http.StatusConflict {
			countConflict++
		}
	}

	assert.Equal(t, 1, countAccepted, "expected exactly 1 accepted operation")
	assert.Equal(t, 49, countConflict, "expected 49 conflict (409) rejections")
}

// Acceptance Test 3: 20 Duplicate Create Requests with Idempotency Key
// Invariant: Exactly 1 operation created, zero duplicate operations.
func TestAcceptance20DuplicateCreatePrevention(t *testing.T) {
	ctx := context.Background()
	opRepo := newTestOpRepo()
	opSvc := serviceOperation.NewService(opRepo, nil)

	idemKey := "test-create-vm-idempotent-key"
	instID := uuid.New()

	var wg sync.WaitGroup
	ops := make([]*domainOperation.Operation, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			op, err := opSvc.CreateOperation(ctx, serviceOperation.CreateOperationInput{
				Type:           "provision_instance",
				ResourceType:   "instance",
				ResourceID:     instID,
				IdempotencyKey: idemKey,
				Retryable:      true,
				MaxRetries:     3,
				TraceID:        "trace-acceptance-create",
			})
			if err == nil {
				ops[idx] = op
			}
		}(i)
	}

	wg.Wait()

	firstOpID := ops[0].ID
	for _, op := range ops {
		require.NotNil(t, op)
		assert.Equal(t, firstOpID, op.ID, "all calls must return the same single operation")
	}
	assert.Equal(t, 1, len(opRepo.operations))
}

// Acceptance Test 4: 20 Worker Crash / Stuck Operation Recoveries
// Invariant: All 20 stuck operations recovered into retry queue; 0 lost tasks.
func TestAcceptance20WorkerCrashRecoveries(t *testing.T) {
	ctx := context.Background()
	infraRepo := newTestInfraRepo()
	opRepo := newTestOpRepo()
	opSvc := serviceOperation.NewService(opRepo, nil)

	stuckTime := time.Now().UTC().Add(-15 * time.Minute)
	opIDs := make([]uuid.UUID, 20)

	for i := 0; i < 20; i++ {
		id := uuid.New()
		opIDs[i] = id
		msg := "stuck"
		phase := "provisioning"
		opRepo.operations[id] = &domainOperation.Operation{
			ID:         id,
			Type:       "provision_instance",
			Status:     domainOperation.StatusRunning,
			Phase:      &phase,
			MessageKey: &msg,
			Progress:   40,
			Retryable:  true,
			RetryCount: 0,
			MaxRetries: 3,
			StartedAt:  &stuckTime,
		}
	}

	rec := reconciler.NewReconciler(infraRepo, opRepo, opSvc, nil, 3*time.Minute, 1*time.Minute)
	recovered, err := rec.ReconcileStuckOperations(ctx)
	require.NoError(t, err)
	assert.Equal(t, 20, recovered)

	// All 20 must now be in StatusRetrying
	for _, id := range opIDs {
		assert.Equal(t, domainOperation.StatusRetrying, opRepo.operations[id].Status)
	}
}

// Acceptance Test 5: 20 Node Offline / Agent Disconnect
// Invariant: Nodes marked offline, instances marked unknown, NEVER deleted!
func TestAcceptance20NodeOfflineAndAgentDisconnectNeverDeleted(t *testing.T) {
	ctx := context.Background()
	infraRepo := newTestInfraRepo()
	opRepo := newTestOpRepo()
	opSvc := serviceOperation.NewService(opRepo, nil)

	staleTime := time.Now().UTC().Add(-10 * time.Minute)
	nodeIDs := make([]uuid.UUID, 20)
	instIDs := make([]uuid.UUID, 20)

	for i := 0; i < 20; i++ {
		nid := uuid.New()
		iid := uuid.New()
		nodeIDs[i] = nid
		instIDs[i] = iid

		infraRepo.nodes[nid] = &domainInfrastructure.Node{
			ID:         nid,
			Name:       fmt.Sprintf("node-%d", i),
			Status:     "active",
			LastSeenAt: &staleTime,
		}

		infraRepo.instances[iid] = &domainInfrastructure.Instance{
			ID:            iid,
			NodeID:        &nid,
			DesiredState:  "running",
			ObservedState: "running",
		}
	}

	rec := reconciler.NewReconciler(infraRepo, opRepo, opSvc, nil, 3*time.Minute, 1*time.Minute)
	marked, err := rec.ReconcileNodeHeartbeats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 20, marked)

	// Check all 20 nodes are offline
	for _, nid := range nodeIDs {
		assert.Equal(t, "offline", infraRepo.nodes[nid].Status)
	}

	// Check all 20 instances are "unknown", NEVER deleted!
	assert.Equal(t, 20, len(infraRepo.instances), "zero instances may be deleted")
	for _, iid := range instIDs {
		assert.Equal(t, "unknown", infraRepo.instances[iid].ObservedState)
	}
}

// Acceptance Test 6: Auth Bypass Prevention
// Invariant: Unauthenticated requests to protected endpoints return 401 Unauthorized.
func TestAcceptanceAuthBypassPrevention(t *testing.T) {
	cfg := &config.Config{
		UserWebOrigin:  "http://localhost:3000",
		AdminWebOrigin: "http://localhost:3001",
	}

	userRepo := newTestUserRepo()
	adminRepo := newTestAdminRepo()
	auditRepo := newTestAuditRepo()
	rbacRepo := &testRBACRepo{}
	sessStore := session.NewMemorySessionStore()
	sessMgr := session.NewManager(sessStore, false)
	auditSvc := serviceAudit.NewService(auditRepo)
	userSvc := serviceIdentity.NewUserService(userRepo, auditSvc, sessMgr)
	adminSvc := serviceIdentity.NewAdminService(adminRepo, rbacRepo, auditSvc, sessMgr)

	commerceRepo := newTestCommerceRepo()
	fakeGateway := serviceCommerce.NewFakePaymentGateway("secret_test")
	productSvc := serviceCommerce.NewProductService(commerceRepo)
	orderSvc := serviceCommerce.NewOrderService(commerceRepo, commerceRepo)
	paymentSvc := serviceCommerce.NewPaymentService(commerceRepo, commerceRepo, fakeGateway)
	walletSvc := serviceCommerce.NewWalletService(commerceRepo, commerceRepo)

	subRepo := newTestSubRepo()
	infraRepo := newTestInfraRepo()
	opRepo := newTestOpRepo()
	subSvc := serviceSubscription.NewSubscriptionService(subRepo, commerceRepo)
	opSvc := serviceOperation.NewService(opRepo, nil)

	r := server.NewRouterWithDeps(server.RouterDeps{
		Config:          cfg,
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

	protectedPaths := []string{
		"/api/v1/auth/me",
		"/api/v1/orders",
		"/api/v1/wallet",
		"/api/v1/invoices",
		"/api/v1/subscriptions",
		"/api/v1/instances",
		"/api/v1/admin/auth/me",
		"/api/v1/admin/audit",
		"/api/v1/admin/providers",
		"/api/v1/admin/nodes",
		"/api/v1/admin/instances",
		"/api/v1/admin/operations",
		"/api/v1/admin/users",
		"/api/v1/admin/admins",
	}

	for _, path := range protectedPaths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "path %s must return 401 Unauthorized without credentials", path)
	}
}

// Acceptance Test 7: Prometheus Metrics Endpoint Availability
func TestAcceptanceMetricsEndpoint(t *testing.T) {
	cfg := &config.Config{
		UserWebOrigin:  "http://localhost:3000",
		AdminWebOrigin: "http://localhost:3001",
	}

	r := server.NewRouterWithDeps(server.RouterDeps{
		Config: cfg,
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "vps_billing_info")
	assert.Contains(t, body, "vps_billing_requests_total")
	assert.Contains(t, body, "vps_billing_uptime_seconds")
}
