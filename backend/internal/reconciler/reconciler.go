package reconciler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	domainInfrastructure "vps-billing/internal/domain/infrastructure"
	domainOperation "vps-billing/internal/domain/operation"
	"vps-billing/internal/provider"
	serviceOperation "vps-billing/internal/service/operation"
)

type Reconciler struct {
	infraRepo    domainInfrastructure.InfrastructureRepository
	opRepo       domainOperation.OperationRepository
	opSvc        *serviceOperation.Service
	providersMap map[string]provider.Provider
	stuckTimeout time.Duration
	nodeTTL      time.Duration
}

func NewReconciler(
	infraRepo domainInfrastructure.InfrastructureRepository,
	opRepo domainOperation.OperationRepository,
	opSvc *serviceOperation.Service,
	providersMap map[string]provider.Provider,
	stuckTimeout time.Duration,
	nodeTTL time.Duration,
) *Reconciler {
	if stuckTimeout <= 0 {
		stuckTimeout = 3 * time.Minute
	}
	if nodeTTL <= 0 {
		nodeTTL = 1 * time.Minute
	}
	return &Reconciler{
		infraRepo:    infraRepo,
		opRepo:       opRepo,
		opSvc:        opSvc,
		providersMap: providersMap,
		stuckTimeout: stuckTimeout,
		nodeTTL:      nodeTTL,
	}
}

// ReconcileExpiredReservations finds and releases expired reservations back to node capacity.
func (r *Reconciler) ReconcileExpiredReservations(ctx context.Context) (int, error) {
	expired, err := r.infraRepo.ListExpiredReservations(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list expired reservations: %w", err)
	}

	released := 0
	for _, res := range expired {
		if err := r.infraRepo.ReleaseReservation(ctx, res.ID); err == nil {
			released++
			slog.Info("released expired resource reservation",
				slog.String("reservation_id", res.ID.String()),
				slog.String("node_id", res.NodeID.String()),
			)
		}
	}
	return released, nil
}

// ReconcileStuckOperations detects operations that stalled due to worker crash or network partition.
func (r *Reconciler) ReconcileStuckOperations(ctx context.Context) (int, error) {
	threshold := time.Now().UTC().Add(-r.stuckTimeout)
	stuckOps, err := r.opRepo.ListStuckOperations(ctx, threshold)
	if err != nil {
		return 0, fmt.Errorf("failed to list stuck operations: %w", err)
	}

	recovered := 0
	for _, op := range stuckOps {
		if op.Retryable && op.RetryCount < op.MaxRetries {
			// Transition back to retrying so a worker picks it up
			err := r.opSvc.UpdateProgress(ctx, op.ID, domainOperation.StatusRetrying, "retrying", "operations.recovered_after_crash", op.Progress, nil)
			if err == nil {
				recovered++
				slog.Warn("recovered stuck operation back into retrying queue",
					slog.String("operation_id", op.ID.String()),
					slog.String("type", op.Type),
				)
			}
		} else {
			// Fail permanently
			_ = r.opSvc.FailOperation(ctx, op.ID, "WORKER_CRASHED_OR_TIMEOUT", "operation timed out with no worker heartbeat")
			recovered++
		}
	}
	return recovered, nil
}

// ReconcileNodeHeartbeats checks node freshness and marks expired nodes offline and their instances unknown.
func (r *Reconciler) ReconcileNodeHeartbeats(ctx context.Context) (int, error) {
	nodes, err := r.infraRepo.ListActiveNodes(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list active nodes: %w", err)
	}

	markedOffline := 0
	now := time.Now().UTC()
	for _, n := range nodes {
		if n.LastSeenAt != nil && now.Sub(*n.LastSeenAt) > r.nodeTTL {
			// Node missed heartbeats, mark offline
			if err := r.infraRepo.UpdateNodeStatus(ctx, n.ID, "offline"); err == nil {
				markedOffline++
				slog.Warn("marked node offline due to expired heartbeat",
					slog.String("node_id", n.ID.String()),
					slog.String("name", n.Name),
				)
				// Set instances on this node to observed_state = "unknown" (NEVER deleted!)
				r.markNodeInstancesUnknown(ctx, n.ID)
			}
		}
	}
	return markedOffline, nil
}

