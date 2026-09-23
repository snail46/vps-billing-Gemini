package repository

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	domainInfrastructure "vps-billing/internal/domain/infrastructure"
)

type PostgresInfrastructureRepository struct {
	db *pgxpool.Pool
	q  *Queries
}

func NewPostgresInfrastructureRepository(db *pgxpool.Pool, q *Queries) *PostgresInfrastructureRepository {
	return &PostgresInfrastructureRepository{
		db: db,
		q:  q,
	}
}

// Providers
func (r *PostgresInfrastructureRepository) CreateProvider(ctx context.Context, p *domainInfrastructure.Provider) (*domainInfrastructure.Provider, error) {
	row, err := r.q.CreateProvider(ctx, CreateProviderParams{
		ID:            toPgUUID(p.ID),
		Name:          p.Name,
		ProviderType:  p.ProviderType,
		Endpoint:      toPgTextPtr(p.Endpoint),
		CredentialRef: toPgTextPtr(p.CredentialRef),
		Status:        p.Status,
		Version:       toPgTextPtr(p.Version),
		Config:        p.Config,
		Capabilities:  p.Capabilities,
	})
	if err != nil {
		return nil, err
	}
	return toDomainProvider(row), nil
}

func (r *PostgresInfrastructureRepository) GetProviderByID(ctx context.Context, id uuid.UUID) (*domainInfrastructure.Provider, error) {
	row, err := r.q.GetProviderByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainInfrastructure.ErrProviderNotFound
		}
		return nil, err
	}
	return toDomainProvider(row), nil
}

func (r *PostgresInfrastructureRepository) ListProviders(ctx context.Context) ([]*domainInfrastructure.Provider, error) {
	rows, err := r.q.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]*domainInfrastructure.Provider, len(rows))
	for i, row := range rows {
		list[i] = toDomainProvider(row)
	}
	return list, nil
}

func (r *PostgresInfrastructureRepository) UpdateProviderHealth(ctx context.Context, id uuid.UUID, status string) error {
	return r.q.UpdateProviderHealth(ctx, UpdateProviderHealthParams{
		ID:     toPgUUID(id),
		Status: status,
	})
}

// Node Groups
func (r *PostgresInfrastructureRepository) CreateNodeGroup(ctx context.Context, ng *domainInfrastructure.NodeGroup) (*domainInfrastructure.NodeGroup, error) {
	row, err := r.q.CreateNodeGroup(ctx, CreateNodeGroupParams{
		ID:     toPgUUID(ng.ID),
		Name:   ng.Name,
		Region: ng.Region,
		Status: ng.Status,
	})
	if err != nil {
		return nil, err
	}
	return toDomainNodeGroup(row), nil
}

func (r *PostgresInfrastructureRepository) GetNodeGroupByID(ctx context.Context, id uuid.UUID) (*domainInfrastructure.NodeGroup, error) {
	row, err := r.q.GetNodeGroupByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainInfrastructure.ErrNodeGroupNotFound
		}
		return nil, err
	}
	return toDomainNodeGroup(row), nil
}

func (r *PostgresInfrastructureRepository) ListNodeGroups(ctx context.Context) ([]*domainInfrastructure.NodeGroup, error) {
	rows, err := r.q.ListNodeGroups(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]*domainInfrastructure.NodeGroup, len(rows))
	for i, row := range rows {
		list[i] = toDomainNodeGroup(row)
	}
	return list, nil
}

