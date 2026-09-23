package subscription

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	domainSubscription "vps-billing/internal/domain/subscription"
)

type mockSubRepo struct {
	subs map[uuid.UUID]*domainSubscription.Subscription
}

func newMockSubRepo() *mockSubRepo {
	return &mockSubRepo{subs: make(map[uuid.UUID]*domainSubscription.Subscription)}
}

func (m *mockSubRepo) CreateSubscription(_ context.Context, sub *domainSubscription.Subscription) (*domainSubscription.Subscription, error) {
	m.subs[sub.ID] = sub
	return sub, nil
}

func (m *mockSubRepo) GetSubscriptionByID(_ context.Context, id uuid.UUID) (*domainSubscription.Subscription, error) {
	if s, ok := m.subs[id]; ok {
		return s, nil
	}
	return nil, domainSubscription.ErrSubscriptionNotFound
}

func (m *mockSubRepo) ListSubscriptionsByUserID(_ context.Context, userID uuid.UUID) ([]*domainSubscription.Subscription, error) {
	var res []*domainSubscription.Subscription
	for _, s := range m.subs {
		if s.UserID == userID {
			res = append(res, s)
		}
	}
	return res, nil
}

func (m *mockSubRepo) ListAllSubscriptions(_ context.Context) ([]*domainSubscription.Subscription, error) {
	var res []*domainSubscription.Subscription
	for _, s := range m.subs {
		res = append(res, s)
	}
	return res, nil
}

func (m *mockSubRepo) ListDueSubscriptions(_ context.Context, dueBefore time.Time, _ int) ([]*domainSubscription.Subscription, error) {
	var res []*domainSubscription.Subscription
	for _, s := range m.subs {
		if s.Status == domainSubscription.StatusActive && s.NextDueAt != nil && !s.NextDueAt.After(dueBefore) {
			res = append(res, s)
		}
	}
	return res, nil
}

func (m *mockSubRepo) ListGraceExpiredSubscriptions(_ context.Context, graceBefore time.Time, _ int) ([]*domainSubscription.Subscription, error) {
	var res []*domainSubscription.Subscription
	for _, s := range m.subs {
		if s.Status == domainSubscription.StatusPastDue && s.GraceUntil != nil && !s.GraceUntil.After(graceBefore) {
			res = append(res, s)
		}
	}
	return res, nil
}

func (m *mockSubRepo) UpdateSubscriptionStatus(_ context.Context, id uuid.UUID, status domainSubscription.SubscriptionStatus) (*domainSubscription.Subscription, error) {
	if s, ok := m.subs[id]; ok {
		s.Status = status
		return s, nil
	}
	return nil, domainSubscription.ErrSubscriptionNotFound
}

func (m *mockSubRepo) RenewSubscription(_ context.Context, id uuid.UUID, periodStart, periodEnd, nextDue time.Time) (*domainSubscription.Subscription, error) {
	if s, ok := m.subs[id]; ok {
		s.Status = domainSubscription.StatusActive
		s.CurrentPeriodStart = &periodStart
		s.CurrentPeriodEnd = &periodEnd
		s.NextDueAt = &nextDue
		s.GraceUntil = nil
		return s, nil
	}
	return nil, domainSubscription.ErrSubscriptionNotFound
}

func (m *mockSubRepo) SetCancelAtPeriodEnd(_ context.Context, id uuid.UUID, cancel bool) (*domainSubscription.Subscription, error) {
	if s, ok := m.subs[id]; ok {
		s.CancelAtPeriodEnd = cancel
		return s, nil
	}
	return nil, domainSubscription.ErrSubscriptionNotFound
}

func TestCalculatePeriod(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	monthly := CalculatePeriod(start, domainSubscription.CycleMonthly)
	if monthly.Month() != time.February {
		t.Errorf("expected Feb, got %v", monthly.Month())
	}

	quarterly := CalculatePeriod(start, domainSubscription.CycleQuarterly)
	if quarterly.Month() != time.April {
		t.Errorf("expected April, got %v", quarterly.Month())
	}

	annually := CalculatePeriod(start, domainSubscription.CycleAnnually)
	if annually.Year() != 2027 {
		t.Errorf("expected 2027, got %v", annually.Year())
	}
}

func TestSubscriptionLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := newMockSubRepo()
	svc := NewSubscriptionService(repo, nil)

	userID := uuid.New()
	subID := uuid.New()
	t0 := time.Now().UTC().Add(-48 * time.Hour)
	tDue := t0.Add(24 * time.Hour) // Due 24 hours ago!

	sub := &domainSubscription.Subscription{
		ID:                 subID,
		UserID:             userID,
		PlanID:             uuid.New(),
		Status:             domainSubscription.StatusActive,
		BillingCycle:       domainSubscription.CycleMonthly,
		PriceMinor:         1000,
		Currency:           "USD",
		CurrentPeriodStart: &t0,
		CurrentPeriodEnd:   &tDue,
		NextDueAt:          &tDue,
	}
	_, _ = repo.CreateSubscription(ctx, sub)

	// Step 1: Check due -> transitions to past_due
	now := time.Now().UTC()
	dueCount, suspendedCount, err := svc.ProcessDueAndGraceCheck(ctx, now, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed due check: %v", err)
	}
	if dueCount != 1 || suspendedCount != 0 {
		t.Errorf("expected 1 due, got due=%d, suspended=%d", dueCount, suspendedCount)
	}
	if sub.Status != domainSubscription.StatusPastDue {
		t.Errorf("expected status past_due, got %s", sub.Status)
	}

	// Step 2: Grace period expires -> transitions to suspended
	sub.GraceUntil = &t0 // Grace expired in past
	dueCount, suspendedCount, err = svc.ProcessDueAndGraceCheck(ctx, now, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed grace check: %v", err)
	}
	if suspendedCount != 1 {
		t.Errorf("expected 1 suspended, got %d", suspendedCount)
	}
	if sub.Status != domainSubscription.StatusSuspended {
		t.Errorf("expected status suspended, got %s", sub.Status)
	}

	// Step 3: Renew subscription -> reactivates to active
	renewed, err := svc.RenewSubscription(ctx, subID)
	if err != nil {
		t.Fatalf("failed renewal: %v", err)
	}
	if renewed.Status != domainSubscription.StatusActive {
		t.Errorf("expected status active after renewal, got %s", renewed.Status)
	}
	if renewed.GraceUntil != nil {
		t.Errorf("expected nil grace_until after renewal")
	}
}
