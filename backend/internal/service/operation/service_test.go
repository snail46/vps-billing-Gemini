package operation_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	domainOperation "vps-billing/internal/domain/operation"
	serviceOperation "vps-billing/internal/service/operation"
)

type mockOpRepo struct {
	mu         sync.Mutex
	operations map[uuid.UUID]*domainOperation.Operation
	steps      map[uuid.UUID][]*domainOperation.OperationStep
}

func newMockOpRepo() *mockOpRepo {
	return &mockOpRepo{
		operations: make(map[uuid.UUID]*domainOperation.Operation),
		steps:      make(map[uuid.UUID][]*domainOperation.OperationStep),
	}
}

func (m *mockOpRepo) CreateOperation(ctx context.Context, op *domainOperation.Operation) (*domainOperation.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operations[op.ID] = op
	return op, nil
}

func (m *mockOpRepo) GetOperationByID(ctx context.Context, id uuid.UUID) (*domainOperation.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	op, ok := m.operations[id]
	if !ok {
		return nil, domainOperation.ErrOperationNotFound
	}
	op.Steps = m.steps[id]
	return op, nil
}

func (m *mockOpRepo) GetOperationByIdempotencyKey(ctx context.Context, key string) (*domainOperation.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, op := range m.operations {
		if op.IdempotencyKey == key {
			op.Steps = m.steps[op.ID]
			return op, nil
		}
	}
	return nil, domainOperation.ErrOperationNotFound
}

func (m *mockOpRepo) ListOperationsByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*domainOperation.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []*domainOperation.Operation
	for _, op := range m.operations {
		if op.ResourceType == resourceType && op.ResourceID == resourceID {
			res = append(res, op)
		}
	}
	return res, nil
}

func (m *mockOpRepo) ListPendingOperations(ctx context.Context, limit int) ([]*domainOperation.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []*domainOperation.Operation
	for _, op := range m.operations {
		if op.Status == domainOperation.StatusQueued || op.Status == domainOperation.StatusRetrying {
			res = append(res, op)
			if len(res) >= limit {
				break
			}
		}
	}
	return res, nil
}

func (m *mockOpRepo) ListRecentOperations(ctx context.Context, limit int) ([]*domainOperation.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []*domainOperation.Operation
	for _, op := range m.operations {
		res = append(res, op)
		if len(res) >= limit {
			break
		}
	}
	return res, nil
}

func (m *mockOpRepo) ListStuckOperations(ctx context.Context, olderThan time.Time) ([]*domainOperation.Operation, error) {
	return nil, nil
}

func (m *mockOpRepo) UpdateOperationProgress(ctx context.Context, id uuid.UUID, status domainOperation.Status, phase, messageKey string, progress int, startedAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	op, ok := m.operations[id]
	if !ok {
		return domainOperation.ErrOperationNotFound
	}
	op.Status = status
	op.Phase = &phase
	op.MessageKey = &messageKey
	op.Progress = progress
	if startedAt != nil {
		op.StartedAt = startedAt
	}
	return nil
}

func (m *mockOpRepo) CompleteOperation(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	op, ok := m.operations[id]
	if !ok {
		return domainOperation.ErrOperationNotFound
	}
	op.Status = domainOperation.StatusSucceeded
	op.Progress = 100
	now := time.Now().UTC()
	op.FinishedAt = &now
	return nil
}

func (m *mockOpRepo) FailOperation(ctx context.Context, id uuid.UUID, errorCode, errorMessage string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	op, ok := m.operations[id]
	if !ok {
		return domainOperation.ErrOperationNotFound
	}
	op.Status = domainOperation.StatusFailed
	op.ErrorCode = &errorCode
	op.ErrorMessage = &errorMessage
	now := time.Now().UTC()
	op.FinishedAt = &now
	return nil
}

func (m *mockOpRepo) SetOperationProviderInfo(ctx context.Context, id uuid.UUID, providerID uuid.UUID, providerOpID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	op, ok := m.operations[id]
	if !ok {
		return domainOperation.ErrOperationNotFound
	}
	op.ProviderID = &providerID
	op.ProviderOperationID = &providerOpID
	return nil
}

func (m *mockOpRepo) CreateOperationStep(ctx context.Context, step *domainOperation.OperationStep) (*domainOperation.OperationStep, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.steps[step.OperationID] = append(m.steps[step.OperationID], step)
	return step, nil
}

func (m *mockOpRepo) ListOperationSteps(ctx context.Context, opID uuid.UUID) ([]*domainOperation.OperationStep, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.steps[opID], nil
}

func (m *mockOpRepo) UpdateOperationStep(ctx context.Context, opID uuid.UUID, stepKey string, status domainOperation.StepStatus, progress int, errorCode, errorMessage *string, startedAt, finishedAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.steps[opID] {
		if s.StepKey == stepKey {
			s.Status = status
			s.Progress = progress
			s.ErrorCode = errorCode
			s.ErrorMessage = errorMessage
			s.StartedAt = startedAt
			s.FinishedAt = finishedAt
			return nil
		}
	}
	return nil
}

func TestOperationServiceLifecycleAndStreaming(t *testing.T) {
	ctx := context.Background()
	repo := newMockOpRepo()
	svc := serviceOperation.NewService(repo, nil)

	resID := uuid.New()
	idemKey := "test-idempotency-1"

	// 1. Create Operation
	op, err := svc.CreateOperation(ctx, serviceOperation.CreateOperationInput{
		Type:           "provision_instance",
		ResourceType:   "instance",
		ResourceID:     resID,
		IdempotencyKey: idemKey,
		Retryable:      true,
		MaxRetries:     3,
		TraceID:        "trace-123",
		Steps:          []string{"step1", "step2", "step3"},
	})
	require.NoError(t, err)
	assert.Equal(t, domainOperation.StatusQueued, op.Status)
	assert.Len(t, op.Steps, 3)

	// 2. Test Idempotent creation
	opDup, err := svc.CreateOperation(ctx, serviceOperation.CreateOperationInput{
		Type:           "provision_instance",
		ResourceType:   "instance",
		ResourceID:     resID,
		IdempotencyKey: idemKey,
	})
	require.NoError(t, err)
	assert.Equal(t, op.ID, opDup.ID)

	// 3. Test Subscription & Live Updates
	events, unsub := svc.Subscribe(ctx, op.ID)
	defer unsub()

	// Update step
	err = svc.UpdateStep(ctx, op.ID, "step1", domainOperation.StepRunning, 50, nil, nil, nil, nil)
	require.NoError(t, err)

	select {
	case evt := <-events:
		assert.Equal(t, op.ID, evt.ID)
		assert.Equal(t, domainOperation.StepRunning, evt.Steps[0].Status)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for step update event")
	}

	// Update overall progress
	err = svc.UpdateProgress(ctx, op.ID, domainOperation.StatusRunning, "step2", "operations.step2", 66, nil)
	require.NoError(t, err)

	select {
	case evt := <-events:
		assert.Equal(t, 66, evt.Progress)
		assert.Equal(t, domainOperation.StatusRunning, evt.Status)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for progress update event")
	}

	// Complete Operation
	err = svc.CompleteOperation(ctx, op.ID)
	require.NoError(t, err)

	select {
	case evt := <-events:
		assert.Equal(t, 100, evt.Progress)
		assert.Equal(t, domainOperation.StatusSucceeded, evt.Status)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for completion event")
	}
}
