package infrastructure

import (
	"context"

	"github.com/google/uuid"
)

type InfrastructureRepository interface {
	// Providers
	CreateProvider(ctx context.Context, p *Provider) (*Provider, error)
	GetProviderByID(ctx context.Context, id uuid.UUID) (*Provider, error)
	ListProviders(ctx context.Context) ([]*Provider, error)
	UpdateProviderHealth(ctx context.Context, id uuid.UUID, status string) error

	// Node Groups
	CreateNodeGroup(ctx context.Context, ng *NodeGroup) (*NodeGroup, error)
	GetNodeGroupByID(ctx context.Context, id uuid.UUID) (*NodeGroup, error)
	ListNodeGroups(ctx context.Context) ([]*NodeGroup, error)

	// Nodes
	CreateNode(ctx context.Context, node *Node) (*Node, error)
	GetNodeByID(ctx context.Context, id uuid.UUID) (*Node, error)
	ListNodes(ctx context.Context) ([]*Node, error)
	ListActiveNodes(ctx context.Context) ([]*Node, error)
	ListNodesByNodeGroup(ctx context.Context, nodeGroupID uuid.UUID) ([]*Node, error)
	UpdateNodeStatus(ctx context.Context, id uuid.UUID, status string) error

	// Atomic Reservation & Node Allocation (run in transactions)
	ReserveResources(ctx context.Context, res *ResourceReservation) (*ResourceReservation, error)
	GetReservationByID(ctx context.Context, id uuid.UUID) (*ResourceReservation, error)
	CommitReservation(ctx context.Context, reservationID uuid.UUID) error
	ReleaseReservation(ctx context.Context, reservationID uuid.UUID) error
	ListExpiredReservations(ctx context.Context) ([]*ResourceReservation, error)

	// Instances
	CreateInstance(ctx context.Context, inst *Instance) (*Instance, error)
	GetInstanceByID(ctx context.Context, id uuid.UUID) (*Instance, error)
	GetInstanceBySubscriptionID(ctx context.Context, subID uuid.UUID) (*Instance, error)
	ListInstances(ctx context.Context) ([]*Instance, error)
	UpdateInstanceStates(ctx context.Context, id uuid.UUID, desiredState, observedState string, provInstID *string) error
}
