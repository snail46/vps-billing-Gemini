package subscription

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionStatus string

const (
	StatusPending    SubscriptionStatus = "pending"
	StatusActive     SubscriptionStatus = "active"
	StatusPastDue    SubscriptionStatus = "past_due"
	StatusSuspended  SubscriptionStatus = "suspended"
	StatusCancelled  SubscriptionStatus = "cancelled"
	StatusExpired    SubscriptionStatus = "expired"
	StatusTerminated SubscriptionStatus = "terminated"
)

type BillingCycle string

const (
	CycleMonthly      BillingCycle = "monthly"
	CycleQuarterly    BillingCycle = "quarterly"
	CycleSemiAnnually BillingCycle = "semi_annually"
	CycleAnnually     BillingCycle = "annually"
)

type Subscription struct {
	ID                 uuid.UUID          `json:"id"`
	UserID             uuid.UUID          `json:"user_id"`
	PlanID             uuid.UUID          `json:"plan_id"`
	Status             SubscriptionStatus `json:"status"`
	BillingCycle       BillingCycle       `json:"billing_cycle"`
	PriceMinor         int64              `json:"price_minor"`
	Currency           string             `json:"currency"`
	StartedAt          *time.Time         `json:"started_at,omitempty"`
	CurrentPeriodStart *time.Time         `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *time.Time         `json:"current_period_end,omitempty"`
	NextDueAt          *time.Time         `json:"next_due_at,omitempty"`
	GraceUntil         *time.Time         `json:"grace_until,omitempty"`
	CancelAtPeriodEnd  bool               `json:"cancel_at_period_end"`
	EndedAt            *time.Time         `json:"ended_at,omitempty"`
	Version            int64              `json:"version"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}
