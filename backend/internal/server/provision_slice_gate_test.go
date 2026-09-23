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
	domainInfrastructure "vps-billing/internal/domain/infrastructure"
	domainOperation "vps-billing/internal/domain/operation"
	domainSubscription "vps-billing/internal/domain/subscription"
	"vps-billing/internal/provider"
	"vps-billing/internal/provider/lxdapi"
	"vps-billing/internal/provider/mock"
	"vps-billing/internal/server"
	serviceAudit "vps-billing/internal/service/audit"
	serviceCommerce "vps-billing/internal/service/commerce"
	serviceFulfillment "vps-billing/internal/service/fulfillment"
	serviceIdentity "vps-billing/internal/service/identity"
	serviceOperation "vps-billing/internal/service/operation"
	serviceScheduler "vps-billing/internal/service/scheduler"
	"vps-billing/internal/service/session"
	serviceSubscription "vps-billing/internal/service/subscription"
	"vps-billing/internal/workflow"
)

// In-memory subscription repository
type testSubRepo struct {
	mu   sync.Mutex
	subs map[uuid.UUID]*domainSubscription.Subscription
}

func newTestSubRepo() *testSubRepo {
	return &testSubRepo{subs: make(map[uuid.UUID]*domainSubscription.Subscription)}
}

func (r *testSubRepo) CreateSubscription(_ context.Context, sub *domainSubscription.Subscription) (*domainSubscription.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subs[sub.ID] = sub
	return sub, nil
}

func (r *testSubRepo) GetSubscriptionByID(_ context.Context, id uuid.UUID) (*domainSubscription.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sub, ok := r.subs[id]
	if !ok {
		return nil, domainSubscription.ErrSubscriptionNotFound
	}
	return sub, nil
}

func (r *testSubRepo) ListSubscriptionsByUserID(_ context.Context, userID uuid.UUID) ([]*domainSubscription.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainSubscription.Subscription
	for _, s := range r.subs {
		if s.UserID == userID {
			res = append(res, s)
		}
	}
	return res, nil
}

func (r *testSubRepo) ListAllSubscriptions(_ context.Context) ([]*domainSubscription.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainSubscription.Subscription
	for _, s := range r.subs {
		res = append(res, s)
	}
	return res, nil
}

func (r *testSubRepo) ListDueSubscriptions(_ context.Context, _ time.Time, _ int) ([]*domainSubscription.Subscription, error) {
	return nil, nil
}

func (r *testSubRepo) ListGraceExpiredSubscriptions(_ context.Context, _ time.Time, _ int) ([]*domainSubscription.Subscription, error) {
	return nil, nil
}

func (r *testSubRepo) UpdateSubscriptionStatus(_ context.Context, id uuid.UUID, status domainSubscription.SubscriptionStatus) (*domainSubscription.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sub, ok := r.subs[id]
	if !ok {
		return nil, domainSubscription.ErrSubscriptionNotFound
	}
	sub.Status = status
	return sub, nil
}

func (r *testSubRepo) RenewSubscription(_ context.Context, id uuid.UUID, periodStart, periodEnd, nextDue time.Time) (*domainSubscription.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sub, ok := r.subs[id]
	if !ok {
		return nil, domainSubscription.ErrSubscriptionNotFound
	}
	sub.CurrentPeriodStart = &periodStart
	sub.CurrentPeriodEnd = &periodEnd
	sub.NextDueAt = &nextDue
	return sub, nil
}

func (r *testSubRepo) SetCancelAtPeriodEnd(_ context.Context, id uuid.UUID, cancel bool) (*domainSubscription.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sub, ok := r.subs[id]
	if !ok {
		return nil, domainSubscription.ErrSubscriptionNotFound
	}
	sub.CancelAtPeriodEnd = cancel
	return sub, nil
}

// In-memory infrastructure repository
type testInfraRepo struct {
	mu           sync.Mutex
	providers    map[uuid.UUID]*domainInfrastructure.Provider
	nodeGroups   map[uuid.UUID]*domainInfrastructure.NodeGroup
	nodes        map[uuid.UUID]*domainInfrastructure.Node
	reservations map[uuid.UUID]*domainInfrastructure.ResourceReservation
	instances    map[uuid.UUID]*domainInfrastructure.Instance
}

func newTestInfraRepo() *testInfraRepo {
	return &testInfraRepo{
		providers:    make(map[uuid.UUID]*domainInfrastructure.Provider),
		nodeGroups:   make(map[uuid.UUID]*domainInfrastructure.NodeGroup),
		nodes:        make(map[uuid.UUID]*domainInfrastructure.Node),
		reservations: make(map[uuid.UUID]*domainInfrastructure.ResourceReservation),
		instances:    make(map[uuid.UUID]*domainInfrastructure.Instance),
	}
}

