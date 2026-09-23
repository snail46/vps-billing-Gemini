package audit

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	domainAudit "vps-billing/internal/domain/audit"
	"vps-billing/internal/logger"
)

type Service struct {
	repo domainAudit.AuditRepository
}

func NewService(repo domainAudit.AuditRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Record(ctx context.Context, event *domainAudit.AuditEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.Must(uuid.NewV7())
	}
	if event.RequestID == "" {
		event.RequestID = logger.GetRequestID(ctx)
	}
	if event.TraceID == "" {
		event.TraceID = logger.GetTraceID(ctx)
	}

	if err := s.repo.Create(ctx, event); err != nil {
		slog.ErrorContext(ctx, "failed to record audit event",
			slog.String("action", event.Action),
			slog.String("error", err.Error()),
		)
		return err
	}

	slog.InfoContext(ctx, "audit event recorded",
		slog.String("action", event.Action),
		slog.String("actor_type", string(event.ActorType)),
		slog.String("audit_id", event.ID.String()),
	)
	return nil
}

func (s *Service) List(ctx context.Context, limit, offset int32) ([]domainAudit.AuditEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset)
}
