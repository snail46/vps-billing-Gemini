package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	domainOperation "vps-billing/internal/domain/operation"
)

type PostgresOperationRepository struct {
	q *Queries
}

func NewPostgresOperationRepository(q *Queries) *PostgresOperationRepository {
	return &PostgresOperationRepository{q: q}
}

func (r *PostgresOperationRepository) CreateOperation(ctx context.Context, op *domainOperation.Operation) (*domainOperation.Operation, error) {
	row, err := r.q.CreateOperation(ctx, CreateOperationParams{
		ID:                  toPgUUID(op.ID),
		Type:                op.Type,
		ResourceType:        op.ResourceType,
		ResourceID:          toPgUUID(op.ResourceID),
		Status:              string(op.Status),
		Phase:               toPgTextPtr(op.Phase),
		Progress:            int32(op.Progress),
		MessageKey:          toPgTextPtr(op.MessageKey),
		ProviderID:          toPgUUIDPtr(op.ProviderID),
		ProviderOperationID: toPgTextPtr(op.ProviderOperationID),
		IdempotencyKey:      op.IdempotencyKey,
		Retryable:           op.Retryable,
		RetryCount:          int32(op.RetryCount),
		MaxRetries:          int32(op.MaxRetries),
		TraceID:             op.TraceID,
	})
	if err != nil {
		return nil, err
	}
	return toDomainOperation(row), nil
}

func (r *PostgresOperationRepository) GetOperationByID(ctx context.Context, id uuid.UUID) (*domainOperation.Operation, error) {
	row, err := r.q.GetOperationByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainOperation.ErrOperationNotFound
		}
		return nil, err
	}
	op := toDomainOperation(row)

	// Fetch steps
	stepRows, err := r.q.ListOperationSteps(ctx, toPgUUID(id))
	if err == nil && len(stepRows) > 0 {
		steps := make([]*domainOperation.OperationStep, len(stepRows))
		for i, s := range stepRows {
			steps[i] = toDomainOperationStep(s)
		}
		op.Steps = steps
	}

	return op, nil
}

func (r *PostgresOperationRepository) GetOperationByIdempotencyKey(ctx context.Context, key string) (*domainOperation.Operation, error) {
	row, err := r.q.GetOperationByIdempotencyKey(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainOperation.ErrOperationNotFound
		}
		return nil, err
	}
	return toDomainOperation(row), nil
}

func (r *PostgresOperationRepository) ListOperationsByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*domainOperation.Operation, error) {
	rows, err := r.q.ListOperationsByResource(ctx, ListOperationsByResourceParams{
		ResourceType: resourceType,
		ResourceID:   toPgUUID(resourceID),
	})
	if err != nil {
		return nil, err
	}
	list := make([]*domainOperation.Operation, len(rows))
	for i, row := range rows {
		list[i] = toDomainOperation(row)
	}
	return list, nil
}

