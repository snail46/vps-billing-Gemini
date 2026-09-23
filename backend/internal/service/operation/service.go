package operation

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	domainOperation "vps-billing/internal/domain/operation"
)

type CreateOperationInput struct {
	Type           string
	ResourceType   string
	ResourceID     uuid.UUID
	IdempotencyKey string
	Retryable      bool
	MaxRetries     int
	TraceID        string
	Steps          []string
}

type EventBroadcaster interface {
	Publish(ctx context.Context, opID uuid.UUID, event *domainOperation.Operation) error
	Subscribe(ctx context.Context, opID uuid.UUID) (<-chan *domainOperation.Operation, func())
}

// LocalBroadcaster implements in-memory pub-sub for operation updates (with fallback or standalone).
type LocalBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID][]chan *domainOperation.Operation
}

func NewLocalBroadcaster() *LocalBroadcaster {
	return &LocalBroadcaster{
		subscribers: make(map[uuid.UUID][]chan *domainOperation.Operation),
	}
}

func (b *LocalBroadcaster) Publish(ctx context.Context, opID uuid.UUID, event *domainOperation.Operation) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	subs := b.subscribers[opID]
	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			// Non-blocking drop if consumer buffer is full
		}
	}
	return nil
}

func (b *LocalBroadcaster) Subscribe(ctx context.Context, opID uuid.UUID) (<-chan *domainOperation.Operation, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan *domainOperation.Operation, 16)
	b.subscribers[opID] = append(b.subscribers[opID], ch)

	cleanup := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		list := b.subscribers[opID]
		for i, c := range list {
			if c == ch {
				b.subscribers[opID] = append(list[:i], list[i+1:]...)
				close(ch)
				break
			}
		}
		if len(b.subscribers[opID]) == 0 {
			delete(b.subscribers, opID)
		}
	}

	return ch, cleanup
}

type Service struct {
	repo        domainOperation.OperationRepository
	broadcaster EventBroadcaster
	redisClient *redis.Client
	mu          sync.Mutex
}

func NewService(repo domainOperation.OperationRepository, redisClient *redis.Client) *Service {
	return &Service{
		repo:        repo,
		broadcaster: NewLocalBroadcaster(),
		redisClient: redisClient,
	}
}

func (s *Service) CreateOperation(ctx context.Context, in CreateOperationInput) (*domainOperation.Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if in.IdempotencyKey != "" {
		existing, err := s.repo.GetOperationByIdempotencyKey(ctx, in.IdempotencyKey)
		if err == nil && existing != nil {
			return existing, nil
		}
	}

	opID := uuid.New()
	initialPhase := "queued"
	initialMsg := fmt.Sprintf("operations.%s.queued", in.Type)

	op := &domainOperation.Operation{
		ID:             opID,
		Type:           in.Type,
		ResourceType:   in.ResourceType,
		ResourceID:     in.ResourceID,
		Status:         domainOperation.StatusQueued,
		Phase:          &initialPhase,
		Progress:       0,
		MessageKey:     &initialMsg,
		IdempotencyKey: in.IdempotencyKey,
		Retryable:      in.Retryable,
		RetryCount:     0,
		MaxRetries:     in.MaxRetries,
		TraceID:        in.TraceID,
	}

	createdOp, err := s.repo.CreateOperation(ctx, op)
	if err != nil {
		return nil, err
	}

	// Create initial steps if defined
	for i, stepKey := range in.Steps {
		step := &domainOperation.OperationStep{
			ID:          uuid.New(),
			OperationID: opID,
			StepKey:     stepKey,
			StepOrder:   i + 1,
			Status:      domainOperation.StepPending,
			Progress:    0,
			Attempt:     0,
		}
		createdStep, stepErr := s.repo.CreateOperationStep(ctx, step)
		if stepErr == nil {
			createdOp.Steps = append(createdOp.Steps, createdStep)
		}
	}

	// Broadcast queued event
	_ = s.broadcast(ctx, createdOp)

	return createdOp, nil
}

func (s *Service) GetOperation(ctx context.Context, id uuid.UUID) (*domainOperation.Operation, error) {
	return s.repo.GetOperationByID(ctx, id)
}

func (s *Service) ListByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*domainOperation.Operation, error) {
	return s.repo.ListOperationsByResource(ctx, resourceType, resourceID)
}

func (s *Service) UpdateProgress(ctx context.Context, opID uuid.UUID, status domainOperation.Status, phase, messageKey string, progress int, startedAt *time.Time) error {
	if progress > 100 {
		progress = 100
	}
	if progress < 0 {
		progress = 0
	}

	err := s.repo.UpdateOperationProgress(ctx, opID, status, phase, messageKey, progress, startedAt)
	if err != nil {
		return err
	}

	op, err := s.repo.GetOperationByID(ctx, opID)
	if err == nil {
		_ = s.broadcast(ctx, op)
	}
	return nil
}

func (s *Service) UpdateStep(ctx context.Context, opID uuid.UUID, stepKey string, status domainOperation.StepStatus, progress int, errCode, errMsg *string, startedAt, finishedAt *time.Time) error {
	err := s.repo.UpdateOperationStep(ctx, opID, stepKey, status, progress, errCode, errMsg, startedAt, finishedAt)
	if err != nil {
		return err
	}

	op, err := s.repo.GetOperationByID(ctx, opID)
	if err == nil {
		_ = s.broadcast(ctx, op)
	}
	return nil
}

func (s *Service) CompleteOperation(ctx context.Context, opID uuid.UUID) error {
	err := s.repo.CompleteOperation(ctx, opID)
	if err != nil {
		return err
	}

	op, err := s.repo.GetOperationByID(ctx, opID)
	if err == nil {
		_ = s.broadcast(ctx, op)
	}
	return nil
}

func (s *Service) FailOperation(ctx context.Context, opID uuid.UUID, errCode, errMsg string) error {
	err := s.repo.FailOperation(ctx, opID, errCode, errMsg)
	if err != nil {
		return err
	}

	op, err := s.repo.GetOperationByID(ctx, opID)
	if err == nil {
		_ = s.broadcast(ctx, op)
	}
	return nil
}

func (s *Service) SetProviderInfo(ctx context.Context, opID uuid.UUID, providerID uuid.UUID, providerOpID string) error {
	return s.repo.SetOperationProviderInfo(ctx, opID, providerID, providerOpID)
}

func (s *Service) ListRecentOperations(ctx context.Context, limit int) ([]*domainOperation.Operation, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListRecentOperations(ctx, limit)
}

func (s *Service) Subscribe(ctx context.Context, opID uuid.UUID) (<-chan *domainOperation.Operation, func()) {
	return s.broadcaster.Subscribe(ctx, opID)
}

func (s *Service) broadcast(ctx context.Context, op *domainOperation.Operation) error {
	_ = s.broadcaster.Publish(ctx, op.ID, op)

	// Also publish to Redis PubSub if client configured
	if s.redisClient != nil {
		channel := fmt.Sprintf("operation:%s", op.ID.String())
		payload, err := json.Marshal(op)
		if err == nil {
			_ = s.redisClient.Publish(ctx, channel, payload).Err()
		}
	}
	return nil
}
