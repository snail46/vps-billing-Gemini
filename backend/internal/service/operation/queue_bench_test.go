package operation_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	domainOperation "vps-billing/internal/domain/operation"
	serviceOperation "vps-billing/internal/service/operation"
)

type benchOpRepo struct {
	mu    sync.Mutex
	ops   map[uuid.UUID]*domainOperation.Operation
	byKey map[string]*domainOperation.Operation
}

func newBenchOpRepo() *benchOpRepo {
	return &benchOpRepo{
		ops:   make(map[uuid.UUID]*domainOperation.Operation),
		byKey: make(map[string]*domainOperation.Operation),
	}
}

func (r *benchOpRepo) CreateOperation(_ context.Context, op *domainOperation.Operation) (*domainOperation.Operation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ops[op.ID] = op
	if op.IdempotencyKey != "" {
		r.byKey[op.IdempotencyKey] = op
	}
	return op, nil
}

func (r *benchOpRepo) GetOperationByID(_ context.Context, id uuid.UUID) (*domainOperation.Operation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ops[id], nil
}

func (r *benchOpRepo) GetOperationByIdempotencyKey(_ context.Context, key string) (*domainOperation.Operation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.byKey[key], nil
}

func (r *benchOpRepo) ListOperationsByResource(_ context.Context, _ string, _ uuid.UUID) ([]*domainOperation.Operation, error) {
	return nil, nil
}

func (r *benchOpRepo) ListPendingOperations(_ context.Context, _ int) ([]*domainOperation.Operation, error) {
	return nil, nil
}

func (r *benchOpRepo) ListRecentOperations(_ context.Context, _ int) ([]*domainOperation.Operation, error) {
	return nil, nil
}

func (r *benchOpRepo) ListStuckOperations(_ context.Context, _ time.Time) ([]*domainOperation.Operation, error) {
	return nil, nil
}

func (r *benchOpRepo) UpdateOperationProgress(_ context.Context, id uuid.UUID, status domainOperation.Status, phase, msgKey string, progress int, _ *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if op, ok := r.ops[id]; ok {
		op.Status = status
		op.Phase = &phase
		op.Progress = progress
	}
	return nil
}

func (r *benchOpRepo) CompleteOperation(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if op, ok := r.ops[id]; ok {
		op.Status = domainOperation.StatusSucceeded
	}
	return nil
}

func (r *benchOpRepo) FailOperation(_ context.Context, id uuid.UUID, code, msg string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if op, ok := r.ops[id]; ok {
		op.Status = domainOperation.StatusFailed
		op.ErrorCode = &code
		op.ErrorMessage = &msg
	}
	return nil
}

func (r *benchOpRepo) SetOperationProviderInfo(_ context.Context, _, _ uuid.UUID, _ string) error {
	return nil
}

func (r *benchOpRepo) CreateOperationStep(_ context.Context, s *domainOperation.OperationStep) (*domainOperation.OperationStep, error) {
	return s, nil
}

func (r *benchOpRepo) ListOperationSteps(_ context.Context, _ uuid.UUID) ([]*domainOperation.OperationStep, error) {
	return nil, nil
}

func (r *benchOpRepo) UpdateOperationStep(_ context.Context, _ uuid.UUID, _ string, _ domainOperation.StepStatus, _ int, _, _ *string, _, _ *time.Time) error {
	return nil
}

func TestConcurrentOperationQueueUnderLoad(t *testing.T) {
	repo := newBenchOpRepo()
	svc := serviceOperation.NewService(repo, nil)

	var wg sync.WaitGroup
	concurrency := 20
	opsPerWorker := 50

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < opsPerWorker; j++ {
				op, err := svc.CreateOperation(context.Background(), serviceOperation.CreateOperationInput{
					Type:           "start_instance",
					ResourceType:   "instance",
					ResourceID:     uuid.New(),
					IdempotencyKey: fmt.Sprintf("bench-op-%d-%d", workerID, j),
					Retryable:      true,
					MaxRetries:     3,
					TraceID:        "bench-trace",
				})
				if err != nil {
					t.Errorf("failed to create operation: %v", err)
					return
				}

				// Update progress
				_ = svc.UpdateProgress(context.Background(), op.ID, domainOperation.StatusRunning, "booting", "operations.start.booting", 50, nil)

				// Complete
				_ = svc.CompleteOperation(context.Background(), op.ID)
			}

		}(i)
	}

	wg.Wait()

	totalExpected := concurrency * opsPerWorker
	if len(repo.ops) != totalExpected {
		t.Fatalf("expected %d operations, got %d", totalExpected, len(repo.ops))
	}
}
