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
	"vps-billing/internal/config"
	domainCommerce "vps-billing/internal/domain/commerce"
	"vps-billing/internal/server"
	serviceAudit "vps-billing/internal/service/audit"
	serviceCommerce "vps-billing/internal/service/commerce"
	serviceIdentity "vps-billing/internal/service/identity"
	"vps-billing/internal/service/session"
)

// In-memory thread-safe commerce repository for unit and gate testing
type testCommerceRepo struct {
	mu sync.Mutex

	products   map[uuid.UUID]*domainCommerce.Product
	plans      map[uuid.UUID]*domainCommerce.Plan
	orders     map[uuid.UUID]*domainCommerce.Order
	payments   map[uuid.UUID]*domainCommerce.Payment
	invoices   map[uuid.UUID]*domainCommerce.Invoice
	wallets    map[string]*domainCommerce.Wallet // key: userID:currency
	ledgerTxs  []*domainCommerce.LedgerTransaction
	ledgerEnts []*domainCommerce.LedgerEntry
	outboxEvts []*domainCommerce.OutboxEvent
}

func newTestCommerceRepo() *testCommerceRepo {
	return &testCommerceRepo{
		products: make(map[uuid.UUID]*domainCommerce.Product),
		plans:    make(map[uuid.UUID]*domainCommerce.Plan),
		orders:   make(map[uuid.UUID]*domainCommerce.Order),
		payments: make(map[uuid.UUID]*domainCommerce.Payment),
		invoices: make(map[uuid.UUID]*domainCommerce.Invoice),
		wallets:  make(map[string]*domainCommerce.Wallet),
	}
}

// ProductRepo methods
func (r *testCommerceRepo) CreateProduct(_ context.Context, p *domainCommerce.Product) (*domainCommerce.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[p.ID] = p
	return p, nil
}

func (r *testCommerceRepo) GetProductByID(_ context.Context, id uuid.UUID) (*domainCommerce.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.products[id]; ok {
		return p, nil
	}
	return nil, domainCommerce.ErrProductNotFound
}

func (r *testCommerceRepo) GetProductBySlug(_ context.Context, slug string) (*domainCommerce.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.products {
		if p.Slug == slug {
			return p, nil
		}
	}
	return nil, domainCommerce.ErrProductNotFound
}

func (r *testCommerceRepo) ListActiveProductsWithPlans(_ context.Context) ([]*domainCommerce.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainCommerce.Product
	for _, p := range r.products {
		if p.Status == "active" {
			pCopy := *p
			pCopy.Plans = nil
			for _, pl := range r.plans {
				if pl.ProductID == p.ID && pl.Status == "active" {
					pCopy.Plans = append(pCopy.Plans, *pl)
				}
			}
			res = append(res, &pCopy)
		}
	}
	return res, nil
}

func (r *testCommerceRepo) ListAllProducts(_ context.Context) ([]*domainCommerce.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainCommerce.Product
	for _, p := range r.products {
		res = append(res, p)
	}
	return res, nil
}

func (r *testCommerceRepo) UpdateProduct(_ context.Context, p *domainCommerce.Product) (*domainCommerce.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[p.ID] = p
	return p, nil
}

func (r *testCommerceRepo) CreatePlan(_ context.Context, plan *domainCommerce.Plan) (*domainCommerce.Plan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plans[plan.ID] = plan
	return plan, nil
}

func (r *testCommerceRepo) GetPlanByID(_ context.Context, id uuid.UUID) (*domainCommerce.Plan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if pl, ok := r.plans[id]; ok {
		return pl, nil
	}
	return nil, domainCommerce.ErrPlanNotFound
}

func (r *testCommerceRepo) GetPlanByProductAndSlug(_ context.Context, productID uuid.UUID, slug string) (*domainCommerce.Plan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, pl := range r.plans {
		if pl.ProductID == productID && pl.Slug == slug {
			return pl, nil
		}
	}
	return nil, domainCommerce.ErrPlanNotFound
}

