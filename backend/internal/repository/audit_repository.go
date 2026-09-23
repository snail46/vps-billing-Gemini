package repository

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"vps-billing/internal/domain/audit"
)

type PostgresAuditRepository struct {
	q *Queries
}

func NewPostgresAuditRepository(q *Queries) *PostgresAuditRepository {
	return &PostgresAuditRepository{q: q}
}

func (r *PostgresAuditRepository) Create(ctx context.Context, event *audit.AuditEvent) error {
	var ipAddr *netip.Addr
	if event.IPAddress != "" {
		if parsed, err := netip.ParseAddr(event.IPAddress); err == nil {
			ipAddr = &parsed
		}
	}

	var actorID pgtype.UUID
	if event.ActorID != nil {
		actorID = toPgUUID(*event.ActorID)
	}

	var resourceID pgtype.UUID
	if event.ResourceID != nil {
		resourceID = toPgUUID(*event.ResourceID)
	}

	row, err := r.q.CreateAuditEvent(ctx, CreateAuditEventParams{
		ID:           toPgUUID(event.ID),
		ActorType:    string(event.ActorType),
		ActorID:      actorID,
		Action:       event.Action,
		ResourceType: event.ResourceType,
		ResourceID:   resourceID,
		BeforeData:   event.BeforeData,
		AfterData:    event.AfterData,
		IpAddress:    ipAddr,
		UserAgent:    toPgText(event.UserAgent),
		RequestID:    toPgText(event.RequestID),
		TraceID:      toPgText(event.TraceID),
	})
	if err != nil {
		return fmt.Errorf("failed to create audit event: %w", err)
	}

	event.CreatedAt = row.CreatedAt.Time
	return nil
}

func (r *PostgresAuditRepository) List(ctx context.Context, limit int32, offset int32) ([]audit.AuditEvent, error) {
	rows, err := r.q.ListAuditEvents(ctx, ListAuditEventsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list audit events: %w", err)
	}

	events := make([]audit.AuditEvent, len(rows))
	for i, row := range rows {
		var actorID *uuid.UUID
		if row.ActorID.Valid {
			id := fromPgUUID(row.ActorID)
			actorID = &id
		}

		var resourceID *uuid.UUID
		if row.ResourceID.Valid {
			id := fromPgUUID(row.ResourceID)
			resourceID = &id
		}

		ipStr := ""
		if row.IpAddress != nil {
			ipStr = row.IpAddress.String()
		}

		events[i] = audit.AuditEvent{
			ID:           fromPgUUID(row.ID),
			ActorType:    audit.ActorType(row.ActorType),
			ActorID:      actorID,
			Action:       row.Action,
			ResourceType: row.ResourceType,
			ResourceID:   resourceID,
			BeforeData:   row.BeforeData,
			AfterData:    row.AfterData,
			IPAddress:    ipStr,
			UserAgent:    row.UserAgent.String,
			RequestID:    row.RequestID.String,
			TraceID:      row.TraceID.String,
			CreatedAt:    row.CreatedAt.Time,
		}
	}

	return events, nil
}