func (r *Reconciler) markNodeInstancesUnknown(ctx context.Context, nodeID uuid.UUID) {
	instances, err := r.infraRepo.ListInstances(ctx)
	if err != nil {
		return
	}
	for _, inst := range instances {
		if inst.NodeID != nil && *inst.NodeID == nodeID {
			if inst.ObservedState != "unknown" {
				_ = r.infraRepo.UpdateInstanceStates(ctx, inst.ID, inst.DesiredState, "unknown", nil)
				slog.Info("updated instance state to unknown due to offline node",
					slog.String("instance_id", inst.ID.String()),
					slog.String("node_id", nodeID.String()),
				)
			}
		}
	}
}

// ReconcileInstanceStates compares desired state vs observed state and provider state.
func (r *Reconciler) ReconcileInstanceStates(ctx context.Context) (int, error) {
	instances, err := r.infraRepo.ListInstances(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list instances: %w", err)
	}

	reconciled := 0
	for _, inst := range instances {
		if inst.ProviderInstanceID == nil || *inst.ProviderInstanceID == "" {
			continue
		}

		prov := r.resolveProvider(inst.ProviderID)
		if prov == nil {
			continue
		}

		nodeIDStr := ""
		if inst.NodeID != nil {
			nodeIDStr = inst.NodeID.String()
		}

		pInst, err := prov.GetInstance(ctx, provider.GetInstanceRequest{
			ProviderInstanceID: *inst.ProviderInstanceID,
			PlatformInstanceID: inst.ID.String(),
			NodeID:             nodeIDStr,
		})
		if err != nil {
			// If provider is unreachable, set state to unknown, NEVER mark deleted!
			_ = r.infraRepo.UpdateInstanceStates(ctx, inst.ID, inst.DesiredState, "unknown", nil)
			continue
		}

		if pInst.State != inst.ObservedState {
			_ = r.infraRepo.UpdateInstanceStates(ctx, inst.ID, inst.DesiredState, pInst.State, nil)
			reconciled++
		}
	}
	return reconciled, nil
}

// VerifyOrAdoptCreate handles "create-success-but-timeout".
// If a create operation timed out, it verifies whether the instance was actually created on provider.
func (r *Reconciler) VerifyOrAdoptCreate(ctx context.Context, prov provider.Provider, instID uuid.UUID, nodeID string) (*provider.Instance, error) {
	pInst, err := prov.GetInstance(ctx, provider.GetInstanceRequest{
		PlatformInstanceID: instID.String(),
		NodeID:             nodeID,
	})
	if err == nil && pInst != nil && pInst.ProviderInstanceID != "" {
		// Instance was successfully created on provider despite caller timeout!
		slog.Info("recovered and adopted instance created despite timeout",
			slog.String("instance_id", instID.String()),
			slog.String("provider_instance_id", pInst.ProviderInstanceID),
		)
		_ = r.infraRepo.UpdateInstanceStates(ctx, instID, "running", pInst.State, &pInst.ProviderInstanceID)
		return pInst, nil
	}
	return nil, err
}

func (r *Reconciler) resolveProvider(provID *uuid.UUID) provider.Provider {
	if provID != nil {
		if p, ok := r.providersMap[provID.String()]; ok {
			return p
		}
	}
	return r.providersMap["default"]
}

// RunOnce runs one full reconciliation cycle across all subsystems.
func (r *Reconciler) RunOnce(ctx context.Context) {
	_, _ = r.ReconcileExpiredReservations(ctx)
	_, _ = r.ReconcileStuckOperations(ctx)
	_, _ = r.ReconcileNodeHeartbeats(ctx)
	_, _ = r.ReconcileInstanceStates(ctx)
}