func (r *testCommerceRepo) ListPlansByProductID(_ context.Context, productID uuid.UUID) ([]*domainCommerce.Plan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainCommerce.Plan
	for _, pl := range r.plans {
		if pl.ProductID == productID {
			res = append(res, pl)
		}
	}
	return res, nil
}

func (r *testCommerceRepo) ListActivePlansByProductID(_ context.Context, productID uuid.UUID) ([]*domainCommerce.Plan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainCommerce.Plan
	for _, pl := range r.plans {
		if pl.ProductID == productID && pl.Status == "active" {
			res = append(res, pl)
		}
	}
	return res, nil
}

func (r *testCommerceRepo) UpdatePlan(_ context.Context, plan *domainCommerce.Plan) (*domainCommerce.Plan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plans[plan.ID] = plan
	return plan, nil
}

// OrderRepo methods
func (r *testCommerceRepo) CreateOrderWithItems(
	_ context.Context,
	order *domainCommerce.Order,
	items []domainCommerce.OrderItem,
	invoice *domainCommerce.Invoice,
	invoiceItem *domainCommerce.InvoiceItem,
) (*domainCommerce.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	order.Items = items
	r.orders[order.ID] = order

	if invoice != nil {
		if invoiceItem != nil {
			invoice.Items = []domainCommerce.InvoiceItem{*invoiceItem}
		}
		r.invoices[invoice.ID] = invoice
	}

	return order, nil
}

func (r *testCommerceRepo) GetOrderByID(_ context.Context, id uuid.UUID) (*domainCommerce.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ord, ok := r.orders[id]; ok {
		return ord, nil
	}
	return nil, domainCommerce.ErrOrderNotFound
}

func (r *testCommerceRepo) GetOrderByOrderNo(_ context.Context, orderNo string) (*domainCommerce.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ord := range r.orders {
		if ord.OrderNo == orderNo {
			return ord, nil
		}
	}
	return nil, domainCommerce.ErrOrderNotFound
}

func (r *testCommerceRepo) ListOrdersByUserID(_ context.Context, userID uuid.UUID) ([]*domainCommerce.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainCommerce.Order
	for _, ord := range r.orders {
		if ord.UserID == userID {
			res = append(res, ord)
		}
	}
	return res, nil
}

func (r *testCommerceRepo) ListAllOrders(_ context.Context) ([]*domainCommerce.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainCommerce.Order
	for _, ord := range r.orders {
		res = append(res, ord)
	}
	return res, nil
}

func (r *testCommerceRepo) UpdateOrderStatus(_ context.Context, id uuid.UUID, status domainCommerce.OrderStatus) (*domainCommerce.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ord, ok := r.orders[id]; ok {
		ord.Status = status
		if status == domainCommerce.OrderStatusPaid {
			now := time.Now().UTC()
			ord.PaidAt = &now
		}
		return ord, nil
	}
	return nil, domainCommerce.ErrOrderNotFound
}

// PaymentRepo methods
func (r *testCommerceRepo) CreatePayment(_ context.Context, p *domainCommerce.Payment) (*domainCommerce.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.payments[p.ID] = p
	return p, nil
}

func (r *testCommerceRepo) GetPaymentByID(_ context.Context, id uuid.UUID) (*domainCommerce.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.payments[id]; ok {
		return p, nil
	}
	return nil, domainCommerce.ErrPaymentNotFound
}

func (r *testCommerceRepo) GetPaymentByPaymentNo(_ context.Context, paymentNo string) (*domainCommerce.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.payments {
		if p.PaymentNo == paymentNo {
			return p, nil
		}
	}
	return nil, domainCommerce.ErrPaymentNotFound
}

func (r *testCommerceRepo) GetPaymentByIdempotencyKey(_ context.Context, key string) (*domainCommerce.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.payments {
		if p.IdempotencyKey == key {
			return p, nil
		}
	}
	return nil, domainCommerce.ErrPaymentNotFound
}

