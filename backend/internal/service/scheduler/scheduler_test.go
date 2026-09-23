package scheduler_test

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	domainInfrastructure "vps-billing/internal/domain/infrastructure"
	"vps-billing/internal/service/scheduler"
)

type mockInfraRepo struct {
	mu           sync.Mutex
	nodes        []*domainInfrastructure.Node
	reservations map[uuid.UUID]*domainInfrastructure.ResourceReservation
	failNodeID   uuid.UUID // simulate race failure on this node
}

func newMockInfraRepo(nodes []*domainInfrastructure.Node) *mockInfraRepo {
	return &mockInfraRepo{
		nodes:        nodes,
		reservations: make(map[uuid.UUID]*domainInfrastructure.ResourceReservation),
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
func (m *mockInfraRepo) CreateNode(ctx context.Context, node *domainInfrastructure.Node) (*domainInfrastructure.Node, error) {
	return node, nil
}
func (m *mockInfraRepo) GetNodeByID(ctx context.Context, id uuid.UUID) (*domainInfrastructure.Node, error) {
	for _, n := range m.nodes {
		if n.ID == id {
			return n, nil
		}
	}
	return nil, domainInfrastructure.ErrNodeNotFound
}
func (m *mockInfraRepo) ListNodes(ctx context.Context) ([]*domainInfrastructure.Node, error) {
	return m.nodes, nil
}
func (m *mockInfraRepo) ListActiveNodes(ctx context.Context) ([]*domainInfrastructure.Node, error) {
	var active []*domainInfrastructure.Node
	for _, n := range m.nodes {
		if n.Status == "active" {
			active = append(active, n)
		}
	}
	return active, nil
}
func (m *mockInfraRepo) ListNodesByNodeGroup(ctx context.Context, nodeGroupID uuid.UUID) ([]*domainInfrastructure.Node, error) {
	var list []*domainInfrastructure.Node
	for _, n := range m.nodes {
		if n.NodeGroupID != nil && *n.NodeGroupID == nodeGroupID && n.Status == "active" {
			list = append(list, n)
		}
	}
	return list, nil
}
func (m *mockInfraRepo) ReserveResources(ctx context.Context, res *domainInfrastructure.ResourceReservation) (*domainInfrastructure.ResourceReservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if res.NodeID == m.failNodeID {
		return nil, domainInfrastructure.ErrResourceExhausted
	}

	for _, n := range m.nodes {
		if n.ID == res.NodeID {
			if n.AvailableCPU() < res.CPUCores || n.AvailableMemoryMB() < res.MemoryMB || n.AvailableDiskGB() < res.DiskGB {
				return nil, domainInfrastructure.ErrResourceExhausted
			}
			n.CPUReserved += res.CPUCores
			n.MemoryReservedMB += res.MemoryMB
			n.DiskReservedGB += res.DiskGB
			m.reservations[res.ID] = res
			return res, nil
		}
	}
	return nil, domainInfrastructure.ErrNodeNotFound
}
func (m *mockInfraRepo) GetReservationByID(ctx context.Context, id uuid.UUID) (*domainInfrastructure.ResourceReservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res, ok := m.reservations[id]
	if !ok {
		return nil, domainInfrastructure.ErrReservationNotFound
	}
	return res, nil
}
func (m *mockInfraRepo) CommitReservation(ctx context.Context, reservationID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	res, ok := m.reservations[reservationID]
	if !ok {
		return domainInfrastructure.ErrReservationNotFound
	}
	for _, n := range m.nodes {
		if n.ID == res.NodeID {
			n.CPUReserved -= res.CPUCores
			n.MemoryReservedMB -= res.MemoryMB
			n.DiskReservedGB -= res.DiskGB
			n.CPUAllocated += res.CPUCores
			n.MemoryAllocatedMB += res.MemoryMB
			n.DiskAllocatedGB += res.DiskGB
			res.Status = "committed"
			return nil
		}
	}
	return nil
}
func (m *mockInfraRepo) ReleaseReservation(ctx context.Context, reservationID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	res, ok := m.reservations[reservationID]
	if !ok {
		return domainInfrastructure.ErrReservationNotFound
	}
	for _, n := range m.nodes {
		if n.ID == res.NodeID {
			n.CPUReserved -= res.CPUCores
			n.MemoryReservedMB -= res.MemoryMB
			n.DiskReservedGB -= res.DiskGB
			res.Status = "released"
			return nil
		}
	}
	return nil
}
func (m *mockInfraRepo) CreateInstance(ctx context.Context, inst *domainInfrastructure.Instance) (*domainInfrastructure.Instance, error) {
	return inst, nil
}
func (m *mockInfraRepo) GetInstanceByID(ctx context.Context, id uuid.UUID) (*domainInfrastructure.Instance, error) {
	return nil, nil
}
func (m *mockInfraRepo) GetInstanceBySubscriptionID(ctx context.Context, subID uuid.UUID) (*domainInfrastructure.Instance, error) {
	return nil, nil
}
func (m *mockInfraRepo) UpdateInstanceStates(ctx context.Context, id uuid.UUID, desiredState, observedState string, provInstID *string) error {
	return nil
}
func (m *mockInfraRepo) UpdateNodeStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (m *mockInfraRepo) ListExpiredReservations(ctx context.Context) ([]*domainInfrastructure.ResourceReservation, error) {
	return nil, nil
}
func (m *mockInfraRepo) ListInstances(ctx context.Context) ([]*domainInfrastructure.Instance, error) {
	return nil, nil
}

func TestDeterministicScheduler(t *testing.T) {
	ctx := context.Background()

	nodeGroupA := uuid.New()
	nodeGroupB := uuid.New()

	node1 := &domainInfrastructure.Node{
		ID:            uuid.New(),
		Name:          "node-1",
		NodeGroupID:   &nodeGroupA,
		Status:        "active",
		CPUTotal:      16,
		MemoryTotalMB: 65536,
		DiskTotalGB:   1000,
		Weight:        100,
	}
	node2 := &domainInfrastructure.Node{
		ID:            uuid.New(),
		Name:          "node-2",
		NodeGroupID:   &nodeGroupA,
		Status:        "active",
		CPUTotal:      32,
		MemoryTotalMB: 131072,
		DiskTotalGB:   2000,
		Weight:        200, // Higher weight and capacity
	}
	node3 := &domainInfrastructure.Node{
		ID:            uuid.New(),
		Name:          "node-3",
		NodeGroupID:   &nodeGroupB,
		Status:        "active",
		CPUTotal:      8,
		MemoryTotalMB: 16384,
		DiskTotalGB:   500,
		Weight:        50,
	}
	nodeOffline := &domainInfrastructure.Node{
		ID:            uuid.New(),
		Name:          "node-offline",
		NodeGroupID:   &nodeGroupA,
		Status:        "offline",
		CPUTotal:      64,
		MemoryTotalMB: 262144,
		DiskTotalGB:   4000,
		Weight:        500,
	}

	repo := newMockInfraRepo([]*domainInfrastructure.Node{node1, node2, node3, nodeOffline})
	sched := scheduler.NewScheduler(repo)

	t.Run("Picks Highest Weighted Node in Group", func(t *testing.T) {
		res, pickedNode, err := sched.SelectAndReserve(ctx, scheduler.ScheduleRequirements{
			OperationID: uuid.New(),
			NodeGroupID: &nodeGroupA,
			CPUCores:    2,
			MemoryMB:    4096,
			DiskGB:      50,
		})
		require.NoError(t, err)
		assert.Equal(t, node2.ID, pickedNode.ID)
		assert.Equal(t, "reserved", res.Status)
		assert.Equal(t, float64(2), res.CPUCores)
	})

	t.Run("Ignores Offline Nodes", func(t *testing.T) {
		// Even if nodeOffline has massive capacity and weight 500, it's never picked
		for i := 0; i < 5; i++ {
			_, pickedNode, err := sched.SelectAndReserve(ctx, scheduler.ScheduleRequirements{
				OperationID: uuid.New(),
				NodeGroupID: &nodeGroupA,
				CPUCores:    1,
				MemoryMB:    1024,
				DiskGB:      10,
			})
			require.NoError(t, err)
			assert.NotEqual(t, nodeOffline.ID, pickedNode.ID)
		}
	})

	t.Run("Filters By NodeGroup", func(t *testing.T) {
		_, pickedNode, err := sched.SelectAndReserve(ctx, scheduler.ScheduleRequirements{
			OperationID: uuid.New(),
			NodeGroupID: &nodeGroupB,
			CPUCores:    1,
			MemoryMB:    1024,
			DiskGB:      10,
		})
		require.NoError(t, err)
		assert.Equal(t, node3.ID, pickedNode.ID)
	})

	t.Run("Fails When Resources Exhausted", func(t *testing.T) {
		_, _, err := sched.SelectAndReserve(ctx, scheduler.ScheduleRequirements{
			OperationID: uuid.New(),
			NodeGroupID: &nodeGroupB,
			CPUCores:    64, // Exceeds node3 capacity (8 cores)
			MemoryMB:    1024,
			DiskGB:      10,
		})
		require.ErrorIs(t, err, domainInfrastructure.ErrNoSchedulableNode)
	})

	t.Run("Deterministic Fallback On Concurrency Race", func(t *testing.T) {
		// Simulate race: node2 will reject reservation with ErrResourceExhausted
		repo.failNodeID = node2.ID

		res, pickedNode, err := sched.SelectAndReserve(ctx, scheduler.ScheduleRequirements{
			OperationID: uuid.New(),
			NodeGroupID: &nodeGroupA,
			CPUCores:    2,
			MemoryMB:    2048,
			DiskGB:      20,
		})
		require.NoError(t, err)
		// Should smoothly fall back to node1
		assert.Equal(t, node1.ID, pickedNode.ID)
		assert.Equal(t, "reserved", res.Status)
	})
}
