package reconciler_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	domainInfrastructure "vps-billing/internal/domain/infrastructure"
	domainOperation "vps-billing/internal/domain/operation"
	"vps-billing/internal/provider"
	"vps-billing/internal/provider/mock"
	"vps-billing/internal/reconciler"
	serviceOperation "vps-billing/internal/service/operation"
)

// In-memory mock repositories for test isolation
type mockInfraRepo struct {
	nodes        map[uuid.UUID]*domainInfrastructure.Node
	reservations map[uuid.UUID]*domainInfrastructure.ResourceReservation
	instances    map[uuid.UUID]*domainInfrastructure.Instance
}

func newMockInfraRepo() *mockInfraRepo {
	return &mockInfraRepo{
		nodes:        make(map[uuid.UUID]*domainInfrastructure.Node),
		reservations: make(map[uuid.UUID]*domainInfrastructure.ResourceReservation),
		instances:    make(map[uuid.UUID]*domainInfrastructure.Instance),
	}
}

func (m *mockInfraRepo) CreateProvider(ctx context.Context, p *domainInfrastructure.Provider) (*domainInfrastructure.Provider, error) {
	return p, nil
}
func (m *mockInfraRepo) GetProviderByID(ctx context.Context, id uuid.UUID) (*domainInfrastructure.Provider, error) {
	return nil, nil
}
func (m *mockInfraRepo) ListProviders(ctx context.Context) ([]*domainInfrastructure.Provider, error) {
	return nil, nil
}
func (m *mockInfraRepo) UpdateProviderHealth(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (m *mockInfraRepo) CreateNodeGroup(ctx context.Context, ng *domainInfrastructure.NodeGroup) (*domainInfrastructure.NodeGroup, error) {
	return ng, nil
}
func (m *mockInfraRepo) GetNodeGroupByID(ctx context.Context, id uuid.UUID) (*domainInfrastructure.NodeGroup, error) {
	return nil, nil
}
func (m *mockInfraRepo) ListNodeGroups(ctx context.Context) ([]*domainInfrastructure.NodeGroup, error) {
	return nil, nil
}
func (m *mockInfraRepo) CreateNode(ctx context.Context, n *domainInfrastructure.Node) (*domainInfrastructure.Node, error) {
	m.nodes[n.ID] = n
	return n, nil
}
func (m *mockInfraRepo) GetNodeByID(ctx context.Context, id uuid.UUID) (*domainInfrastructure.Node, error) {
	return m.nodes[id], nil
}
func (m *mockInfraRepo) ListNodes(ctx context.Context) ([]*domainInfrastructure.Node, error) {
	var list []*domainInfrastructure.Node
	for _, n := range m.nodes {
		list = append(list, n)
	}
	return list, nil
}
func (m *mockInfraRepo) ListActiveNodes(ctx context.Context) ([]*domainInfrastructure.Node, error) {
	var list []*domainInfrastructure.Node
	for _, n := range m.nodes {
		if n.Status == "active" {
			list = append(list, n)
		}
	}
	return list, nil
}
func (m *mockInfraRepo) ListNodesByNodeGroup(ctx context.Context, gid uuid.UUID) ([]*domainInfrastructure.Node, error) {
	return nil, nil
}
func (m *mockInfraRepo) UpdateNodeStatus(ctx context.Context, id uuid.UUID, status string) error {
	if n, ok := m.nodes[id]; ok {
		n.Status = status
	}
	return nil
}
func (m *mockInfraRepo) ReserveResources(ctx context.Context, res *domainInfrastructure.ResourceReservation) (*domainInfrastructure.ResourceReservation, error) {
	m.reservations[res.ID] = res
	return res, nil
}
func (m *mockInfraRepo) GetReservationByID(ctx context.Context, id uuid.UUID) (*domainInfrastructure.ResourceReservation, error) {
	return m.reservations[id], nil
}
func (m *mockInfraRepo) CommitReservation(ctx context.Context, id uuid.UUID) error {
	if r, ok := m.reservations[id]; ok {
		r.Status = "committed"
	}
	return nil
}
func (m *mockInfraRepo) ReleaseReservation(ctx context.Context, id uuid.UUID) error {
	if r, ok := m.reservations[id]; ok {
		r.Status = "released"
	}
	return nil
}
func (m *mockInfraRepo) ListExpiredReservations(ctx context.Context) ([]*domainInfrastructure.ResourceReservation, error) {
	var list []*domainInfrastructure.ResourceReservation
	now := time.Now().UTC()
	for _, r := range m.reservations {
		if r.Status == "reserved" && r.ExpiresAt.Before(now) {
			list = append(list, r)
		}
	}
	return list, nil
}
func (m *mockInfraRepo) CreateInstance(ctx context.Context, inst *domainInfrastructure.Instance) (*domainInfrastructure.Instance, error) {
	m.instances[inst.ID] = inst
	return inst, nil
}
func (m *mockInfraRepo) GetInstanceByID(ctx context.Context, id uuid.UUID) (*domainInfrastructure.Instance, error) {
	return m.instances[id], nil
}
func (m *mockInfraRepo) GetInstanceBySubscriptionID(ctx context.Context, sid uuid.UUID) (*domainInfrastructure.Instance, error) {
	for _, i := range m.instances {
		if i.SubscriptionID == sid {
			return i, nil
		}
	}
	return nil, domainInfrastructure.ErrInstanceNotFound
}
func (m *mockInfraRepo) ListInstances(ctx context.Context) ([]*domainInfrastructure.Instance, error) {
	var list []*domainInfrastructure.Instance
	for _, i := range m.instances {
		list = append(list, i)
	}
	return list, nil
}
func (m *mockInfraRepo) UpdateInstanceStates(ctx context.Context, id uuid.UUID, desiredState, observedState string, provInstID *string) error {
	if inst, ok := m.instances[id]; ok {
		inst.DesiredState = desiredState
		inst.ObservedState = observedState
		if provInstID != nil {
			inst.ProviderInstanceID = provInstID
		}
	}
	return nil
}

type mockOpRepo struct {
	ops map[uuid.UUID]*domainOperation.Operation
}

func newMockOpRepo() *mockOpRepo {
	return &mockOpRepo{ops: make(map[uuid.UUID]*domainOperation.Operation)}
}

func (m *mockOpRepo) CreateOperation(ctx context.Context, op *domainOperation.Operation) (*domainOperation.Operation, error) {
	m.ops[op.ID] = op
	return op, nil
}
func (m *mockOpRepo) GetOperationByID(ctx context.Context, id uuid.UUID) (*domainOperation.Operation, error) {
	return m.ops[id], nil
}
func (m *mockOpRepo) GetOperationByIdempotencyKey(ctx context.Context, key string) (*domainOperation.Operation, error) {
	return nil, nil
}
func (m *mockOpRepo) ListOperationsByResource(ctx context.Context, rType string, rID uuid.UUID) ([]*domainOperation.Operation, error) {
	return nil, nil
}
func (m *mockOpRepo) ListPendingOperations(ctx context.Context, limit int) ([]*domainOperation.Operation, error) {
	return nil, nil
}
func (m *mockOpRepo) ListRecentOperations(ctx context.Context, limit int) ([]*domainOperation.Operation, error) {
	return nil, nil
}
func (m *mockOpRepo) ListStuckOperations(ctx context.Context, olderThan time.Time) ([]*domainOperation.Operation, error) {
	var list []*domainOperation.Operation
	for _, op := range m.ops {
		if op.Status == domainOperation.StatusRunning && op.UpdatedAt.Before(olderThan) {
			list = append(list, op)
		}
	}
	return list, nil
}
func (m *mockOpRepo) UpdateOperationProgress(ctx context.Context, id uuid.UUID, status domainOperation.Status, phase, msgKey string, progress int, startedAt *time.Time) error {
	if op, ok := m.ops[id]; ok {
		op.Status = status
		op.Phase = &phase
		op.Progress = progress
		op.UpdatedAt = time.Now().UTC()
	}
	return nil
}
func (m *mockOpRepo) CompleteOperation(ctx context.Context, id uuid.UUID) error {
	if op, ok := m.ops[id]; ok {
		op.Status = domainOperation.StatusSucceeded
		op.UpdatedAt = time.Now().UTC()
	}
	return nil
}
func (m *mockOpRepo) FailOperation(ctx context.Context, id uuid.UUID, code, msg string) error {
	if op, ok := m.ops[id]; ok {
		op.Status = domainOperation.StatusFailed
		op.ErrorCode = &code
		op.ErrorMessage = &msg
		op.UpdatedAt = time.Now().UTC()
	}
	return nil
}
func (m *mockOpRepo) SetOperationProviderInfo(ctx context.Context, id, provID uuid.UUID, provOpID string) error {
	return nil
}
func (m *mockOpRepo) CreateOperationStep(ctx context.Context, s *domainOperation.OperationStep) (*domainOperation.OperationStep, error) {
	return s, nil
}
func (m *mockOpRepo) ListOperationSteps(ctx context.Context, opID uuid.UUID) ([]*domainOperation.OperationStep, error) {
	return nil, nil
}
func (m *mockOpRepo) UpdateOperationStep(ctx context.Context, opID uuid.UUID, stepKey string, status domainOperation.StepStatus, progress int, errCode, errMsg *string, startedAt, finishedAt *time.Time) error {
	return nil
}

func TestReconcileExpiredReservations(t *testing.T) {
	ctx := context.Background()
	infraRepo := newMockInfraRepo()
	opRepo := newMockOpRepo()
	opSvc := serviceOperation.NewService(opRepo, nil)

	// Add an expired reservation
	resID := uuid.New()
	infraRepo.reservations[resID] = &domainInfrastructure.ResourceReservation{
		ID:        resID,
		NodeID:    uuid.New(),
		Status:    "reserved",
		ExpiresAt: time.Now().UTC().Add(-10 * time.Minute), // expired
	}

	rec := reconciler.NewReconciler(infraRepo, opRepo, opSvc, nil, 3*time.Minute, 1*time.Minute)
	released, err := rec.ReconcileExpiredReservations(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, released)
	assert.Equal(t, "released", infraRepo.reservations[resID].Status)
}

func TestReconcileStuckOperations(t *testing.T) {
	ctx := context.Background()
	infraRepo := newMockInfraRepo()
	opRepo := newMockOpRepo()
	opSvc := serviceOperation.NewService(opRepo, nil)

	// Add stuck operation from crashed worker
	stuckOpID := uuid.New()
	opRepo.ops[stuckOpID] = &domainOperation.Operation{
		ID:         stuckOpID,
		Type:       "provision_instance",
		Status:     domainOperation.StatusRunning,
		Retryable:  true,
		RetryCount: 0,
		MaxRetries: 3,
		Progress:   30,
		UpdatedAt:  time.Now().UTC().Add(-10 * time.Minute), // stuck!
	}

	rec := reconciler.NewReconciler(infraRepo, opRepo, opSvc, nil, 3*time.Minute, 1*time.Minute)
	recovered, err := rec.ReconcileStuckOperations(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, recovered)

	// Stuck operation should now be transitioned back to StatusRetrying!
	assert.Equal(t, domainOperation.StatusRetrying, opRepo.ops[stuckOpID].Status)
}

func TestReconcileNodeHeartbeats(t *testing.T) {
	ctx := context.Background()
	infraRepo := newMockInfraRepo()
	opRepo := newMockOpRepo()
	opSvc := serviceOperation.NewService(opRepo, nil)

	nodeID := uuid.New()
	lastSeen := time.Now().UTC().Add(-5 * time.Minute) // missed heartbeats
	infraRepo.nodes[nodeID] = &domainInfrastructure.Node{
		ID:         nodeID,
		Name:       "stale-node",
		Status:     "active",
		LastSeenAt: &lastSeen,
	}

	instID := uuid.New()
	infraRepo.instances[instID] = &domainInfrastructure.Instance{
		ID:            instID,
		NodeID:        &nodeID,
		ObservedState: "running",
		DesiredState:  "running",
	}

	rec := reconciler.NewReconciler(infraRepo, opRepo, opSvc, nil, 3*time.Minute, 1*time.Minute)
	marked, err := rec.ReconcileNodeHeartbeats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, marked)

	// Node must be offline
	assert.Equal(t, "offline", infraRepo.nodes[nodeID].Status)

	// Instance must be marked "unknown", NEVER deleted!
	assert.Equal(t, "unknown", infraRepo.instances[instID].ObservedState)
}