func (r *testCommerceRepo) GetPaymentByGatewayAndExternalID(_ context.Context, gateway, externalID string) (*domainCommerce.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.payments {
		if p.Gateway == gateway && p.GatewayPaymentID != nil && *p.GatewayPaymentID == externalID {
			return p, nil
		}
	}
	return nil, domainCommerce.ErrPaymentNotFound
}

func (r *testCommerceRepo) ListPaymentsByOrderID(_ context.Context, orderID uuid.UUID) ([]*domainCommerce.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainCommerce.Payment
	for _, p := range r.payments {
		if p.OrderID == orderID {
			res = append(res, p)
		}
	}
	return res, nil
}

func (r *testCommerceRepo) ListAllPayments(_ context.Context) ([]*domainCommerce.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainCommerce.Payment
	for _, p := range r.payments {
		res = append(res, p)
	}
	return res, nil
}

// ProcessPaymentSuccessTx simulates the exact atomic transaction with row locking and idempotency protection
func (r *testCommerceRepo) ProcessPaymentSuccessTx(_ context.Context, params domainCommerce.PaymentSuccessParams) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.payments[params.PaymentID]
	if !ok {
		return false, domainCommerce.ErrPaymentNotFound
	}

	// If already succeeded, return immediately with alreadyProcessed = true
	if p.Status == domainCommerce.PaymentStatusSucceeded {
		return true, nil
	}

	now := time.Now().UTC()
	p.Status = domainCommerce.PaymentStatusSucceeded
	p.GatewayPaymentID = &params.GatewayPaymentID
	p.GatewayPayload = params.GatewayPayload
	p.PaidAt = &now

	// Update order
	if ord, ok := r.orders[params.OrderID]; ok {
		ord.Status = domainCommerce.OrderStatusPaid
		ord.PaidAt = &now
	}

	// Update invoice
	for _, inv := range r.invoices {
		if inv.OrderID != nil && *inv.OrderID == params.OrderID {
			inv.Status = domainCommerce.InvoiceStatusPaid
			inv.PaidAt = &now
		}
	}

	// Double entry ledger transaction & entries
	ledgerTxID := uuid.New()
	desc := fmt.Sprintf("Payment %s for Order %s", params.PaymentNo, params.OrderID)
	refType := "payment"
	r.ledgerTxs = append(r.ledgerTxs, &domainCommerce.LedgerTransaction{
		ID:            ledgerTxID,
		Type:          "order_payment",
		ReferenceType: &refType,
		ReferenceID:   &params.PaymentID,
		Description:   &desc,
		CreatedAt:     now,
	})

	r.ledgerEnts = append(r.ledgerEnts,
		&domainCommerce.LedgerEntry{
			ID:            uuid.New(),
			TransactionID: ledgerTxID,
			AccountType:   domainCommerce.AccountPaymentGateway,
			AccountID:     domainCommerce.SystemAccountPaymentGateway,
			Direction:     domainCommerce.DirectionDebit,
			AmountMinor:   params.AmountMinor,
			Currency:      params.Currency,
			CreatedAt:     now,
		},
		&domainCommerce.LedgerEntry{
			ID:            uuid.New(),
			TransactionID: ledgerTxID,
			AccountType:   domainCommerce.AccountPlatformRevenue,
			AccountID:     domainCommerce.SystemAccountPlatformRevenue,
			Direction:     domainCommerce.DirectionCredit,
			AmountMinor:   params.AmountMinor,
			Currency:      params.Currency,
			CreatedAt:     now,
		},
	)

	// Outbox event
	r.outboxEvts = append(r.outboxEvts, &domainCommerce.OutboxEvent{
		ID:            uuid.New(),
		EventType:     "payment.succeeded.v1",
		AggregateType: "payment",
		AggregateID:   params.PaymentID,
		Payload:       json.RawMessage(`{}`),
		Status:        "pending",
		CreatedAt:     now,
	})

	return false, nil
}

// InvoiceRepo methods
func (r *testCommerceRepo) GetInvoiceByID(_ context.Context, id uuid.UUID) (*domainCommerce.Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if inv, ok := r.invoices[id]; ok {
		return inv, nil
	}
	return nil, domainCommerce.ErrInvoiceNotFound
}

