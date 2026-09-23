package audit

import "context"

type AuditRepository interface {
	Create(ctx context.Context, event *AuditEvent) error
	List(ctx context.Context, limit int32, offset int32) ([]AuditEvent, error)
}