func (r *testInfraRepo) CreateProvider(_ context.Context, p *domainInfrastructure.Provider) (*domainInfrastructure.Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.ID] = p
	return p, nil
}

func (r *testInfraRepo) GetProviderByID(_ context.Context, id uuid.UUID) (*domainInfrastructure.Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.providers[id]
	if !ok {
		return nil, domainInfrastructure.ErrProviderNotFound
	}
	return p, nil
}

func (r *testInfraRepo) ListProviders(_ context.Context) ([]*domainInfrastructure.Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainInfrastructure.Provider
	for _, p := range r.providers {
		res = append(res, p)
	}
	return res, nil
}

func (r *testInfraRepo) UpdateProviderHealth(_ context.Context, id uuid.UUID, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.providers[id]; ok {
		p.Status = status
	}
	return nil
}

func (r *testInfraRepo) CreateNodeGroup(_ context.Context, ng *domainInfrastructure.NodeGroup) (*domainInfrastructure.NodeGroup, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodeGroups[ng.ID] = ng
	return ng, nil
}

func (r *testInfraRepo) GetNodeGroupByID(_ context.Context, id uuid.UUID) (*domainInfrastructure.NodeGroup, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ng, ok := r.nodeGroups[id]
	if !ok {
		return nil, domainInfrastructure.ErrNodeGroupNotFound
	}
	return ng, nil
}

func (r *testInfraRepo) ListNodeGroups(_ context.Context) ([]*domainInfrastructure.NodeGroup, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainInfrastructure.NodeGroup
	for _, ng := range r.nodeGroups {
		res = append(res, ng)
	}
	return res, nil
}

func (r *testInfraRepo) CreateNode(_ context.Context, node *domainInfrastructure.Node) (*domainInfrastructure.Node, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodes[node.ID] = node
	return node, nil
}

func (r *testInfraRepo) GetNodeByID(_ context.Context, id uuid.UUID) (*domainInfrastructure.Node, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.nodes[id]
	if !ok {
		return nil, domainInfrastructure.ErrNodeNotFound
	}
	return n, nil
}

func (r *testInfraRepo) ListNodes(_ context.Context) ([]*domainInfrastructure.Node, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainInfrastructure.Node
	for _, n := range r.nodes {
		res = append(res, n)
	}
	return res, nil
}

func (r *testInfraRepo) ListActiveNodes(_ context.Context) ([]*domainInfrastructure.Node, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainInfrastructure.Node
	for _, n := range r.nodes {
		if n.Status == "active" {
			res = append(res, n)
		}
	}
	return res, nil
}

func (r *testInfraRepo) ListNodesByNodeGroup(_ context.Context, nodeGroupID uuid.UUID) ([]*domainInfrastructure.Node, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainInfrastructure.Node
	for _, n := range r.nodes {
		if n.NodeGroupID != nil && *n.NodeGroupID == nodeGroupID && n.Status == "active" {
			res = append(res, n)
		}
	}
	return res, nil
}

func (r *testInfraRepo) ReserveResources(_ context.Context, res *domainInfrastructure.ResourceReservation) (*domainInfrastructure.ResourceReservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	node, ok := r.nodes[res.NodeID]
	if !ok {
		return nil, domainInfrastructure.ErrNodeNotFound
	}
	if node.AvailableCPU() < res.CPUCores || node.AvailableMemoryMB() < res.MemoryMB || node.AvailableDiskGB() < res.DiskGB {
		return nil, domainInfrastructure.ErrResourceExhausted
	}
	node.CPUReserved += res.CPUCores
	node.MemoryReservedMB += res.MemoryMB
	node.DiskReservedGB += res.DiskGB
	r.reservations[res.ID] = res
	return res, nil
}

func (r *testInfraRepo) GetReservationByID(_ context.Context, id uuid.UUID) (*domainInfrastructure.ResourceReservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.reservations[id]
	if !ok {
		return nil, domainInfrastructure.ErrReservationNotFound
	}
	return res, nil
}

func (r *testInfraRepo) CommitReservation(_ context.Context, reservationID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.reservations[reservationID]
	if !ok {
		return domainInfrastructure.ErrReservationNotFound
	}
	node, ok := r.nodes[res.NodeID]
	if !ok {
		return domainInfrastructure.ErrNodeNotFound
	}
	node.CPUReserved -= res.CPUCores
	node.MemoryReservedMB -= res.MemoryMB
	node.DiskReservedGB -= res.DiskGB
	node.CPUAllocated += res.CPUCores
	node.MemoryAllocatedMB += res.MemoryMB
	node.DiskAllocatedGB += res.DiskGB
	res.Status = "committed"
	return nil
}