func TestCreateSuccessButTimeoutAdoption(t *testing.T) {
	ctx := context.Background()
	infraRepo := newMockInfraRepo()
	opRepo := newMockOpRepo()
	opSvc := serviceOperation.NewService(opRepo, nil)

	mockProv := mock.NewMockProvider("mock-prov")
	providersMap := map[string]provider.Provider{
		"default": mockProv,
	}

	instID := uuid.New()
	// Pre-create on provider as if network timed out after create succeeded
	_, err := mockProv.CreateInstance(ctx, provider.CreateInstanceRequest{
		OperationID:    uuid.New().String(),
		InstanceID:     instID.String(),
		Name:           "timeout-vm",
		Virtualization: "kvm",
	})
	require.NoError(t, err)

	rec := reconciler.NewReconciler(infraRepo, opRepo, opSvc, providersMap, 3*time.Minute, 1*time.Minute)

	// VerifyOrAdoptCreate must find the instance and adopt it without duplicate creation!
	adopted, err := rec.VerifyOrAdoptCreate(ctx, mockProv, instID, "node-1")
	require.NoError(t, err)
	require.NotNil(t, adopted)
	assert.NotEmpty(t, adopted.ProviderInstanceID)
}

func TestReconcileInstanceStates(t *testing.T) {
	ctx := context.Background()
	infraRepo := newMockInfraRepo()
	opRepo := newMockOpRepo()
	opSvc := serviceOperation.NewService(opRepo, nil)

	mockProv := mock.NewMockProvider("mock-prov")
	providersMap := map[string]provider.Provider{
		"default": mockProv,
	}

	instID := uuid.New()
	_, err := mockProv.CreateInstance(ctx, provider.CreateInstanceRequest{
		OperationID:    uuid.New().String(),
		InstanceID:     instID.String(),
		Name:           "drift-vm",
		Virtualization: "kvm",
	})
	require.NoError(t, err)

	provInst, err := mockProv.GetInstance(ctx, provider.GetInstanceRequest{
		PlatformInstanceID: instID.String(),
	})
	require.NoError(t, err)

	provInstID := provInst.ProviderInstanceID
	infraRepo.instances[instID] = &domainInfrastructure.Instance{
		ID:                 instID,
		ProviderInstanceID: &provInstID,
		DesiredState:       "running",
		ObservedState:      "stopped", // drifted from provider's "running"
	}

	rec := reconciler.NewReconciler(infraRepo, opRepo, opSvc, providersMap, 3*time.Minute, 1*time.Minute)
	reconciled, err := rec.ReconcileInstanceStates(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, reconciled)

	// Observed state should be synced to provider's running state
	assert.Equal(t, "running", infraRepo.instances[instID].ObservedState)
}