func (r *PostgresOperationRepository) ListPendingOperations(ctx context.Context, limit int) ([]*domainOperation.Operation, error) {
	rows, err := r.q.ListPendingOperations(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	list := make([]*domainOperation.Operation, len(rows))
	for i, row := range rows {
		list[i] = toDomainOperation(row)
	}
	return list, nil
}

func (r *PostgresOperationRepository) ListRecentOperations(ctx context.Context, limit int) ([]*domainOperation.Operation, error) {
	rows, err := r.q.ListRecentOperations(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	list := make([]*domainOperation.Operation, len(rows))
	for i, row := range rows {
		list[i] = toDomainOperation(row)
	}
	return list, nil
}

func (r *PostgresOperationRepository) ListStuckOperations(ctx context.Context, olderThan time.Time) ([]*domainOperation.Operation, error) {
	rows, err := r.q.ListStuckOperations(ctx, toPgTimestamptz(&olderThan))
	if err != nil {
		return nil, err
	}
	list := make([]*domainOperation.Operation, len(rows))
	for i, row := range rows {
		list[i] = toDomainOperation(row)
	}
	return list, nil
}

func (r *PostgresOperationRepository) UpdateOperationProgress(ctx context.Context, id uuid.UUID, status domainOperation.Status, phase, messageKey string, progress int, startedAt *time.Time) error {
	return r.q.UpdateOperationProgress(ctx, UpdateOperationProgressParams{
		ID:         toPgUUID(id),
		Status:     string(status),
		Phase:      toPgText(phase),
		Progress:   int32(progress),
		MessageKey: toPgText(messageKey),
		StartedAt:  toPgTimestamptz(startedAt),
	})
}

func (r *PostgresOperationRepository) CompleteOperation(ctx context.Context, id uuid.UUID) error {
	return r.q.CompleteOperation(ctx, toPgUUID(id))
}

func (r *PostgresOperationRepository) FailOperation(ctx context.Context, id uuid.UUID, errorCode, errorMessage string) error {
	return r.q.FailOperation(ctx, FailOperationParams{
		ID:           toPgUUID(id),
		ErrorCode:    toPgText(errorCode),
		ErrorMessage: toPgText(errorMessage),
	})
}

func (r *PostgresOperationRepository) SetOperationProviderInfo(ctx context.Context, id uuid.UUID, providerID uuid.UUID, providerOpID string) error {
	return r.q.SetOperationProviderInfo(ctx, SetOperationProviderInfoParams{
		ID:                  toPgUUID(id),
		ProviderID:          toPgUUID(providerID),
		ProviderOperationID: toPgText(providerOpID),
	})
}

func (r *PostgresOperationRepository) CreateOperationStep(ctx context.Context, step *domainOperation.OperationStep) (*domainOperation.OperationStep, error) {
	row, err := r.q.CreateOperationStep(ctx, CreateOperationStepParams{
		ID:          toPgUUID(step.ID),
		OperationID: toPgUUID(step.OperationID),
		StepKey:     step.StepKey,
		StepOrder:   int32(step.StepOrder),
		Status:      string(step.Status),
		Progress:    int32(step.Progress),
		Attempt:     int32(step.Attempt),
	})
	if err != nil {
		return nil, err
	}
	return toDomainOperationStep(row), nil
}

func (r *PostgresOperationRepository) ListOperationSteps(ctx context.Context, opID uuid.UUID) ([]*domainOperation.OperationStep, error) {
	rows, err := r.q.ListOperationSteps(ctx, toPgUUID(opID))
	if err != nil {
		return nil, err
	}
	list := make([]*domainOperation.OperationStep, len(rows))
	for i, row := range rows {
		list[i] = toDomainOperationStep(row)
	}
	return list, nil
}

func (r *PostgresOperationRepository) UpdateOperationStep(ctx context.Context, opID uuid.UUID, stepKey string, status domainOperation.StepStatus, progress int, errorCode, errorMessage *string, startedAt, finishedAt *time.Time) error {
	return r.q.UpdateOperationStep(ctx, UpdateOperationStepParams{
		OperationID:  toPgUUID(opID),
		StepKey:      stepKey,
		Status:       string(status),
		Progress:     int32(progress),
		ErrorCode:    toPgTextPtr(errorCode),
		ErrorMessage: toPgTextPtr(errorMessage),
		StartedAt:    toPgTimestamptz(startedAt),
		FinishedAt:   toPgTimestamptz(finishedAt),
	})
}

func toDomainOperation(o Operations) *domainOperation.Operation {
	return &domainOperation.Operation{
		ID:                  fromPgUUID(o.ID),
		Type:                o.Type,
		ResourceType:        o.ResourceType,
		ResourceID:          fromPgUUID(o.ResourceID),
		Status:              domainOperation.Status(o.Status),
		Phase:               fromPgText(o.Phase),
		Progress:            int(o.Progress),
		MessageKey:          fromPgText(o.MessageKey),
		ProviderID:          fromPgUUIDPtr(o.ProviderID),
		ProviderOperationID: fromPgText(o.ProviderOperationID),
		IdempotencyKey:      o.IdempotencyKey,
		Retryable:           o.Retryable,
		RetryCount:          int(o.RetryCount),
		MaxRetries:          int(o.MaxRetries),
		ErrorCode:           fromPgText(o.ErrorCode),
		ErrorMessage:        fromPgText(o.ErrorMessage),
		TraceID:             o.TraceID,
		StartedAt:           fromPgTimestamptz(o.StartedAt),
		FinishedAt:          fromPgTimestamptz(o.FinishedAt),
		CreatedAt:           o.CreatedAt.Time,
		UpdatedAt:           o.UpdatedAt.Time,
	}
}

func toDomainOperationStep(s OperationSteps) *domainOperation.OperationStep {
	return &domainOperation.OperationStep{
		ID:           fromPgUUID(s.ID),
		OperationID:  fromPgUUID(s.OperationID),
		StepKey:      s.StepKey,
		StepOrder:    int(s.StepOrder),
		Status:       domainOperation.StepStatus(s.Status),
		Progress:     int(s.Progress),
		Attempt:      int(s.Attempt),
		ErrorCode:    fromPgText(s.ErrorCode),
		ErrorMessage: fromPgText(s.ErrorMessage),
		StartedAt:    fromPgTimestamptz(s.StartedAt),
		FinishedAt:   fromPgTimestamptz(s.FinishedAt),
		CreatedAt:    s.CreatedAt.Time,
		UpdatedAt:    s.UpdatedAt.Time,
	}
}