func (r *testInfraRepo) ReleaseReservation(_ context.Context, reservationID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.reservations[reservationID]
	if !ok {
		return domainInfrastructure.ErrReservationNotFound
	}
	node, ok := r.nodes[res.NodeID]
	if !ok {
		return domainInfrastructure.ErrNodeNotFound
	}
	node.CPUReserved -= res.CPUCores
	node.MemoryReservedMB -= res.MemoryMB
	node.DiskReservedGB -= res.DiskGB
	res.Status = "released"
	return nil
}

func (r *testInfraRepo) CreateInstance(_ context.Context, inst *domainInfrastructure.Instance) (*domainInfrastructure.Instance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instances[inst.ID] = inst
	return inst, nil
}

func (r *testInfraRepo) GetInstanceByID(_ context.Context, id uuid.UUID) (*domainInfrastructure.Instance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	inst, ok := r.instances[id]
	if !ok {
		return nil, domainInfrastructure.ErrInstanceNotFound
	}
	return inst, nil
}

func (r *testInfraRepo) GetInstanceBySubscriptionID(_ context.Context, subID uuid.UUID) (*domainInfrastructure.Instance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, inst := range r.instances {
		if inst.SubscriptionID == subID {
			return inst, nil
		}
	}
	return nil, domainInfrastructure.ErrInstanceNotFound
}

func (r *testInfraRepo) UpdateInstanceStates(_ context.Context, id uuid.UUID, desiredState, observedState string, provInstID *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	inst, ok := r.instances[id]
	if !ok {
		return domainInfrastructure.ErrInstanceNotFound
	}
	inst.DesiredState = desiredState
	inst.ObservedState = observedState
	if provInstID != nil {
		inst.ProviderInstanceID = provInstID
	}
	return nil
}

func (r *testInfraRepo) UpdateNodeStatus(_ context.Context, id uuid.UUID, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n, ok := r.nodes[id]; ok {
		n.Status = status
	}
	return nil
}

func (r *testInfraRepo) ListExpiredReservations(_ context.Context) ([]*domainInfrastructure.ResourceReservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainInfrastructure.ResourceReservation
	for _, resv := range r.reservations {
		if resv.Status == "active" {
			res = append(res, resv)
		}
	}
	return res, nil
}

func (r *testInfraRepo) ListInstances(_ context.Context) ([]*domainInfrastructure.Instance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainInfrastructure.Instance
	for _, inst := range r.instances {
		res = append(res, inst)
	}
	return res, nil
}

// In-memory operation repository
type testOpRepo struct {
	mu         sync.Mutex
	operations map[uuid.UUID]*domainOperation.Operation
	steps      map[uuid.UUID][]*domainOperation.OperationStep
}

func newTestOpRepo() *testOpRepo {
	return &testOpRepo{
		operations: make(map[uuid.UUID]*domainOperation.Operation),
		steps:      make(map[uuid.UUID][]*domainOperation.OperationStep),
	}
}

func (r *testOpRepo) CreateOperation(_ context.Context, op *domainOperation.Operation) (*domainOperation.Operation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.operations[op.ID] = op
	return op, nil
}

func (r *testOpRepo) GetOperationByID(_ context.Context, id uuid.UUID) (*domainOperation.Operation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	op, ok := r.operations[id]
	if !ok {
		return nil, domainOperation.ErrOperationNotFound
	}
	op.Steps = r.steps[id]
	return op, nil
}

func (r *testOpRepo) GetOperationByIdempotencyKey(_ context.Context, key string) (*domainOperation.Operation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, op := range r.operations {
		if op.IdempotencyKey == key {
			op.Steps = r.steps[op.ID]
			return op, nil
		}
	}
	return nil, domainOperation.ErrOperationNotFound
}

func (r *testOpRepo) ListOperationsByResource(_ context.Context, resourceType string, resourceID uuid.UUID) ([]*domainOperation.Operation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainOperation.Operation
	for _, op := range r.operations {
		if op.ResourceType == resourceType && op.ResourceID == resourceID {
			res = append(res, op)
		}
	}
	return res, nil
}

func (r *testOpRepo) ListPendingOperations(_ context.Context, limit int) ([]*domainOperation.Operation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainOperation.Operation
	for _, op := range r.operations {
		if op.Status == domainOperation.StatusQueued || op.Status == domainOperation.StatusRetrying {
			res = append(res, op)
			if len(res) >= limit {
				break
			}
		}
	}
	return res, nil
}

func (r *testOpRepo) ListRecentOperations(_ context.Context, limit int) ([]*domainOperation.Operation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainOperation.Operation
	for _, op := range r.operations {
		res = append(res, op)
		if limit > 0 && len(res) >= limit {
			break
		}
	}
	return res, nil
}

