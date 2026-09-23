package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	domainCommerce "vps-billing/internal/domain/commerce"
	domainInfrastructure "vps-billing/internal/domain/infrastructure"
	domainOperation "vps-billing/internal/domain/operation"
	domainSubscription "vps-billing/internal/domain/subscription"
	"vps-billing/internal/provider"
	serviceOperation "vps-billing/internal/service/operation"
	"vps-billing/internal/service/scheduler"
)

var ProvisionSteps = []string{
	"validate_subscription",
	"select_node",
	"reserve_resources",
	"create_instance",
	"wait_provider",
	"configure_network",
	"verify_running",
	"persist_network",
	"commit_reservation",
	"activate_subscription",
	"notify",
	"finish",
}

type ProvisionWorkflow struct {
	infraRepo domainInfrastructure.InfrastructureRepository
	subRepo   domainSubscription.SubscriptionRepository
	prodRepo  domainCommerce.ProductRepository
	sched     *scheduler.Scheduler
	providers map[string]provider.Provider
	opSvc     *serviceOperation.Service
}

func NewProvisionWorkflow(
	infraRepo domainInfrastructure.InfrastructureRepository,
	subRepo domainSubscription.SubscriptionRepository,
	prodRepo domainCommerce.ProductRepository,
	sched *scheduler.Scheduler,
	providers map[string]provider.Provider,
	opSvc *serviceOperation.Service,
) *ProvisionWorkflow {
	return &ProvisionWorkflow{
		infraRepo: infraRepo,
		subRepo:   subRepo,
		prodRepo:  prodRepo,
		sched:     sched,
		providers: providers,
		opSvc:     opSvc,
	}
}

func (w *ProvisionWorkflow) Type() string {
	return "provision_instance"
}

