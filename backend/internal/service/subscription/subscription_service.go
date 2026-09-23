package subscription

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	domainCommerce "vps-billing/internal/domain/commerce"
	domainSubscription "vps-billing/internal/domain/subscription"
)

type SubscriptionService struct {
	subRepo     domainSubscription.SubscriptionRepository
	productRepo domainCommerce.ProductRepository
}

func NewSubscriptionService(
	subRepo domainSubscription.SubscriptionRepository,
	productRepo domainCommerce.ProductRepository,
) *SubscriptionService {
	return &SubscriptionService{
		subRepo:     subRepo,
		productRepo: productRepo,
	}
}

func CalculatePeriod(start time.Time, cycle domainSubscription.BillingCycle) time.Time {
	switch cycle {
	case domainSubscription.CycleQuarterly:
		return start.AddDate(0, 3, 0)
	case domainSubscription.CycleSemiAnnually:
		return start.AddDate(0, 6, 0)
	case domainSubscription.CycleAnnually:
		return start.AddDate(1, 0, 0)
	default: // monthly
		return start.AddDate(0, 1, 0)
	}
}

func (s *SubscriptionService) CreateSubscriptionFromPlan(
	ctx context.Context,
	userID uuid.UUID,
	planID uuid.UUID,
) (*domainSubscription.Subscription, error) {
	plan, err := s.productRepo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch plan: %w", err)
	}

	now := time.Now().UTC()
	periodEnd := CalculatePeriod(now, domainSubscription.BillingCycle(plan.BillingCycle))

	sub := &domainSubscription.Subscription{
		ID:                 uuid.New(),
		UserID:             userID,
		PlanID:             plan.ID,
		Status:             domainSubscription.StatusActive,
		BillingCycle:       domainSubscription.BillingCycle(plan.BillingCycle),
		PriceMinor:         plan.PriceMinor,
		Currency:           plan.Currency,
		StartedAt:          &now,
		CurrentPeriodStart: &now,
		CurrentPeriodEnd:   &periodEnd,
		NextDueAt:          &periodEnd,
		CancelAtPeriodEnd:  false,
		Version:            1,
	}

	return s.subRepo.CreateSubscription(ctx, sub)
}

func (s *SubscriptionService) GetSubscriptionByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*domainSubscription.Subscription, error) {
	sub, err := s.subRepo.GetSubscriptionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sub.UserID != userID {
		return nil, domainSubscription.ErrSubscriptionNotFound
	}
	return sub, nil
}

func (s *SubscriptionService) ListUserSubscriptions(ctx context.Context, userID uuid.UUID) ([]*domainSubscription.Subscription, error) {
	return s.subRepo.ListSubscriptionsByUserID(ctx, userID)
}

func (s *SubscriptionService) ListAllSubscriptions(ctx context.Context) ([]*domainSubscription.Subscription, error) {
	return s.subRepo.ListAllSubscriptions(ctx)
}

func (s *SubscriptionService) RenewSubscription(ctx context.Context, id uuid.UUID) (*domainSubscription.Subscription, error) {
	sub, err := s.subRepo.GetSubscriptionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if sub.Status == domainSubscription.StatusCancelled || sub.Status == domainSubscription.StatusTerminated {
		return nil, domainSubscription.ErrSubscriptionExpired
	}

	now := time.Now().UTC()
	var newStart time.Time
	if sub.CurrentPeriodEnd != nil && sub.CurrentPeriodEnd.After(now) {
		newStart = *sub.CurrentPeriodEnd
	} else {
		newStart = now
	}

	newEnd := CalculatePeriod(newStart, sub.BillingCycle)
	nextDue := newEnd

	return s.subRepo.RenewSubscription(ctx, id, newStart, newEnd, nextDue)
}

func (s *SubscriptionService) CancelSubscription(ctx context.Context, userID uuid.UUID, id uuid.UUID, immediate bool) (*domainSubscription.Subscription, error) {
	sub, err := s.GetSubscriptionByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if sub.Status != domainSubscription.StatusActive && sub.Status != domainSubscription.StatusPastDue {
		return nil, domainSubscription.ErrInvalidSubscriptionStatus
	}

	if immediate {
		return s.subRepo.UpdateSubscriptionStatus(ctx, id, domainSubscription.StatusCancelled)
	}

	return s.subRepo.SetCancelAtPeriodEnd(ctx, id, true)
}

// ProcessDueAndGraceCheck performs scheduled state machine transitions:
// active -> past_due (when now > next_due_at)
// past_due -> suspended (when now > grace_until)
func (s *SubscriptionService) ProcessDueAndGraceCheck(ctx context.Context, now time.Time, graceWindow time.Duration) (int, int, error) {
	// 1. Find active subscriptions that passed due date
	dueSubs, err := s.subRepo.ListDueSubscriptions(ctx, now, 100)
	if err != nil {
		return 0, 0, err
	}

	dueCount := 0
	for _, sub := range dueSubs {
		graceUntil := now.Add(graceWindow)
		sub.GraceUntil = &graceUntil
		_, err := s.subRepo.UpdateSubscriptionStatus(ctx, sub.ID, domainSubscription.StatusPastDue)
		if err == nil {
			dueCount++
		}
	}

	// 2. Find past_due subscriptions that passed grace window
	graceExpiredSubs, err := s.subRepo.ListGraceExpiredSubscriptions(ctx, now, 100)
	if err != nil {
		return dueCount, 0, err
	}

	suspendedCount := 0
	for _, sub := range graceExpiredSubs {
		_, err := s.subRepo.UpdateSubscriptionStatus(ctx, sub.ID, domainSubscription.StatusSuspended)
		if err == nil {
			suspendedCount++
		}
	}

	return dueCount, suspendedCount, nil
}