func (r *testOpRepo) ListStuckOperations(_ context.Context, olderThan time.Time) ([]*domainOperation.Operation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domainOperation.Operation
	for _, op := range r.operations {
		if op.Status == domainOperation.StatusRunning && op.StartedAt != nil && op.StartedAt.Before(olderThan) {
			res = append(res, op)
		}
	}
	return res, nil
}

func (r *testOpRepo) UpdateOperationProgress(_ context.Context, id uuid.UUID, status domainOperation.Status, phase, messageKey string, progress int, startedAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	op, ok := r.operations[id]
	if !ok {
		return domainOperation.ErrOperationNotFound
	}
	op.Status = status
	op.Phase = &phase
	op.MessageKey = &messageKey
	op.Progress = progress
	if startedAt != nil {
		op.StartedAt = startedAt
	}
	return nil
}

func (r *testOpRepo) CompleteOperation(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	op, ok := r.operations[id]
	if !ok {
		return domainOperation.ErrOperationNotFound
	}
	op.Status = domainOperation.StatusSucceeded
	op.Progress = 100
	now := time.Now().UTC()
	op.FinishedAt = &now
	return nil
}

func (r *testOpRepo) FailOperation(_ context.Context, id uuid.UUID, errorCode, errorMessage string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	op, ok := r.operations[id]
	if !ok {
		return domainOperation.ErrOperationNotFound
	}
	op.Status = domainOperation.StatusFailed
	op.ErrorCode = &errorCode
	op.ErrorMessage = &errorMessage
	now := time.Now().UTC()
	op.FinishedAt = &now
	return nil
}

func (r *testOpRepo) SetOperationProviderInfo(_ context.Context, id uuid.UUID, providerID uuid.UUID, providerOpID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	op, ok := r.operations[id]
	if !ok {
		return domainOperation.ErrOperationNotFound
	}
	op.ProviderID = &providerID
	op.ProviderOperationID = &providerOpID
	return nil
}

func (r *testOpRepo) CreateOperationStep(_ context.Context, step *domainOperation.OperationStep) (*domainOperation.OperationStep, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.steps[step.OperationID] = append(r.steps[step.OperationID], step)
	return step, nil
}

func (r *testOpRepo) ListOperationSteps(_ context.Context, opID uuid.UUID) ([]*domainOperation.OperationStep, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.steps[opID], nil
}

func (r *testOpRepo) UpdateOperationStep(_ context.Context, opID uuid.UUID, stepKey string, status domainOperation.StepStatus, progress int, errorCode, errorMessage *string, startedAt, finishedAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.steps[opID] {
		if s.StepKey == stepKey {
			s.Status = status
			s.Progress = progress
			s.ErrorCode = errorCode
			s.ErrorMessage = errorMessage
			s.StartedAt = startedAt
			s.FinishedAt = finishedAt
			return nil
		}
	}
	return nil
}