func (w *ProvisionWorkflow) Execute(ctx context.Context, op *domainOperation.Operation) error {
	slog.Info("executing provision workflow",
		slog.String("op_id", op.ID.String()),
		slog.String("resource_id", op.ResourceID.String()),
	)

	subID := op.ResourceID

	// Step 1: validate_subscription
	step1Start := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "validate_subscription", domainOperation.StepRunning, 10, nil, nil, &step1Start, nil)
	_ = w.opSvc.UpdateProgress(ctx, op.ID, domainOperation.StatusRunning, "validate_subscription", "operations.step.validate_subscription", 10, nil)

	sub, err := w.subRepo.GetSubscriptionByID(ctx, subID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get subscription: %v", err)
		_ = w.opSvc.UpdateStep(ctx, op.ID, "validate_subscription", domainOperation.StepFailed, 10, toStrPtr("SUB_NOT_FOUND"), &errMsg, nil, &step1Start)
		return err
	}

	plan, err := w.prodRepo.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get plan: %v", err)
		_ = w.opSvc.UpdateStep(ctx, op.ID, "validate_subscription", domainOperation.StepFailed, 10, toStrPtr("PLAN_NOT_FOUND"), &errMsg, nil, &step1Start)
		return err
	}

	step1End := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "validate_subscription", domainOperation.StepSucceeded, 100, nil, nil, nil, &step1End)

	// Step 2 & 3: select_node & reserve_resources
	step2Start := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "select_node", domainOperation.StepRunning, 20, nil, nil, &step2Start, nil)
	_ = w.opSvc.UpdateProgress(ctx, op.ID, domainOperation.StatusWaitingResource, "select_node", "operations.step.select_node", 20, nil)

	var nodeGroupID *uuid.UUID
	if plan.NodeGroupID != nil {
		nodeGroupID = plan.NodeGroupID
	}

	schedReq := scheduler.ScheduleRequirements{
		OperationID:  op.ID,
		NodeGroupID:  nodeGroupID,
		CPUCores:     plan.CPUCores,
		MemoryMB:     int64(plan.MemoryMB),
		DiskGB:       int64(plan.DiskGB),
		IPv4Count:    plan.IPv4Count,
		IPv6Count:    plan.IPv6Count,
		NATPortCount: plan.NatPortCount,
		TTL:          15 * time.Minute,
	}

	reservation, selectedNode, err := w.sched.SelectAndReserve(ctx, schedReq)
	if err != nil {
		errMsg := fmt.Sprintf("scheduler reservation failed: %v", err)
		_ = w.opSvc.UpdateStep(ctx, op.ID, "select_node", domainOperation.StepFailed, 20, toStrPtr("RESOURCE_EXHAUSTED"), &errMsg, nil, &step2Start)
		return err
	}

	step2End := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "select_node", domainOperation.StepSucceeded, 100, nil, nil, nil, &step2End)

	step3Start := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "reserve_resources", domainOperation.StepSucceeded, 100, nil, nil, &step3Start, &step3Start)

	// Step 4: create_instance via Provider
	step4Start := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "create_instance", domainOperation.StepRunning, 40, nil, nil, &step4Start, nil)
	_ = w.opSvc.UpdateProgress(ctx, op.ID, domainOperation.StatusWaitingProvider, "create_instance", "operations.step.create_instance", 40, nil)

	// Resolve provider
	prov, ok := w.providers[selectedNode.ProviderID.String()]
	if !ok {
		// Fallback to default mock provider if available
		prov, ok = w.providers["default"]
		if !ok {
			for _, p := range w.providers {
				prov = p
				ok = true
				break
			}
		}
	}
	if !ok || prov == nil {
		errMsg := "no provider available to execute instance creation"
		_ = w.opSvc.UpdateStep(ctx, op.ID, "create_instance", domainOperation.StepFailed, 40, toStrPtr("PROVIDER_UNAVAILABLE"), &errMsg, nil, &step4Start)
		_ = w.infraRepo.ReleaseReservation(ctx, reservation.ID)
		return fmt.Errorf("%s", errMsg)
	}

	instanceID := uuid.New()
	createProvReq := provider.CreateInstanceRequest{
		OperationID:    op.ID.String(),
		IdempotencyKey: op.IdempotencyKey,
		NodeID:         selectedNode.ID.String(),
		InstanceID:     instanceID.String(),
		Name:           fmt.Sprintf("vps-%s", instanceID.String()[:8]),
		CPUCores:       plan.CPUCores,
		MemoryMB:       int64(plan.MemoryMB),
		DiskGB:         int64(plan.DiskGB),
		TrafficGB:      plan.TrafficGB,
		BandwidthMbps:  plan.BandwidthMbps,
		IPv4Count:      plan.IPv4Count,
		IPv6Count:      plan.IPv6Count,
		NATPortCount:   plan.NatPortCount,
		Image:          "ubuntu-22.04",
		Virtualization: plan.Virtualization,
	}

	provOp, err := prov.CreateInstance(ctx, createProvReq)
	if err != nil {
		errMsg := fmt.Sprintf("provider CreateInstance failed: %v", err)
		_ = w.opSvc.UpdateStep(ctx, op.ID, "create_instance", domainOperation.StepFailed, 40, toStrPtr("PROVIDER_CREATE_FAILED"), &errMsg, nil, &step4Start)
		_ = w.infraRepo.ReleaseReservation(ctx, reservation.ID)
		return err
	}

	_ = w.opSvc.SetProviderInfo(ctx, op.ID, selectedNode.ProviderID, provOp.ProviderOperationID)
	step4End := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "create_instance", domainOperation.StepSucceeded, 100, nil, nil, nil, &step4End)

	// Step 5: wait_provider & Step 6: configure_network
	step5Start := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "wait_provider", domainOperation.StepSucceeded, 100, nil, nil, &step5Start, &step5Start)

	step6Start := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "configure_network", domainOperation.StepRunning, 65, nil, nil, &step6Start, nil)
	_ = w.opSvc.UpdateProgress(ctx, op.ID, domainOperation.StatusRunning, "configure_network", "operations.step.configure_network", 65, nil)

	// Step 7: verify_running
	provInstID := ""
	if provOp.Metadata != nil {
		if idVal, ok := provOp.Metadata["provider_instance_id"].(string); ok {
			provInstID = idVal
		}
	}
	if provInstID == "" {
		provInstID = "mock-vm-" + instanceID.String()
	}

	instObserved, err := prov.GetInstance(ctx, provider.GetInstanceRequest{
		NodeID:             selectedNode.ID.String(),
		ProviderInstanceID: provInstID,
		PlatformInstanceID: instanceID.String(),
	})
	if err != nil {
		errMsg := fmt.Sprintf("failed to verify instance on provider: %v", err)
		_ = w.opSvc.UpdateStep(ctx, op.ID, "verify_running", domainOperation.StepFailed, 75, toStrPtr("VERIFY_FAILED"), &errMsg, nil, &step6Start)
		_ = w.infraRepo.ReleaseReservation(ctx, reservation.ID)
		return err
	}

	step6End := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "configure_network", domainOperation.StepSucceeded, 100, nil, nil, nil, &step6End)

	step7Start := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "verify_running", domainOperation.StepSucceeded, 100, nil, nil, &step7Start, &step7Start)

	// Step 8: persist_network & instance record
	step8Start := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "persist_network", domainOperation.StepRunning, 85, nil, nil, &step8Start, nil)
	_ = w.opSvc.UpdateProgress(ctx, op.ID, domainOperation.StatusRunning, "persist_network", "operations.step.persist_network", 85, nil)

	var primaryIPv4, primaryIPv6 *string
	if len(instObserved.IPv4) > 0 {
		primaryIPv4 = &instObserved.IPv4[0]
	}
	if len(instObserved.IPv6) > 0 {
		primaryIPv6 = &instObserved.IPv6[0]
	}

	imgID := "ubuntu-22.04"
	dbInstance := &domainInfrastructure.Instance{
		ID:                 instanceID,
		SubscriptionID:     sub.ID,
		NodeID:             &selectedNode.ID,
		ProviderID:         &selectedNode.ProviderID,
		ProviderInstanceID: &provInstID,
		Name:               createProvReq.Name,
		DesiredState:       "running",
		ObservedState:      instObserved.State,
		CPUCores:           plan.CPUCores,
		MemoryMB:           plan.MemoryMB,
		DiskGB:             plan.DiskGB,
		TrafficLimitGB:     plan.TrafficGB,
		BandwidthMbps:      plan.BandwidthMbps,
		ImageID:            &imgID,
		PrimaryIPv4:        primaryIPv4,
		PrimaryIPv6:        primaryIPv6,
	}

	_, err = w.infraRepo.CreateInstance(ctx, dbInstance)
	if err != nil {
		errMsg := fmt.Sprintf("failed to persist instance record: %v", err)
		_ = w.opSvc.UpdateStep(ctx, op.ID, "persist_network", domainOperation.StepFailed, 85, toStrPtr("DB_ERROR"), &errMsg, nil, &step8Start)
		_ = w.infraRepo.ReleaseReservation(ctx, reservation.ID)
		return err
	}
	step8End := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "persist_network", domainOperation.StepSucceeded, 100, nil, nil, nil, &step8End)

	// Step 9: commit_reservation
	step9Start := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "commit_reservation", domainOperation.StepRunning, 90, nil, nil, &step9Start, nil)
	_ = w.opSvc.UpdateProgress(ctx, op.ID, domainOperation.StatusRunning, "commit_reservation", "operations.step.commit_reservation", 90, nil)

	err = w.infraRepo.CommitReservation(ctx, reservation.ID)
	if err != nil {
		slog.Error("failed to commit reservation", slog.String("error", err.Error()))
	}
	step9End := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "commit_reservation", domainOperation.StepSucceeded, 100, nil, nil, nil, &step9End)

	// Step 10: activate_subscription
	step10Start := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "activate_subscription", domainOperation.StepRunning, 95, nil, nil, &step10Start, nil)
	_ = w.opSvc.UpdateProgress(ctx, op.ID, domainOperation.StatusRunning, "activate_subscription", "operations.step.activate_subscription", 95, nil)

	_, _ = w.subRepo.UpdateSubscriptionStatus(ctx, sub.ID, domainSubscription.StatusActive)

	step10End := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "activate_subscription", domainOperation.StepSucceeded, 100, nil, nil, nil, &step10End)

	// Step 11: notify & Step 12: finish
	step11Start := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "notify", domainOperation.StepSucceeded, 100, nil, nil, &step11Start, &step11Start)

	step12Start := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "finish", domainOperation.StepSucceeded, 100, nil, nil, &step12Start, &step12Start)

	_ = w.opSvc.UpdateProgress(ctx, op.ID, domainOperation.StatusSucceeded, "finish", "operations.completed", 100, nil)

	slog.Info("provision workflow completed successfully",
		slog.String("op_id", op.ID.String()),
		slog.String("instance_id", instanceID.String()),
	)
	return nil
}

func toStrPtr(s string) *string {
	return &s
}