// Nodes
func (r *PostgresInfrastructureRepository) CreateNode(ctx context.Context, node *domainInfrastructure.Node) (*domainInfrastructure.Node, error) {
	row, err := r.q.CreateNode(ctx, CreateNodeParams{
		ID:                toPgUUID(node.ID),
		ProviderID:        toPgUUID(node.ProviderID),
		NodeGroupID:       toPgUUIDPtr(node.NodeGroupID),
		ProviderNodeID:    toPgTextPtr(node.ProviderNodeID),
		Name:              node.Name,
		Region:            node.Region,
		Status:            node.Status,
		CpuTotal:          toPgNumeric(node.CPUTotal),
		MemoryTotalMb:     node.MemoryTotalMB,
		DiskTotalGb:       node.DiskTotalGB,
		CpuAllocated:      toPgNumeric(node.CPUAllocated),
		MemoryAllocatedMb: node.MemoryAllocatedMB,
		DiskAllocatedGb:   node.DiskAllocatedGB,
		CpuReserved:       toPgNumeric(node.CPUReserved),
		MemoryReservedMb:  node.MemoryReservedMB,
		DiskReservedGb:    node.DiskReservedGB,
		Weight:            int32(node.Weight),
		Capabilities:      node.Capabilities,
	})
	if err != nil {
		return nil, err
	}
	return toDomainNode(row), nil
}

func (r *PostgresInfrastructureRepository) GetNodeByID(ctx context.Context, id uuid.UUID) (*domainInfrastructure.Node, error) {
	row, err := r.q.GetNodeByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainInfrastructure.ErrNodeNotFound
		}
		return nil, err
	}
	return toDomainNode(row), nil
}

func (r *PostgresInfrastructureRepository) ListNodes(ctx context.Context) ([]*domainInfrastructure.Node, error) {
	rows, err := r.q.ListNodes(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]*domainInfrastructure.Node, len(rows))
	for i, row := range rows {
		list[i] = toDomainNode(row)
	}
	return list, nil
}

func (r *PostgresInfrastructureRepository) ListActiveNodes(ctx context.Context) ([]*domainInfrastructure.Node, error) {
	rows, err := r.q.ListActiveNodes(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]*domainInfrastructure.Node, len(rows))
	for i, row := range rows {
		list[i] = toDomainNode(row)
	}
	return list, nil
}

func (r *PostgresInfrastructureRepository) ListNodesByNodeGroup(ctx context.Context, nodeGroupID uuid.UUID) ([]*domainInfrastructure.Node, error) {
	rows, err := r.q.ListNodesByNodeGroup(ctx, toPgUUID(nodeGroupID))
	if err != nil {
		return nil, err
	}
	list := make([]*domainInfrastructure.Node, len(rows))
	for i, row := range rows {
		list[i] = toDomainNode(row)
	}
	return list, nil
}

// ReserveResources performs an atomic check and reservation with row-level locking on the target node.
func (r *PostgresInfrastructureRepository) ReserveResources(ctx context.Context, res *domainInfrastructure.ResourceReservation) (*domainInfrastructure.ResourceReservation, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	// Lock node FOR UPDATE
	nodeRow, err := qtx.GetNodeForUpdate(ctx, toPgUUID(res.NodeID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainInfrastructure.ErrNodeNotFound
		}
		return nil, err
	}

	node := toDomainNode(nodeRow)
	if node.Status != "active" {
		return nil, domainInfrastructure.ErrNodeOffline
	}

	// Verify remaining capacity
	if node.AvailableCPU() < res.CPUCores ||
		node.AvailableMemoryMB() < res.MemoryMB ||
		node.AvailableDiskGB() < res.DiskGB {
		return nil, domainInfrastructure.ErrResourceExhausted
	}

	// Create reservation
	createdRes, err := qtx.CreateResourceReservation(ctx, CreateResourceReservationParams{
		ID:           toPgUUID(res.ID),
		NodeID:       toPgUUID(res.NodeID),
		OperationID:  toPgUUID(res.OperationID),
		CpuCores:     toPgNumeric(res.CPUCores),
		MemoryMb:     res.MemoryMB,
		DiskGb:       res.DiskGB,
		Ipv4Count:    int32(res.IPv4Count),
		Ipv6Count:    int32(res.IPv6Count),
		NatPortCount: int32(res.NATPortCount),
		Status:       "reserved",
		ExpiresAt:    toPgTimestamptz(&res.ExpiresAt),
	})
	if err != nil {
		return nil, err
	}

	// Update node reserved metrics
	err = qtx.UpdateNodeReservationDelta(ctx, UpdateNodeReservationDeltaParams{
		ID:               toPgUUID(res.NodeID),
		CpuReserved:      toPgNumeric(res.CPUCores),
		MemoryReservedMb: res.MemoryMB,
		DiskReservedGb:   res.DiskGB,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return toDomainReservation(createdRes), nil
}

func (r *PostgresInfrastructureRepository) GetReservationByID(ctx context.Context, id uuid.UUID) (*domainInfrastructure.ResourceReservation, error) {
	row, err := r.q.GetResourceReservationByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainInfrastructure.ErrReservationNotFound
		}
		return nil, err
	}
	return toDomainReservation(row), nil
}

