package fulfillment

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	domainCommerce "vps-billing/internal/domain/commerce"
	domainOperation "vps-billing/internal/domain/operation"
	domainSubscription "vps-billing/internal/domain/subscription"
	serviceOperation "vps-billing/internal/service/operation"
	"vps-billing/internal/workflow"
)

type FulfillmentService struct {
	orderRepo domainCommerce.OrderRepository
	subRepo   domainSubscription.SubscriptionRepository
	prodRepo  domainCommerce.ProductRepository
	opSvc     *serviceOperation.Service
	provWf    *workflow.ProvisionWorkflow
}

func NewFulfillmentService(
	orderRepo domainCommerce.OrderRepository,
	subRepo domainSubscription.SubscriptionRepository,
	prodRepo domainCommerce.ProductRepository,
	opSvc *serviceOperation.Service,
	provWf *workflow.ProvisionWorkflow,
) *FulfillmentService {
	return &FulfillmentService{
		orderRepo: orderRepo,
		subRepo:   subRepo,
		prodRepo:  prodRepo,
		opSvc:     opSvc,
		provWf:    provWf,
	}
}

func (s *FulfillmentService) FulfillPaidOrder(ctx context.Context, orderID uuid.UUID) error {
	slog.Info("fulfilling paid order", slog.String("order_id", orderID.String()))

	order, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order.Status != "paid" {
		return fmt.Errorf("order %s is not in paid status (status: %s)", orderID, order.Status)
	}

	for idx, item := range order.Items {
		plan, err := s.prodRepo.GetPlanByID(ctx, item.PlanID)
		if err != nil {
			slog.Error("failed to get plan for order item", slog.String("plan_id", item.PlanID.String()))
			continue
		}

		for q := 0; q < item.Quantity; q++ {
			subID := uuid.New()
			now := time.Now().UTC()
			periodEnd := now.AddDate(0, 1, 0)

			sub := &domainSubscription.Subscription{
				ID:                 subID,
				UserID:             order.UserID,
				PlanID:             item.PlanID,
				Status:             domainSubscription.StatusPending,
				BillingCycle:       domainSubscription.BillingCycle(plan.BillingCycle),
				PriceMinor:         item.UnitPriceMinor,
				Currency:           order.Currency,
				StartedAt:          &now,
				CurrentPeriodStart: &now,
				CurrentPeriodEnd:   &periodEnd,
				NextDueAt:          &periodEnd,
			}

			_, err = s.subRepo.CreateSubscription(ctx, sub)
			if err != nil {
				slog.Error("failed to create subscription for order item",
					slog.String("order_id", orderID.String()),
					slog.String("error", err.Error()),
				)
				continue
			}

			// Create provision_instance operation
			idemKey := fmt.Sprintf("provision-%s-%d-%d", order.ID.String(), idx, q)
			op, err := s.opSvc.CreateOperation(ctx, serviceOperation.CreateOperationInput{
				Type:           "provision_instance",
				ResourceType:   "subscription",
				ResourceID:     subID,
				IdempotencyKey: idemKey,
				Retryable:      true,
				MaxRetries:     3,
				TraceID:        fmt.Sprintf("trace-%s", subID.String()[:8]),
				Steps:          workflow.ProvisionSteps,
			})
			if err != nil {
				slog.Error("failed to create provision operation",
					slog.String("sub_id", subID.String()),
					slog.String("error", err.Error()),
				)
				continue
			}

			slog.Info("created provision operation for subscription",
				slog.String("sub_id", subID.String()),
				slog.String("op_id", op.ID.String()),
			)

			// If synchronous workflow runner provided (or background trigger)
			if s.provWf != nil {
				go func(targetOp *domainOperation.Operation) {
					bgCtx := context.Background()
					nowTime := time.Now().UTC()
					_ = s.opSvc.UpdateProgress(bgCtx, targetOp.ID, domainOperation.StatusRunning, "running", "operations.provision_instance.running", 5, &nowTime)
					if execErr := s.provWf.Execute(bgCtx, targetOp); execErr != nil {
						slog.Error("async provision execution failed", slog.String("error", execErr.Error()))
						_ = s.opSvc.FailOperation(bgCtx, targetOp.ID, "PROVISION_FAILED", execErr.Error())
					} else {
						_ = s.opSvc.CompleteOperation(bgCtx, targetOp.ID)
					}
				}(op)
			}
		}
	}

	return nil
}