func (r *testCommerceRepo) GetInvoiceByOrderID(_ context.Context, orderID uuid.UUID) (*domainCommerce.Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, inv := range r.invoices {
		if inv.OrderID != nil && *inv.OrderID == orderID {
			return inv, nil
		}
	}
	return nil, domainCommerce.ErrInvoiceNotFound
}

func (r *testCommerceRepo) ListInvoicesByUserID(_ context.Context, userID uuid.UUID) ([]*domainCommerce.Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainCommerce.Invoice
	for _, inv := range r.invoices {
		if inv.UserID == userID {
			res = append(res, inv)
		}
	}
	return res, nil
}

func (r *testCommerceRepo) ListAllInvoices(_ context.Context) ([]*domainCommerce.Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainCommerce.Invoice
	for _, inv := range r.invoices {
		res = append(res, inv)
	}
	return res, nil
}

// WalletRepo methods
func (r *testCommerceRepo) GetOrCreateWallet(_ context.Context, userID uuid.UUID, currency string) (*domainCommerce.Wallet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := fmt.Sprintf("%s:%s", userID.String(), currency)
	if w, ok := r.wallets[key]; ok {
		return w, nil
	}
	w := &domainCommerce.Wallet{
		ID:                    uuid.New(),
		UserID:                userID,
		Currency:              currency,
		AvailableBalanceMinor: 0,
		CreatedAt:             time.Now().UTC(),
		UpdatedAt:             time.Now().UTC(),
	}
	r.wallets[key] = w
	return w, nil
}

func (r *testCommerceRepo) ListLedgerTransactions(_ context.Context, limit, offset int) ([]*domainCommerce.LedgerTransaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	start := offset
	if start > len(r.ledgerTxs) {
		return nil, nil
	}
	end := start + limit
	if end > len(r.ledgerTxs) {
		end = len(r.ledgerTxs)
	}
	return r.ledgerTxs[start:end], nil
}

func (r *testCommerceRepo) ListUserLedgerEntries(_ context.Context, userID uuid.UUID) ([]*domainCommerce.LedgerEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainCommerce.LedgerEntry
	for _, ent := range r.ledgerEnts {
		if ent.AccountID == userID {
			res = append(res, ent)
		}
	}
	return res, nil
}

func (r *testCommerceRepo) DepositWalletTx(_ context.Context, userID uuid.UUID, amountMinor int64, currency, description string) (*domainCommerce.Wallet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if currency == "" {
		currency = "USD"
	}
	key := fmt.Sprintf("%s:%s", userID.String(), currency)
	w, ok := r.wallets[key]
	if !ok {
		w = &domainCommerce.Wallet{
			ID:                    uuid.New(),
			UserID:                userID,
			Currency:              currency,
			AvailableBalanceMinor: 0,
			CreatedAt:             time.Now().UTC(),
			UpdatedAt:             time.Now().UTC(),
		}
		r.wallets[key] = w
	}
	w.AvailableBalanceMinor += amountMinor
	w.UpdatedAt = time.Now().UTC()

	now := time.Now().UTC()
	txID := uuid.New()
	refType := "wallet_deposit"
	r.ledgerTxs = append(r.ledgerTxs, &domainCommerce.LedgerTransaction{
		ID:            txID,
		Type:          "wallet_deposit",
		ReferenceType: &refType,
		ReferenceID:   &w.ID,
		Description:   &description,
		CreatedAt:     now,
	})
	r.ledgerEnts = append(r.ledgerEnts,
		&domainCommerce.LedgerEntry{
			ID:            uuid.New(),
			TransactionID: txID,
			AccountType:   domainCommerce.AccountPaymentGateway,
			AccountID:     domainCommerce.SystemAccountPaymentGateway,
			Direction:     domainCommerce.DirectionDebit,
			AmountMinor:   amountMinor,
			Currency:      currency,
			CreatedAt:     now,
		},
		&domainCommerce.LedgerEntry{
			ID:            uuid.New(),
			TransactionID: txID,
			AccountType:   domainCommerce.AccountUserWallet,
			AccountID:     userID,
			Direction:     domainCommerce.DirectionCredit,
			AmountMinor:   amountMinor,
			Currency:      currency,
			CreatedAt:     now,
		},
	)

	return w, nil
}