// CommitReservation transitions reservation status from reserved to committed, transferring reserved resources to allocated.
func (r *PostgresInfrastructureRepository) CommitReservation(ctx context.Context, reservationID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	resRow, err := qtx.GetResourceReservationByID(ctx, toPgUUID(reservationID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domainInfrastructure.ErrReservationNotFound
		}
		return err
	}

	if resRow.Status != "reserved" {
		return domainInfrastructure.ErrInvalidReservationStatus
	}

	// Commit on node: decrease reserved, increase allocated
	err = qtx.CommitNodeReservation(ctx, CommitNodeReservationParams{
		ID:               resRow.NodeID,
		CpuReserved:      resRow.CpuCores,
		MemoryReservedMb: resRow.MemoryMb,
		DiskReservedGb:   resRow.DiskGb,
	})
	if err != nil {
		return err
	}

	// Mark reservation committed
	err = qtx.UpdateResourceReservationStatus(ctx, UpdateResourceReservationStatusParams{
		ID:     resRow.ID,
		Status: "committed",
	})
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ReleaseReservation transitions reservation to released, decrementing node reserved resources.
func (r *PostgresInfrastructureRepository) ReleaseReservation(ctx context.Context, reservationID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	resRow, err := qtx.GetResourceReservationByID(ctx, toPgUUID(reservationID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domainInfrastructure.ErrReservationNotFound
		}
		return err
	}

	if resRow.Status != "reserved" {
		return nil // already resolved
	}

	cpuFloat := fromPgNumeric(resRow.CpuCores)
	// Negate deltas to subtract from reserved
	err = qtx.UpdateNodeReservationDelta(ctx, UpdateNodeReservationDeltaParams{
		ID:               resRow.NodeID,
		CpuReserved:      toPgNumeric(-cpuFloat),
		MemoryReservedMb: -resRow.MemoryMb,
		DiskReservedGb:   -resRow.DiskGb,
	})
	if err != nil {
		return err
	}

	err = qtx.UpdateResourceReservationStatus(ctx, UpdateResourceReservationStatusParams{
		ID:     resRow.ID,
		Status: "released",
	})
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresInfrastructureRepository) ListExpiredReservations(ctx context.Context) ([]*domainInfrastructure.ResourceReservation, error) {
	rows, err := r.q.ListExpiredReservations(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]*domainInfrastructure.ResourceReservation, len(rows))
	for i, row := range rows {
		res[i] = toDomainReservation(row)
	}
	return res, nil
}

func (r *PostgresInfrastructureRepository) UpdateNodeStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.q.UpdateNodeStatus(ctx, UpdateNodeStatusParams{
		ID:     toPgUUID(id),
		Status: status,
	})
}

