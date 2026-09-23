package subscription

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SubscriptionRepository interface {
	CreateSubscription(ctx context.Context, sub *Subscription) (*Subscription, error)
	GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*Subscription, error)
	ListSubscriptionsByUserID(ctx context.Context, userID uuid.UUID) ([]*Subscription, error)
	ListAllSubscriptions(ctx context.Context) ([]*Subscription, error)
	ListDueSubscriptions(ctx context.Context, dueBefore time.Time, limit int) ([]*Subscription, error)
	ListGraceExpiredSubscriptions(ctx context.Context, graceBefore time.Time, limit int) ([]*Subscription, error)
	UpdateSubscriptionStatus(ctx context.Context, id uuid.UUID, status SubscriptionStatus) (*Subscription, error)
	RenewSubscription(ctx context.Context, id uuid.UUID, periodStart, periodEnd, nextDue time.Time) (*Subscription, error)
	SetCancelAtPeriodEnd(ctx context.Context, id uuid.UUID, cancel bool) (*Subscription, error)
}