func TestPhase6ProvisionVerticalSliceGate(t *testing.T) {
	ctx := context.Background()

	// 1. Initialize Repositories
	commerceRepo := newTestCommerceRepo()
	userRepo := newTestUserRepo()
	auditRepo := newTestAuditRepo()
	subRepo := newTestSubRepo()
	infraRepo := newTestInfraRepo()
	opRepo := newTestOpRepo()

	sessionStore := session.NewMemorySessionStore()
	sessionMgr := session.NewManager(sessionStore, false)

	auditSvc := serviceAudit.NewService(auditRepo)
	userSvc := serviceIdentity.NewUserService(userRepo, auditSvc, sessionMgr)

	productSvc := serviceCommerce.NewProductService(commerceRepo)
	orderSvc := serviceCommerce.NewOrderService(commerceRepo, commerceRepo)
	fakeGateway := serviceCommerce.NewFakePaymentGateway("")
	paymentSvc := serviceCommerce.NewPaymentService(commerceRepo, commerceRepo, fakeGateway)
	walletSvc := serviceCommerce.NewWalletService(commerceRepo, commerceRepo)
	subSvc := serviceSubscription.NewSubscriptionService(subRepo, commerceRepo)
	opSvc := serviceOperation.NewService(opRepo, nil)

	// 2. Setup Mock Provider, Node, and Scheduler
	provID := uuid.New()
	mockProv := mock.NewMockProvider("mock-default")
	_, err := infraRepo.CreateProvider(ctx, &domainInfrastructure.Provider{
		ID:           provID,
		Name:         "Default Mock Provider",
		ProviderType: "mock",
		Status:       "active",
		Config:       []byte("{}"),
		Capabilities: []byte(`{"create_instance":true}`),
	})
	require.NoError(t, err)

	nodeID := uuid.New()
	_, err = infraRepo.CreateNode(ctx, &domainInfrastructure.Node{
		ID:            nodeID,
		ProviderID:    provID,
		Name:          "node-us-west-1",
		Region:        "us-west",
		Status:        "active",
		CPUTotal:      64,
		MemoryTotalMB: 262144,
		DiskTotalGB:   4000,
		Weight:        100,
	})
	require.NoError(t, err)

	providersMap := map[string]provider.Provider{
		provID.String(): mockProv,
		"default":       mockProv,
	}
	sched := serviceScheduler.NewScheduler(infraRepo)
	provWf := workflow.NewProvisionWorkflow(infraRepo, subRepo, commerceRepo, sched, providersMap, opSvc)
	fulfillSvc := serviceFulfillment.NewFulfillmentService(commerceRepo, subRepo, commerceRepo, opSvc, provWf)
	paymentSvc.SetFulfiller(fulfillSvc)

	// 3. Seed Product & Plan
	prodID := uuid.New()
	planID := uuid.New()
	_, err = productSvc.CreateProduct(ctx, &domainCommerce.Product{
		ID:              prodID,
		Slug:            "cloud-vps",
		NameI18n:        []byte(`{"en-US":"Cloud VPS","zh-CN":"云服务器"}`),
		DescriptionI18n: []byte(`{}`),
		Status:          "active",
		SortOrder:       1,
	})
	require.NoError(t, err)

	_, err = productSvc.CreatePlan(ctx, &domainCommerce.Plan{
		ID:             planID,
		ProductID:      prodID,
		Slug:           "starter",
		NameI18n:       []byte(`{"en-US":"Starter","zh-CN":"入门型"}`),
		Status:         "active",
		CPUCores:       2,
		MemoryMB:       4096,
		DiskGB:         50,
		IPv4Count:      1,
		IPv6Count:      1,
		NatPortCount:   0,
		Virtualization: "kvm",
		BillingCycle:   "monthly",
		PriceMinor:     1000,
		Currency:       "USD",
		StockMode:      "automatic",
	})
	require.NoError(t, err)

	// 4. Setup Router
	cfg := &config.Config{
		AppEnv:        "testing",
		HTTPPort:      "8080",
		UserWebOrigin: "http://localhost:3000",
	}

	router := server.NewRouterWithDeps(server.RouterDeps{
		Config:          cfg,
		UserSvc:         userSvc,
		AuditSvc:        auditSvc,
		SessionMgr:      sessionMgr,
		ProductSvc:      productSvc,
		OrderSvc:        orderSvc,
		PaymentSvc:      paymentSvc,
		WalletSvc:       walletSvc,
		SubscriptionSvc: subSvc,
		OperationSvc:    opSvc,
	})

	// STEP 1: 浏览 (Browse Catalog)
	t.Log("Step 1: 浏览产品目录")
	browseReq := httptest.NewRequest("GET", "/api/v1/products", nil)
	browseRec := httptest.NewRecorder()
	router.ServeHTTP(browseRec, browseReq)
	require.Equal(t, http.StatusOK, browseRec.Code)

	var browseResp struct {
		Data struct {
			Products []struct {
				ID    string `json:"id"`
				Plans []struct {
					ID string `json:"id"`
				} `json:"plans"`
			} `json:"products"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(browseRec.Body.Bytes(), &browseResp))
	require.NotEmpty(t, browseResp.Data.Products)
	require.NotEmpty(t, browseResp.Data.Products[0].Plans)

	// Register test user
	regBody, _ := json.Marshal(map[string]string{
		"email":    "buyer@example.com",
		"password": "Password123!",
	})
	regReq := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	router.ServeHTTP(regRec, regReq)
	require.Equal(t, http.StatusCreated, regRec.Code)

	// Login test user to get session & csrf cookies
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "buyer@example.com",
		"password": "Password123!",
	})
	loginReq := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)

	sessionCookie := ""
	csrfToken := ""
	for _, cookie := range loginRec.Result().Cookies() {
		if cookie.Name == session.UserSessionCookieName {
			sessionCookie = cookie.Value
		}
		if cookie.Name == session.CSRFCookieName {
			csrfToken = cookie.Value
		}
	}
	require.NotEmpty(t, sessionCookie)

	// STEP 2: 下单 (Create Order)
	t.Log("Step 2: 创建订单")
	orderPayload, _ := json.Marshal(map[string]any{
		"items": []map[string]any{
			{
				"plan_id":  planID.String(),
				"quantity": 1,
			},
		},
	})
	orderReq := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewReader(orderPayload))
	orderReq.Header.Set("Content-Type", "application/json")
	orderReq.Header.Set("Cookie", fmt.Sprintf("%s=%s; %s=%s", session.UserSessionCookieName, sessionCookie, session.CSRFCookieName, csrfToken))
	orderReq.Header.Set("X-CSRF-Token", csrfToken)
	orderRec := httptest.NewRecorder()
	router.ServeHTTP(orderRec, orderReq)
	require.Equal(t, http.StatusCreated, orderRec.Code)

	var orderResp struct {
		Data struct {
			Order struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"order"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(orderRec.Body.Bytes(), &orderResp))
	orderIDStr := orderResp.Data.Order.ID
	assert.Equal(t, "pending", orderResp.Data.Order.Status)

	// STEP 3: 发起支付 (Initiate Payment)
	t.Log("Step 3: 发起假支付")
	payReq := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/orders/%s/pay", orderIDStr), bytes.NewReader([]byte(`{"gateway":"fake"}`)))
	payReq.Header.Set("Content-Type", "application/json")
	payReq.Header.Set("Cookie", fmt.Sprintf("%s=%s; %s=%s", session.UserSessionCookieName, sessionCookie, session.CSRFCookieName, csrfToken))
	payReq.Header.Set("X-CSRF-Token", csrfToken)
	payRec := httptest.NewRecorder()
	router.ServeHTTP(payRec, payReq)
	require.Equal(t, http.StatusOK, payRec.Code)

	var payResp struct {
		Data struct {
			Payment struct {
				PaymentNo string `json:"payment_no"`
			} `json:"payment"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(payRec.Body.Bytes(), &payResp))
	paymentNo := payResp.Data.Payment.PaymentNo
	require.NotEmpty(t, paymentNo)

	// STEP 4: 支付回调模拟 (Payment Simulation Webhook)
	t.Log("Step 4: 支付回调入账")
	simBody, _ := json.Marshal(map[string]string{"payment_no": paymentNo})
	simReq := httptest.NewRequest("POST", "/api/v1/payments/fake/simulate", bytes.NewReader(simBody))
	simReq.Header.Set("Content-Type", "application/json")
	simRec := httptest.NewRecorder()
	router.ServeHTTP(simRec, simReq)
	require.Equal(t, http.StatusOK, simRec.Code)

	// STEP 5: 自动开通与进度验证 (Automatic Provisioning & Progress Verification)
	t.Log("Step 5: 验证自动开通全流程")

	// Wait briefly for the goroutine execution to finish
	time.Sleep(300 * time.Millisecond)

	// Verify order is paid
	orderUUID, _ := uuid.Parse(orderIDStr)
	dbOrder, err := commerceRepo.GetOrderByID(ctx, orderUUID)
	require.NoError(t, err)
	assert.Equal(t, domainCommerce.OrderStatusPaid, dbOrder.Status)

	// Verify subscription created and active
	subs, err := subRepo.ListSubscriptionsByUserID(ctx, dbOrder.UserID)
	require.NoError(t, err)
	require.Len(t, subs, 1)
	sub := subs[0]
	assert.Equal(t, domainSubscription.StatusActive, sub.Status)

	// Verify provision operation exists and succeeded
	ops, err := opRepo.ListOperationsByResource(ctx, "subscription", sub.ID)
	require.NoError(t, err)
	require.Len(t, ops, 1)
	op := ops[0]
	assert.Equal(t, domainOperation.StatusSucceeded, op.Status)
	assert.Equal(t, 100, op.Progress)

	// STEP 6: 实例运行状态验证 (Instance Running Verification)
	t.Log("Step 6: 验证底层实例与网络运行状态")
	inst, err := infraRepo.GetInstanceBySubscriptionID(ctx, sub.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", inst.DesiredState)
	assert.Equal(t, "running", inst.ObservedState)
	assert.NotNil(t, inst.PrimaryIPv4)
	assert.NotEmpty(t, *inst.PrimaryIPv4)

	// Verify MockProvider has the instance running
	provInst, err := mockProv.GetInstance(ctx, provider.GetInstanceRequest{
		ProviderInstanceID: *inst.ProviderInstanceID,
	})
	require.NoError(t, err)
	assert.Equal(t, "running", provInst.State)

	// Verify node reservation was committed (allocated resources updated)
	node, err := infraRepo.GetNodeByID(ctx, nodeID)
	require.NoError(t, err)
	assert.Equal(t, float64(2), node.CPUAllocated)
	assert.Equal(t, int64(4096), node.MemoryAllocatedMB)
	assert.Equal(t, int64(50), node.DiskAllocatedGB)
	assert.Equal(t, float64(0), node.CPUReserved)

	// STEP 7: Operation Polling API 接口验证
	t.Log("Step 7: 通过 /api/v1/operations/{id} 接口验证进度状态")
	getOpReq := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/operations/%s", op.ID.String()), nil)
	getOpRec := httptest.NewRecorder()
	router.ServeHTTP(getOpRec, getOpReq)
	require.Equal(t, http.StatusOK, getOpRec.Code)

	var opGetResp struct {
		Data struct {
			Operation struct {
				Status   string `json:"status"`
				Progress int    `json:"progress"`
				Steps    []struct {
					StepKey string `json:"step_key"`
					Status  string `json:"status"`
				} `json:"steps"`
			} `json:"operation"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(getOpRec.Body.Bytes(), &opGetResp))
	assert.Equal(t, "succeeded", opGetResp.Data.Operation.Status)
	assert.Equal(t, 100, opGetResp.Data.Operation.Progress)
	assert.NotEmpty(t, opGetResp.Data.Operation.Steps)
}

func TestPhase7GateDirectProviderReplacement(t *testing.T) {
	ctx := context.Background()

	// 1. Initialize Repositories (identical business core)
	commerceRepo := newTestCommerceRepo()
	userRepo := newTestUserRepo()
	auditRepo := newTestAuditRepo()
	subRepo := newTestSubRepo()
	infraRepo := newTestInfraRepo()
	opRepo := newTestOpRepo()

	sessionStore := session.NewMemorySessionStore()
	sessionMgr := session.NewManager(sessionStore, false)

	auditSvc := serviceAudit.NewService(auditRepo)
	userSvc := serviceIdentity.NewUserService(userRepo, auditSvc, sessionMgr)

	productSvc := serviceCommerce.NewProductService(commerceRepo)
	orderSvc := serviceCommerce.NewOrderService(commerceRepo, commerceRepo)
	fakeGateway := serviceCommerce.NewFakePaymentGateway("")
	paymentSvc := serviceCommerce.NewPaymentService(commerceRepo, commerceRepo, fakeGateway)
	walletSvc := serviceCommerce.NewWalletService(commerceRepo, commerceRepo)
	subSvc := serviceSubscription.NewSubscriptionService(subRepo, commerceRepo)
	opSvc := serviceOperation.NewService(opRepo, nil)

	// 2. Setup Real HTTP LXD Server
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/1.0" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "sync", "status": "Success", "status_code": 200,
				"metadata": map[string]any{"environment": map[string]any{"server_version": "5.21"}},
			})
			return
		}
		if r.Method == "POST" && r.URL.Path == "/1.0/instances" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "async", "status": "Operation created", "status_code": 100,
				"operation": "/1.0/operations/lxd-op-1",
			})
			return
		}
		if r.Method == "GET" && len(r.URL.Path) > len("/1.0/instances/") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "sync", "status": "Success", "status_code": 200,
				"metadata": map[string]any{"status": "Running"},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer fakeServer.Close()

	// 3. Instantiate Direct LXD Provider Adapter (REPLACING MockProvider WITHOUT TOUCHING BUSINESS CORE)
	lxdAdapter := lxdapi.NewAdapter("lxd-direct", lxdapi.Config{
		Endpoint: fakeServer.URL,
		Token:    "secret-token",
	}, fakeServer.Client())

	provID := uuid.New()
	_, err := infraRepo.CreateProvider(ctx, &domainInfrastructure.Provider{
		ID:           provID,
		Name:         "LXD Direct Node",
		ProviderType: "direct",
		Status:       "active",
		Config:       []byte("{}"),
		Capabilities: []byte(`{"create_instance":true}`),
	})
	require.NoError(t, err)

	nodeID := uuid.New()
	_, err = infraRepo.CreateNode(ctx, &domainInfrastructure.Node{
		ID:            nodeID,
		ProviderID:    provID,
		Name:          "node-lxd-1",
		Region:        "us-east",
		Status:        "active",
		CPUTotal:      128,
		MemoryTotalMB: 524288,
		DiskTotalGB:   8000,
		Weight:        100,
	})
	require.NoError(t, err)

	providersMap := map[string]provider.Provider{
		provID.String(): lxdAdapter,
		"default":       lxdAdapter,
	}
	sched := serviceScheduler.NewScheduler(infraRepo)

	// Business workflows receive the Direct Provider seamlessly
	provWf := workflow.NewProvisionWorkflow(infraRepo, subRepo, commerceRepo, sched, providersMap, opSvc)
	fulfillSvc := serviceFulfillment.NewFulfillmentService(commerceRepo, subRepo, commerceRepo, opSvc, provWf)
	paymentSvc.SetFulfiller(fulfillSvc)

	// Seed Catalog
	prodID := uuid.New()
	planID := uuid.New()
	_, err = productSvc.CreateProduct(ctx, &domainCommerce.Product{
		ID:              prodID,
		Slug:            "lxd-vps",
		NameI18n:        []byte(`{"en-US":"LXD VPS","zh-CN":"LXD 云服务器"}`),
		DescriptionI18n: []byte(`{}`),
		Status:          "active",
	})
	require.NoError(t, err)

	_, err = productSvc.CreatePlan(ctx, &domainCommerce.Plan{
		ID:             planID,
		ProductID:      prodID,
		Slug:           "lxd-starter",
		NameI18n:       []byte(`{"en-US":"LXD Starter","zh-CN":"LXD 入门"}`),
		Status:         "active",
		CPUCores:       2,
		MemoryMB:       2048,
		DiskGB:         20,
		IPv4Count:      1,
		IPv6Count:      1,
		Virtualization: "lxc",
		BillingCycle:   "monthly",
		PriceMinor:     800,
		Currency:       "USD",
	})
	require.NoError(t, err)

	router := server.NewRouterWithDeps(server.RouterDeps{
		Config:          &config.Config{AppEnv: "testing", HTTPPort: "8080", UserWebOrigin: "http://localhost:3000"},
		UserSvc:         userSvc,
		AuditSvc:        auditSvc,
		SessionMgr:      sessionMgr,
		ProductSvc:      productSvc,
		OrderSvc:        orderSvc,
		PaymentSvc:      paymentSvc,
		WalletSvc:       walletSvc,
		SubscriptionSvc: subSvc,
		OperationSvc:    opSvc,
	})

	// User registers & logs in
	regBody, _ := json.Marshal(map[string]string{"email": "lxduser@example.com", "password": "Password123!"})
	regReq := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	router.ServeHTTP(regRec, regReq)

	loginReq := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(regBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	sessionCookie := ""
	csrfToken := ""
	for _, cookie := range loginRec.Result().Cookies() {
		if cookie.Name == session.UserSessionCookieName {
			sessionCookie = cookie.Value
		}
		if cookie.Name == session.CSRFCookieName {
			csrfToken = cookie.Value
		}
	}
	require.NotEmpty(t, sessionCookie)

	// Order & Pay
	orderPayload, _ := json.Marshal(map[string]any{"items": []map[string]any{{"plan_id": planID.String(), "quantity": 1}}})
	orderReq := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewReader(orderPayload))
	orderReq.Header.Set("Content-Type", "application/json")
	orderReq.Header.Set("Cookie", fmt.Sprintf("%s=%s; %s=%s", session.UserSessionCookieName, sessionCookie, session.CSRFCookieName, csrfToken))
	orderReq.Header.Set("X-CSRF-Token", csrfToken)
	orderRec := httptest.NewRecorder()
	router.ServeHTTP(orderRec, orderReq)
	require.Equal(t, http.StatusCreated, orderRec.Code)

	var ordResp struct {
		Data struct {
			Order struct {
				ID string `json:"id"`
			} `json:"order"`
		} `json:"data"`
	}
	_ = json.Unmarshal(orderRec.Body.Bytes(), &ordResp)

	payReq := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/orders/%s/pay", ordResp.Data.Order.ID), bytes.NewReader([]byte(`{"gateway":"fake"}`)))
	payReq.Header.Set("Content-Type", "application/json")
	payReq.Header.Set("Cookie", fmt.Sprintf("%s=%s; %s=%s", session.UserSessionCookieName, sessionCookie, session.CSRFCookieName, csrfToken))
	payReq.Header.Set("X-CSRF-Token", csrfToken)
	payRec := httptest.NewRecorder()
	router.ServeHTTP(payRec, payReq)

	var payResp struct {
		Data struct {
			Payment struct {
				PaymentNo string `json:"payment_no"`
			} `json:"payment"`
		} `json:"data"`
	}
	_ = json.Unmarshal(payRec.Body.Bytes(), &payResp)

	simReq := httptest.NewRequest("POST", "/api/v1/payments/fake/simulate", bytes.NewReader([]byte(fmt.Sprintf(`{"payment_no":"%s"}`, payResp.Data.Payment.PaymentNo))))
	simReq.Header.Set("Content-Type", "application/json")
	simRec := httptest.NewRecorder()
	router.ServeHTTP(simRec, simReq)
	require.Equal(t, http.StatusOK, simRec.Code)

	time.Sleep(300 * time.Millisecond)

	// Verify that with Direct LXD Provider:
	// Subscription became active
	userUUID, _ := uuid.Parse(regRec.Header().Get("X-User-ID"))
	_ = userUUID
	subs, err := subRepo.ListAllSubscriptions(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, subs)
	assert.Equal(t, domainSubscription.StatusActive, subs[len(subs)-1].Status)

	// Instance created in DB and observed running
	inst, err := infraRepo.GetInstanceBySubscriptionID(ctx, subs[len(subs)-1].ID)
	require.NoError(t, err)
	assert.Equal(t, "running", inst.DesiredState)
	assert.Equal(t, "running", inst.ObservedState)
}
