package operation

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OperationRepository interface {
	CreateOperation(ctx context.Context, op *Operation) (*Operation, error)
	GetOperationByID(ctx context.Context, id uuid.UUID) (*Operation, error)
	GetOperationByIdempotencyKey(ctx context.Context, key string) (*Operation, error)
	ListOperationsByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*Operation, error)
	ListPendingOperations(ctx context.Context, limit int) ([]*Operation, error)
	ListRecentOperations(ctx context.Context, limit int) ([]*Operation, error)
	ListStuckOperations(ctx context.Context, olderThan time.Time) ([]*Operation, error)
	UpdateOperationProgress(ctx context.Context, id uuid.UUID, status Status, phase, messageKey string, progress int, startedAt *time.Time) error
	CompleteOperation(ctx context.Context, id uuid.UUID) error
	FailOperation(ctx context.Context, id uuid.UUID, errorCode, errorMessage string) error
	SetOperationProviderInfo(ctx context.Context, id uuid.UUID, providerID uuid.UUID, providerOpID string) error

	CreateOperationStep(ctx context.Context, step *OperationStep) (*OperationStep, error)
	ListOperationSteps(ctx context.Context, opID uuid.UUID) ([]*OperationStep, error)
	UpdateOperationStep(ctx context.Context, opID uuid.UUID, stepKey string, status StepStatus, progress int, errorCode, errorMessage *string, startedAt, finishedAt *time.Time) error
}