// GATE TEST: Duplicate Payment Webhook Concurrency Gate Test (100 concurrent callbacks)
func TestDuplicateWebhookConcurrency(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		AppEnv:         "test",
		HTTPPort:       "8080",
		UserWebOrigin:  "http://localhost:3000",
		AdminWebOrigin: "http://localhost:3001",
	}

	repo := newTestCommerceRepo()
	fakeGateway := serviceCommerce.NewFakePaymentGateway("test_webhook_secret_key_123")
	productSvc := serviceCommerce.NewProductService(repo)
	orderSvc := serviceCommerce.NewOrderService(repo, repo)
	paymentSvc := serviceCommerce.NewPaymentService(repo, repo, fakeGateway)
	walletSvc := serviceCommerce.NewWalletService(repo, repo)

	userRepo := newTestUserRepo()
	auditRepo := newTestAuditRepo()
	auditSvc := serviceAudit.NewService(auditRepo)
	sessStore := session.NewMemorySessionStore()
	sessMgr := session.NewManager(sessStore, false)
	userSvc := serviceIdentity.NewUserService(userRepo, auditSvc, sessMgr)

	// Seed user
	user, err := userSvc.Register(ctx, "payer@example.com", "Password123!", "en-US", "UTC", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	// Seed product and plan
	prodID := uuid.New()
	_, err = productSvc.CreateProduct(ctx, &domainCommerce.Product{
		ID:        prodID,
		Slug:      "vps-cloud",
		NameI18n:  json.RawMessage(`{"en-US":"Cloud VPS"}`),
		Status:    "active",
		SortOrder: 1,
	})
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	planID := uuid.New()
	_, err = productSvc.CreatePlan(ctx, &domainCommerce.Plan{
		ID:             planID,
		ProductID:      prodID,
		Slug:           "1c1g",
		NameI18n:       json.RawMessage(`{"en-US":"1 CPU 1 GB"}`),
		Status:         "active",
		CPUCores:       1,
		MemoryMB:       1024,
		DiskGB:         25,
		PriceMinor:     1200, // $12.00
		Currency:       "USD",
		BillingCycle:   "monthly",
		Virtualization: "kvm",
	})
	if err != nil {
		t.Fatalf("failed to create plan: %v", err)
	}

	// 1. Create order
	order, err := orderSvc.CreateOrder(ctx, user.ID, []serviceCommerce.CreateOrderItemInput{
		{PlanID: planID, Quantity: 1},
	})
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	if order.TotalMinor != 1200 {
		t.Fatalf("expected order total 1200, got %d", order.TotalMinor)
	}

	// 2. Initiate payment
	payRes, err := paymentSvc.InitiatePayment(ctx, order.ID, "fake")
	if err != nil {
		t.Fatalf("failed to initiate payment: %v", err)
	}
	paymentNo := payRes.Payment.PaymentNo

	// 3. Build router
	router := server.NewRouterWithDeps(server.RouterDeps{
		Config:     cfg,
		UserSvc:    userSvc,
		SessionMgr: sessMgr,
		ProductSvc: productSvc,
		OrderSvc:   orderSvc,
		PaymentSvc: paymentSvc,
		WalletSvc:  walletSvc,
	})

	// 4. Generate identical signed webhook payload
	body, sig, err := fakeGateway.GenerateSimulatedWebhook(paymentNo, order.ID, 1200, "USD")
	if err != nil {
		t.Fatalf("failed to generate simulated webhook: %v", err)
	}

	// 5. Fire 100 concurrent requests with the EXACT SAME webhook payload
	const concurrency = 100
	var wg sync.WaitGroup
	var successCount int
	var mu sync.Mutex

	startSignal := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startSignal // ensure all goroutines start simultaneously

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

	// Release all 100 goroutines simultaneously
	close(startSignal)
	wg.Wait()

	// 6. Assertions for Gate
	// All 100 requests must return 200 OK
	if successCount != concurrency {
		t.Fatalf("expected %d successful 200 responses, got %d", concurrency, successCount)
	}

	// Check Payment in DB
	payment, err := repo.GetPaymentByPaymentNo(ctx, paymentNo)
	if err != nil {
		t.Fatalf("failed to get payment: %v", err)
	}
	if payment.Status != domainCommerce.PaymentStatusSucceeded {
		t.Errorf("expected payment status succeeded, got %s", payment.Status)
	}

	// Check Order in DB
	ord, err := repo.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to get order: %v", err)
	}
	if ord.Status != domainCommerce.OrderStatusPaid {
		t.Errorf("expected order status paid, got %s", ord.Status)
	}

	// CRITICAL GATE INVARIANT: Exactly 1 ledger transaction created
	repo.mu.Lock()
	ledgerTxCount := len(repo.ledgerTxs)
	ledgerEntCount := len(repo.ledgerEnts)
	outboxCount := len(repo.outboxEvts)
	repo.mu.Unlock()

	if ledgerTxCount != 1 {
		t.Fatalf("GATE VIOLATION: expected exactly 1 ledger transaction, got %d (duplicate ledger credit occurred!)", ledgerTxCount)
	}

	if ledgerEntCount != 2 {
		t.Fatalf("GATE VIOLATION: expected exactly 2 balanced ledger entries (1 debit, 1 credit), got %d", ledgerEntCount)
	}

	// Verify balance of debit and credit
	var totalDebit, totalCredit int64
	repo.mu.Lock()
	for _, ent := range repo.ledgerEnts {
		if ent.Direction == domainCommerce.DirectionDebit {
			totalDebit += ent.AmountMinor
		} else if ent.Direction == domainCommerce.DirectionCredit {
			totalCredit += ent.AmountMinor
		}
	}
	repo.mu.Unlock()

	if totalDebit != 1200 || totalCredit != 1200 {
		t.Errorf("ledger unbalanced: debit=%d, credit=%d, expected 1200 each", totalDebit, totalCredit)
	}

	if outboxCount != 1 {
		t.Fatalf("expected exactly 1 outbox event, got %d", outboxCount)
	}

	// 7. Follow-up: Fire 50 additional sequential requests to confirm ongoing idempotency
	for i := 0; i < 50; i++ {
		req := httptest.NewRequest("POST", "/api/v1/payments/webhook/fake", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Signature", sig)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("subsequent call %d failed with code %d", i, rec.Code)
		}
	}

	repo.mu.Lock()
	finalLedgerTxCount := len(repo.ledgerTxs)
	finalLedgerEntCount := len(repo.ledgerEnts)
	repo.mu.Unlock()

	if finalLedgerTxCount != 1 || finalLedgerEntCount != 2 {
		t.Fatalf("GATE VIOLATION: subsequent requests created extra ledger entries: txs=%d, ents=%d", finalLedgerTxCount, finalLedgerEntCount)
	}

	t.Logf("GATE PASSED: 150 duplicate webhook callbacks resulted in exactly 1 ledger entry, 0 duplicate charges, 100%% idempotent.")
}

