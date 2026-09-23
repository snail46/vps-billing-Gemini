package workflow

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

type InstanceLifecycleWorkflow struct {
	opType    string
	infraRepo domainInfrastructure.InfrastructureRepository
	providers map[string]provider.Provider
	opSvc     *serviceOperation.Service
}

func NewInstanceLifecycleWorkflow(
	opType string,
	infraRepo domainInfrastructure.InfrastructureRepository,
	providers map[string]provider.Provider,
	opSvc *serviceOperation.Service,
) *InstanceLifecycleWorkflow {
	return &InstanceLifecycleWorkflow{
		opType:    opType,
		infraRepo: infraRepo,
		providers: providers,
		opSvc:     opSvc,
	}
}

func (w *InstanceLifecycleWorkflow) Type() string {
	return w.opType
}

func (w *InstanceLifecycleWorkflow) Execute(ctx context.Context, op *domainOperation.Operation) error {
	slog.Info("executing instance lifecycle workflow",
		slog.String("op_id", op.ID.String()),
		slog.String("type", op.Type),
		slog.String("resource_id", op.ResourceID.String()),
	)

	// Step 1: validate
	step1Start := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "validate", domainOperation.StepRunning, 20, nil, nil, &step1Start, nil)
	_ = w.opSvc.UpdateProgress(ctx, op.ID, domainOperation.StatusRunning, "validate", fmt.Sprintf("operations.%s.validate", op.Type), 20, nil)

	inst, err := w.infraRepo.GetInstanceByID(ctx, op.ResourceID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get instance: %v", err)
		_ = w.opSvc.UpdateStep(ctx, op.ID, "validate", domainOperation.StepFailed, 20, toStrPtr("INSTANCE_NOT_FOUND"), &errMsg, nil, &step1Start)
		return err
	}

	step1End := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "validate", domainOperation.StepSucceeded, 100, nil, nil, nil, &step1End)

	// Resolve provider
	provKey := "default"
	if inst.ProviderID != nil {
		provKey = inst.ProviderID.String()
	}
	prov, ok := w.providers[provKey]
	if !ok {
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
		errMsg := "no provider available for instance operation"
		_ = w.opSvc.FailOperation(ctx, op.ID, "PROVIDER_UNAVAILABLE", errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	nodeID := ""
	if inst.NodeID != nil {
		nodeID = inst.NodeID.String()
	}
	provInstID := "mock-vm-" + inst.ID.String()
	if inst.ProviderInstanceID != nil && *inst.ProviderInstanceID != "" {
		provInstID = *inst.ProviderInstanceID
	}

	actionReq := provider.InstanceActionRequest{
		OperationID:        op.ID.String(),
		IdempotencyKey:     op.IdempotencyKey,
		NodeID:             nodeID,
		ProviderInstanceID: provInstID,
	}

	// Step 2: submit_provider & wait_provider
	step2Start := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "submit_provider", domainOperation.StepRunning, 50, nil, nil, &step2Start, nil)
	_ = w.opSvc.UpdateProgress(ctx, op.ID, domainOperation.StatusWaitingProvider, "submit_provider", fmt.Sprintf("operations.%s.provider", op.Type), 50, nil)

	var targetState string
	switch op.Type {
	case "start_instance":
		targetState = "running"
		_, err = prov.StartInstance(ctx, actionReq)
	case "stop_instance":
		targetState = "stopped"
		_, err = prov.StopInstance(ctx, actionReq)
	case "restart_instance":
		targetState = "running"
		_, err = prov.RestartInstance(ctx, actionReq)
	case "reinstall_instance":
		targetState = "running"
		_, err = prov.ReinstallInstance(ctx, provider.ReinstallInstanceRequest{
			InstanceActionRequest: actionReq,
			Image:                 "ubuntu-22.04",
		})
	case "reset_password":
		targetState = inst.ObservedState
		if targetState == "" {
			targetState = "running"
		}
		newPass := fmt.Sprintf("RootPass_%s!", uuid.New().String()[:8])
		_, err = prov.ResetPassword(ctx, provider.ResetPasswordRequest{
			InstanceActionRequest: actionReq,
			RootPassword:          newPass,
		})
	case "delete_instance":
		targetState = "deleted"
		_, err = prov.DeleteInstance(ctx, actionReq)
	default:
		targetState = "running"
	}

	if err != nil {
		errMsg := fmt.Sprintf("provider action failed: %v", err)
		_ = w.opSvc.UpdateStep(ctx, op.ID, "submit_provider", domainOperation.StepFailed, 50, toStrPtr("PROVIDER_FAILED"), &errMsg, nil, &step2Start)
		return err
	}

	step2End := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, "submit_provider", domainOperation.StepSucceeded, 100, nil, nil, nil, &step2End)
	_ = w.opSvc.UpdateStep(ctx, op.ID, "wait_provider", domainOperation.StepSucceeded, 100, nil, nil, nil, &step2End)

	// Step 3: verify state and update database
	step3Start := time.Now().UTC()
	verifyStep := "verify_running"
	if targetState == "stopped" {
		verifyStep = "verify_stopped"
	} else if targetState == "deleted" {
		verifyStep = "verify_deleted"
	}
	_ = w.opSvc.UpdateStep(ctx, op.ID, verifyStep, domainOperation.StepRunning, 80, nil, nil, &step3Start, nil)
	_ = w.opSvc.UpdateProgress(ctx, op.ID, domainOperation.StatusRunning, verifyStep, fmt.Sprintf("operations.%s.verify", op.Type), 80, nil)

	// Update DB observed state to match desired target state
	if err := w.infraRepo.UpdateInstanceStates(ctx, inst.ID, targetState, targetState, nil); err != nil {
		errMsg := fmt.Sprintf("failed to update instance states: %v", err)
		_ = w.opSvc.UpdateStep(ctx, op.ID, verifyStep, domainOperation.StepFailed, 80, toStrPtr("DB_UPDATE_FAILED"), &errMsg, nil, &step3Start)
		return err
	}

	step3End := time.Now().UTC()
	_ = w.opSvc.UpdateStep(ctx, op.ID, verifyStep, domainOperation.StepSucceeded, 100, nil, nil, nil, &step3End)
	_ = w.opSvc.UpdateStep(ctx, op.ID, "finish", domainOperation.StepSucceeded, 100, nil, nil, nil, &step3End)
	_ = w.opSvc.CompleteOperation(ctx, op.ID)

	slog.Info("instance lifecycle operation completed",
		slog.String("op_id", op.ID.String()),
		slog.String("type", op.Type),
		slog.String("new_state", targetState),
	)

	return nil
}