// Instances
func (r *PostgresInfrastructureRepository) CreateInstance(ctx context.Context, inst *domainInfrastructure.Instance) (*domainInfrastructure.Instance, error) {
	row, err := r.q.CreateInstance(ctx, CreateInstanceParams{
		ID:                 toPgUUID(inst.ID),
		SubscriptionID:     toPgUUID(inst.SubscriptionID),
		NodeID:             toPgUUIDPtr(inst.NodeID),
		ProviderID:         toPgUUIDPtr(inst.ProviderID),
		ProviderInstanceID: toPgTextPtr(inst.ProviderInstanceID),
		Name:               inst.Name,
		DesiredState:       inst.DesiredState,
		ObservedState:      inst.ObservedState,
		CpuCores:           toPgNumeric(inst.CPUCores),
		MemoryMb:           int32(inst.MemoryMB),
		DiskGb:             int32(inst.DiskGB),
		TrafficLimitGb:     toPgInt8(inst.TrafficLimitGB),
		BandwidthMbps:      toPgInt4(inst.BandwidthMbps),
		ImageID:            toPgTextPtr(inst.ImageID),
		PrimaryIpv4:        toPgInet(inst.PrimaryIPv4),
		PrimaryIpv6:        toPgInet(inst.PrimaryIPv6),
	})
	if err != nil {
		return nil, err
	}
	return toDomainInstance(row), nil
}

func (r *PostgresInfrastructureRepository) GetInstanceByID(ctx context.Context, id uuid.UUID) (*domainInfrastructure.Instance, error) {
	row, err := r.q.GetInstanceByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainInfrastructure.ErrInstanceNotFound
		}
		return nil, err
	}
	return toDomainInstance(row), nil
}

func (r *PostgresInfrastructureRepository) GetInstanceBySubscriptionID(ctx context.Context, subID uuid.UUID) (*domainInfrastructure.Instance, error) {
	row, err := r.q.GetInstanceBySubscriptionID(ctx, toPgUUID(subID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainInfrastructure.ErrInstanceNotFound
		}
		return nil, err
	}
	return toDomainInstance(row), nil
}

func (r *PostgresInfrastructureRepository) ListInstances(ctx context.Context) ([]*domainInfrastructure.Instance, error) {
	rows, err := r.q.ListInstances(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]*domainInfrastructure.Instance, len(rows))
	for i, row := range rows {
		res[i] = toDomainInstance(row)
	}
	return res, nil
}

func (r *PostgresInfrastructureRepository) UpdateInstanceStates(ctx context.Context, id uuid.UUID, desiredState, observedState string, provInstID *string) error {
	return r.q.UpdateInstanceStates(ctx, UpdateInstanceStatesParams{
		ID:                 toPgUUID(id),
		DesiredState:       desiredState,
		ObservedState:      observedState,
		ProviderInstanceID: toPgTextPtr(provInstID),
	})
}

// Conversion helpers
func toDomainProvider(p Providers) *domainInfrastructure.Provider {
	return &domainInfrastructure.Provider{
		ID:                fromPgUUID(p.ID),
		Name:              p.Name,
		ProviderType:      p.ProviderType,
		Endpoint:          fromPgText(p.Endpoint),
		CredentialRef:     fromPgText(p.CredentialRef),
		Status:            p.Status,
		Version:           fromPgText(p.Version),
		Config:            p.Config,
		Capabilities:      p.Capabilities,
		LastHealthCheckAt: fromPgTimestamptz(p.LastHealthCheckAt),
		CreatedAt:         p.CreatedAt.Time,
		UpdatedAt:         p.UpdatedAt.Time,
	}
}

func toDomainNodeGroup(ng NodeGroups) *domainInfrastructure.NodeGroup {
	return &domainInfrastructure.NodeGroup{
		ID:        fromPgUUID(ng.ID),
		Name:      ng.Name,
		Region:    ng.Region,
		Status:    ng.Status,
		CreatedAt: ng.CreatedAt.Time,
		UpdatedAt: ng.UpdatedAt.Time,
	}
}

