package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	domainSubscription "vps-billing/internal/domain/subscription"
)

type PostgresSubscriptionRepository struct {
	queries *Queries
}

func NewPostgresSubscriptionRepository(queries *Queries) *PostgresSubscriptionRepository {
	return &PostgresSubscriptionRepository{queries: queries}
}

var _ domainSubscription.SubscriptionRepository = (*PostgresSubscriptionRepository)(nil)

func toDomainSubscription(s Subscriptions) *domainSubscription.Subscription {
	return &domainSubscription.Subscription{
		ID:                 fromPgUUID(s.ID),
		UserID:             fromPgUUID(s.UserID),
		PlanID:             fromPgUUID(s.PlanID),
		Status:             domainSubscription.SubscriptionStatus(s.Status),
		BillingCycle:       domainSubscription.BillingCycle(s.BillingCycle),
		PriceMinor:         s.PriceMinor,
		Currency:           s.Currency,
		StartedAt:          fromPgTimestamptz(s.StartedAt),
		CurrentPeriodStart: fromPgTimestamptz(s.CurrentPeriodStart),
		CurrentPeriodEnd:   fromPgTimestamptz(s.CurrentPeriodEnd),
		NextDueAt:          fromPgTimestamptz(s.NextDueAt),
		GraceUntil:         fromPgTimestamptz(s.GraceUntil),
		CancelAtPeriodEnd:  s.CancelAtPeriodEnd,
		EndedAt:            fromPgTimestamptz(s.EndedAt),
		Version:            s.Version,
		CreatedAt:          s.CreatedAt.Time,
		UpdatedAt:          s.UpdatedAt.Time,
	}
}

func (r *PostgresSubscriptionRepository) CreateSubscription(ctx context.Context, s *domainSubscription.Subscription) (*domainSubscription.Subscription, error) {
	row, err := r.queries.CreateSubscription(ctx, CreateSubscriptionParams{
		ID:                 toPgUUID(s.ID),
		UserID:             toPgUUID(s.UserID),
		PlanID:             toPgUUID(s.PlanID),
		Status:             string(s.Status),
		BillingCycle:       string(s.BillingCycle),
		PriceMinor:         s.PriceMinor,
		Currency:           s.Currency,
		StartedAt:          toPgTimestamptz(s.StartedAt),
		CurrentPeriodStart: toPgTimestamptz(s.CurrentPeriodStart),
		CurrentPeriodEnd:   toPgTimestamptz(s.CurrentPeriodEnd),
		NextDueAt:          toPgTimestamptz(s.NextDueAt),
		GraceUntil:         toPgTimestamptz(s.GraceUntil),
		CancelAtPeriodEnd:  s.CancelAtPeriodEnd,
		EndedAt:            toPgTimestamptz(s.EndedAt),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}
	return toDomainSubscription(row), nil
}

func (r *PostgresSubscriptionRepository) GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*domainSubscription.Subscription, error) {
	row, err := r.queries.GetSubscriptionByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainSubscription.ErrSubscriptionNotFound
		}
		return nil, err
	}
	return toDomainSubscription(row), nil
}

func (r *PostgresSubscriptionRepository) ListSubscriptionsByUserID(ctx context.Context, userID uuid.UUID) ([]*domainSubscription.Subscription, error) {
	rows, err := r.queries.ListSubscriptionsByUserID(ctx, toPgUUID(userID))
	if err != nil {
		return nil, err
	}
	result := make([]*domainSubscription.Subscription, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainSubscription(row))
	}
	return result, nil
}

func (r *PostgresSubscriptionRepository) ListAllSubscriptions(ctx context.Context) ([]*domainSubscription.Subscription, error) {
	rows, err := r.queries.ListAllSubscriptions(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*domainSubscription.Subscription, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainSubscription(row))
	}
	return result, nil
}

func (r *PostgresSubscriptionRepository) ListDueSubscriptions(ctx context.Context, dueBefore time.Time, limit int) ([]*domainSubscription.Subscription, error) {
	rows, err := r.queries.ListDueSubscriptions(ctx, ListDueSubscriptionsParams{
		NextDueAt: toPgTimestamptz(&dueBefore),
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domainSubscription.Subscription, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainSubscription(row))
	}
	return result, nil
}

func (r *PostgresSubscriptionRepository) ListGraceExpiredSubscriptions(ctx context.Context, graceBefore time.Time, limit int) ([]*domainSubscription.Subscription, error) {
	rows, err := r.queries.ListGraceExpiredSubscriptions(ctx, ListGraceExpiredSubscriptionsParams{
		GraceUntil: toPgTimestamptz(&graceBefore),
		Limit:      int32(limit),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domainSubscription.Subscription, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainSubscription(row))
	}
	return result, nil
}

func (r *PostgresSubscriptionRepository) UpdateSubscriptionStatus(ctx context.Context, id uuid.UUID, status domainSubscription.SubscriptionStatus) (*domainSubscription.Subscription, error) {
	row, err := r.queries.UpdateSubscriptionStatus(ctx, UpdateSubscriptionStatusParams{
		ID:     toPgUUID(id),
		Status: string(status),
	})
	if err != nil {
		return nil, err
	}
	return toDomainSubscription(row), nil
}

func (r *PostgresSubscriptionRepository) RenewSubscription(ctx context.Context, id uuid.UUID, periodStart, periodEnd, nextDue time.Time) (*domainSubscription.Subscription, error) {
	row, err := r.queries.RenewSubscription(ctx, RenewSubscriptionParams{
		ID:                 toPgUUID(id),
		CurrentPeriodStart: toPgTimestamptz(&periodStart),
		CurrentPeriodEnd:   toPgTimestamptz(&periodEnd),
		NextDueAt:          toPgTimestamptz(&nextDue),
	})
	if err != nil {
		return nil, err
	}
	return toDomainSubscription(row), nil
}

func (r *PostgresSubscriptionRepository) SetCancelAtPeriodEnd(ctx context.Context, id uuid.UUID, cancel bool) (*domainSubscription.Subscription, error) {
	row, err := r.queries.SetCancelAtPeriodEnd(ctx, SetCancelAtPeriodEndParams{
		ID:                toPgUUID(id),
		CancelAtPeriodEnd: cancel,
	})
	if err != nil {
		return nil, err
	}
	return toDomainSubscription(row), nil
}