func TestCommerceAPI_CatalogAndOrderFlow(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		AppEnv:         "test",
		HTTPPort:       "8080",
		UserWebOrigin:  "http://localhost:3000",
		AdminWebOrigin: "http://localhost:3001",
	}

	repo := newTestCommerceRepo()
	fakeGateway := serviceCommerce.NewFakePaymentGateway("secret_test")
	productSvc := serviceCommerce.NewProductService(repo)
	orderSvc := serviceCommerce.NewOrderService(repo, repo)
	paymentSvc := serviceCommerce.NewPaymentService(repo, repo, fakeGateway)
	walletSvc := serviceCommerce.NewWalletService(repo, repo)

	userRepo := newTestUserRepo()
	adminRepo := newTestAdminRepo()
	rbacRepo := &testRBACRepo{}
	auditRepo := newTestAuditRepo()
	auditSvc := serviceAudit.NewService(auditRepo)
	sessStore := session.NewMemorySessionStore()
	sessMgr := session.NewManager(sessStore, false)
	userSvc := serviceIdentity.NewUserService(userRepo, auditSvc, sessMgr)
	adminSvc := serviceIdentity.NewAdminService(adminRepo, rbacRepo, auditSvc, sessMgr)

	// Seed super admin
	_, err := adminSvc.CreateInitialAdmin(ctx, "admin@test.com", "AdminPass123!", "Super Admin", "super_admin")
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	// Seed user
	_, err = userSvc.Register(ctx, "customer@test.com", "CustomerPass123!", "en-US", "UTC", "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Seed product and plan
	prodID := uuid.New()
	_, _ = productSvc.CreateProduct(ctx, &domainCommerce.Product{
		ID:        prodID,
		Slug:      "vps-premium",
		NameI18n:  json.RawMessage(`{"en-US":"Premium VPS","zh-CN":"高级 VPS"}`),
		Status:    "active",
		SortOrder: 1,
	})

	planID := uuid.New()
	_, _ = productSvc.CreatePlan(ctx, &domainCommerce.Plan{
		ID:             planID,
		ProductID:      prodID,
		Slug:           "2c2g",
		NameI18n:       json.RawMessage(`{"en-US":"2 Cores 2GB RAM","zh-CN":"2核 2G"}`),
		Status:         "active",
		CPUCores:       2,
		MemoryMB:       2048,
		DiskGB:         50,
		PriceMinor:     1500, // $15.00
		Currency:       "USD",
		BillingCycle:   "monthly",
		Virtualization: "kvm",
	})

	router := server.NewRouterWithDeps(server.RouterDeps{
		Config:     cfg,
		UserSvc:    userSvc,
		AdminSvc:   adminSvc,
		SessionMgr: sessMgr,
		ProductSvc: productSvc,
		OrderSvc:   orderSvc,
		PaymentSvc: paymentSvc,
		WalletSvc:  walletSvc,
	})

	// 1. GET /api/v1/products (public)
	req := httptest.NewRequest("GET", "/api/v1/products", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /products failed with %d: %s", rec.Code, rec.Body.String())
	}

	var prodResp struct {
		Success bool `json:"success"`
		Data    struct {
			Products []*domainCommerce.Product `json:"products"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&prodResp); err != nil {
		t.Fatalf("failed to decode products response: %v", err)
	}
	if len(prodResp.Data.Products) != 1 || len(prodResp.Data.Products[0].Plans) != 1 {
		t.Fatalf("expected 1 product with 1 plan, got %d products", len(prodResp.Data.Products))
	}

	// Log in user
	loginBody := `{"email":"customer@test.com","password":"CustomerPass123!"}`
	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	recLogin := httptest.NewRecorder()
	router.ServeHTTP(recLogin, reqLogin)
	if recLogin.Code != http.StatusOK {
		t.Fatalf("failed user login: %d, body: %s", recLogin.Code, recLogin.Body.String())
	}
	var userLoginResp struct {
		Data struct {
			CsrfToken string `json:"csrf_token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recLogin.Body.Bytes(), &userLoginResp)
	userCsrf := userLoginResp.Data.CsrfToken

	var userCookie *http.Cookie
	for _, c := range recLogin.Result().Cookies() {
		if c.Name == session.UserSessionCookieName {
			userCookie = c
			break
		}
	}
	if userCookie == nil {
		t.Fatalf("missing user cookie")
	}

	// Log in admin
	adminLoginBody := `{"email":"admin@test.com","password":"AdminPass123!"}`
	reqAdminLogin := httptest.NewRequest("POST", "/api/v1/admin/auth/login", bytes.NewBufferString(adminLoginBody))
	reqAdminLogin.Header.Set("Content-Type", "application/json")
	recAdminLogin := httptest.NewRecorder()
	router.ServeHTTP(recAdminLogin, reqAdminLogin)
	if recAdminLogin.Code != http.StatusOK {
		t.Fatalf("failed admin login: %d", recAdminLogin.Code)
	}
	var adminCookie *http.Cookie
	for _, c := range recAdminLogin.Result().Cookies() {
		if c.Name == session.AdminSessionCookieName {
			adminCookie = c
			break
		}
	}
	if adminCookie == nil {
		t.Fatalf("missing admin cookie")
	}

	// 2. POST /api/v1/orders (authenticated user)
	orderReqBody, _ := json.Marshal(map[string]any{
		"items": []map[string]any{
			{"plan_id": planID, "quantity": 2},
		},
	})
	req = httptest.NewRequest("POST", "/api/v1/orders", bytes.NewReader(orderReqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", userCsrf)
	req.AddCookie(userCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /orders failed with %d: %s", rec.Code, rec.Body.String())
	}

	var orderResp struct {
		Success bool `json:"success"`
		Data    struct {
			Order *domainCommerce.Order `json:"order"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&orderResp); err != nil {
		t.Fatalf("failed to decode order response: %v", err)
	}
	createdOrder := orderResp.Data.Order
	if createdOrder.TotalMinor != 3000 { // 1500 * 2 = 3000
		t.Fatalf("expected total 3000 minor units, got %d", createdOrder.TotalMinor)
	}

	// 3. GET /api/v1/orders (authenticated user)
	req = httptest.NewRequest("GET", "/api/v1/orders", nil)
	req.AddCookie(userCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /orders failed with %d: %s", rec.Code, rec.Body.String())
	}

	// 4. POST /api/v1/orders/{id}/pay
	payReqBody, _ := json.Marshal(map[string]any{"gateway": "fake"})
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/orders/%s/pay", createdOrder.ID), bytes.NewReader(payReqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", userCsrf)
	req.AddCookie(userCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /orders/{id}/pay failed with %d: %s", rec.Code, rec.Body.String())
	}

	var payInitResp struct {
		Success bool `json:"success"`
		Data    struct {
			Payment     *domainCommerce.Payment `json:"payment"`
			CheckoutURL string                  `json:"checkout_url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payInitResp); err != nil {
		t.Fatalf("failed to decode pay init response: %v", err)
	}
	paymentNo := payInitResp.Data.Payment.PaymentNo

	// 5. POST /api/v1/payments/fake/simulate (simulate customer completing payment)
	simBody, _ := json.Marshal(map[string]any{"payment_no": paymentNo})
	req = httptest.NewRequest("POST", "/api/v1/payments/fake/simulate", bytes.NewReader(simBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /payments/fake/simulate failed with %d: %s", rec.Code, rec.Body.String())
	}

	// 6. Verify order is now paid
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/orders/%s", createdOrder.ID), nil)
	req.AddCookie(userCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /orders/{id} failed with %d", rec.Code)
	}
	var orderPaidResp struct {
		Data struct {
			Order *domainCommerce.Order `json:"order"`
		} `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&orderPaidResp)
	if orderPaidResp.Data.Order.Status != domainCommerce.OrderStatusPaid {
		t.Fatalf("expected order status paid, got %s", orderPaidResp.Data.Order.Status)
	}

	// 7. GET /api/v1/invoices (verify invoice exists and is paid)
	req = httptest.NewRequest("GET", "/api/v1/invoices", nil)
	req.AddCookie(userCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /invoices failed with %d", rec.Code)
	}

	// 8. Admin endpoints: GET /api/v1/admin/orders, /invoices, /ledger
	req = httptest.NewRequest("GET", "/api/v1/admin/orders", nil)
	req.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/orders failed with %d", rec.Code)
	}

	req = httptest.NewRequest("GET", "/api/v1/admin/ledger", nil)
	req.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/ledger failed with %d", rec.Code)
	}
}