func toDomainNode(n Nodes) *domainInfrastructure.Node {
	return &domainInfrastructure.Node{
		ID:                fromPgUUID(n.ID),
		ProviderID:        fromPgUUID(n.ProviderID),
		NodeGroupID:       fromPgUUIDPtr(n.NodeGroupID),
		ProviderNodeID:    fromPgText(n.ProviderNodeID),
		Name:              n.Name,
		Region:            n.Region,
		Status:            n.Status,
		CPUTotal:          fromPgNumeric(n.CpuTotal),
		MemoryTotalMB:     n.MemoryTotalMb,
		DiskTotalGB:       n.DiskTotalGb,
		CPUAllocated:      fromPgNumeric(n.CpuAllocated),
		MemoryAllocatedMB: n.MemoryAllocatedMb,
		DiskAllocatedGB:   n.DiskAllocatedGb,
		CPUReserved:       fromPgNumeric(n.CpuReserved),
		MemoryReservedMB:  n.MemoryReservedMb,
		DiskReservedGB:    n.DiskReservedGb,
		Weight:            int(n.Weight),
		Capabilities:      n.Capabilities,
		LastSeenAt:        fromPgTimestamptz(n.LastSeenAt),
		Version:           n.Version,
		CreatedAt:         n.CreatedAt.Time,
		UpdatedAt:         n.UpdatedAt.Time,
	}
}

func toDomainReservation(r ResourceReservations) *domainInfrastructure.ResourceReservation {
	var exp time.Time
	if r.ExpiresAt.Valid {
		exp = r.ExpiresAt.Time
	}
	return &domainInfrastructure.ResourceReservation{
		ID:           fromPgUUID(r.ID),
		NodeID:       fromPgUUID(r.NodeID),
		OperationID:  fromPgUUID(r.OperationID),
		CPUCores:     fromPgNumeric(r.CpuCores),
		MemoryMB:     r.MemoryMb,
		DiskGB:       r.DiskGb,
		IPv4Count:    int(r.Ipv4Count),
		IPv6Count:    int(r.Ipv6Count),
		NATPortCount: int(r.NatPortCount),
		Status:       r.Status,
		ExpiresAt:    exp,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
	}
}

func toDomainInstance(inst Instances) *domainInfrastructure.Instance {
	var lastSynced *time.Time
	if inst.LastSyncedAt.Valid {
		t := inst.LastSyncedAt.Time
		lastSynced = &t
	}
	var deleted *time.Time
	if inst.DeletedAt.Valid {
		t := inst.DeletedAt.Time
		deleted = &t
	}
	return &domainInfrastructure.Instance{
		ID:                 fromPgUUID(inst.ID),
		SubscriptionID:     fromPgUUID(inst.SubscriptionID),
		NodeID:             fromPgUUIDPtr(inst.NodeID),
		ProviderID:         fromPgUUIDPtr(inst.ProviderID),
		ProviderInstanceID: fromPgText(inst.ProviderInstanceID),
		Name:               inst.Name,
		DesiredState:       inst.DesiredState,
		ObservedState:      inst.ObservedState,
		CPUCores:           fromPgNumeric(inst.CpuCores),
		MemoryMB:           int(inst.MemoryMb),
		DiskGB:             int(inst.DiskGb),
		TrafficLimitGB:     fromPgInt8(inst.TrafficLimitGb),
		BandwidthMbps:      fromPgInt4(inst.BandwidthMbps),
		ImageID:            fromPgText(inst.ImageID),
		PrimaryIPv4:        fromPgInet(inst.PrimaryIpv4),
		PrimaryIPv6:        fromPgInet(inst.PrimaryIpv6),
		LastSyncedAt:       lastSynced,
		Version:            inst.Version,
		CreatedAt:          inst.CreatedAt.Time,
		UpdatedAt:          inst.UpdatedAt.Time,
		DeletedAt:          deleted,
	}
}

func toPgTextPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func toPgInet(ip *string) *netip.Addr {
	if ip == nil || *ip == "" {
		return nil
	}
	addr, err := netip.ParseAddr(*ip)
	if err != nil {
		return nil
	}
	return &addr
}

func fromPgInet(addr *netip.Addr) *string {
	if addr == nil || !addr.IsValid() {
		return nil
	}
	s := addr.String()
	return &s
}
