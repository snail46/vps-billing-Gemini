package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"vps-billing/internal/config"
	domainOperation "vps-billing/internal/domain/operation"
	serviceOperation "vps-billing/internal/service/operation"
)

type Worker struct {
	cfg        *config.Config
	db         *pgxpool.Pool
	redis      *redis.Client
	opRepo     domainOperation.OperationRepository
	opSvc      *serviceOperation.Service
	registry   *WorkflowRegistry
	concur     int
	activeJobs sync.WaitGroup
}

func New(
	cfg *config.Config,
	db *pgxpool.Pool,
	redisClient *redis.Client,
	opRepo domainOperation.OperationRepository,
	opSvc *serviceOperation.Service,
	registry *WorkflowRegistry,
) *Worker {
	if registry == nil {
		registry = NewWorkflowRegistry()
	}
	return &Worker{
		cfg:      cfg,
		db:       db,
		redis:    redisClient,
		opRepo:   opRepo,
		opSvc:    opSvc,
		registry: registry,
		concur:   4,
	}
}

func (w *Worker) RegisterWorkflow(wf Workflow) {
	w.registry.Register(wf)
}

func (w *Worker) Start(ctx context.Context) error {
	slog.Info("operation worker engine starting...",
		slog.String("env", w.cfg.AppEnv),
		slog.Int("concurrency", w.concur),
	)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	sem := make(chan struct{}, w.concur)

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker engine shutting down, waiting for active jobs to drain...")
			w.activeJobs.Wait()
			slog.Info("all active jobs drained cleanly")
			return nil

		case <-ticker.C:
			if w.opRepo == nil || w.opSvc == nil {
				continue
			}

			// Fetch pending operations (queued or retrying)
			ops, err := w.opRepo.ListPendingOperations(ctx, w.concur)
			if err != nil {
				slog.Error("failed to poll pending operations", slog.String("error", err.Error()))
				continue
			}

			for _, op := range ops {
				select {
				case <-ctx.Done():
					return nil
				case sem <- struct{}{}:
					w.activeJobs.Add(1)
					go func(targetOp *domainOperation.Operation) {
						defer func() {
							<-sem
							w.activeJobs.Done()
						}()
						w.processOperation(ctx, targetOp)
					}(op)
				}
			}
		}
	}
}

func (w *Worker) processOperation(ctx context.Context, op *domainOperation.Operation) {
	slog.Info("processing operation",
		slog.String("operation_id", op.ID.String()),
		slog.String("type", op.Type),
		slog.String("trace_id", op.TraceID),
	)

	wf, exists := w.registry.Get(op.Type)
	if !exists {
		errMsg := fmt.Sprintf("unsupported operation type: %s", op.Type)
		slog.Error("no workflow registered for operation",
			slog.String("operation_id", op.ID.String()),
			slog.String("type", op.Type),
		)
		_ = w.opSvc.FailOperation(ctx, op.ID, "UNSUPPORTED_OPERATION", errMsg)
		return
	}

	// Mark running
	now := time.Now().UTC()
	_ = w.opSvc.UpdateProgress(ctx, op.ID, domainOperation.StatusRunning, "running", fmt.Sprintf("operations.%s.running", op.Type), 5, &now)

	// Execute workflow
	err := wf.Execute(ctx, op)
	if err != nil {
		slog.Error("operation workflow failed",
			slog.String("operation_id", op.ID.String()),
			slog.String("type", op.Type),
			slog.String("error", err.Error()),
		)

		if op.Retryable && op.RetryCount < op.MaxRetries {
			// Backoff retry
			slog.Warn("retrying operation with backoff",
				slog.String("operation_id", op.ID.String()),
				slog.Int("attempt", op.RetryCount+1),
				slog.Int("max_retries", op.MaxRetries),
			)
			_ = w.opSvc.UpdateProgress(ctx, op.ID, domainOperation.StatusRetrying, "retrying", fmt.Sprintf("operations.%s.retrying", op.Type), op.Progress, nil)
		} else {
			// Terminal failure
			_ = w.opSvc.FailOperation(ctx, op.ID, "WORKFLOW_FAILED", err.Error())
		}
		return
	}

	// Mark completed
	_ = w.opSvc.CompleteOperation(ctx, op.ID)
	slog.Info("operation completed successfully",
		slog.String("operation_id", op.ID.String()),
		slog.String("type", op.Type),
	)
}
