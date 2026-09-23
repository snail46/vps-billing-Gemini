package repository

import (
	"context"
	"fmt"
	"time"

	domainTicket "vps-billing/internal/domain/ticket"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TicketRepository struct {
	pool *pgxpool.Pool
}

func NewTicketRepository(pool *pgxpool.Pool) *TicketRepository {
	return &TicketRepository{pool: pool}
}

func (r *TicketRepository) Create(ctx context.Context, t *domainTicket.Ticket, initialMsg string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	t.ID = uuid.Must(uuid.NewV7())
	t.CreatedAt = time.Now().UTC()
	t.UpdatedAt = t.CreatedAt

	_, err = tx.Exec(ctx, `
		INSERT INTO tickets (id, user_id, subject, priority, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, t.ID, t.UserID, t.Subject, string(t.Priority), string(t.Status), t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert ticket: %w", err)
	}

	msgID := uuid.Must(uuid.NewV7())
	_, err = tx.Exec(ctx, `
		INSERT INTO ticket_messages (id, ticket_id, sender_type, sender_id, message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, msgID, t.ID, "user", t.UserID, initialMsg, t.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert initial ticket message: %w", err)
	}

	t.Messages = []domainTicket.Message{
		{
			ID:         msgID,
			TicketID:   t.ID,
			SenderType: "user",
			SenderID:   t.UserID,
			Message:    initialMsg,
			CreatedAt:  t.CreatedAt,
		},
	}

	return tx.Commit(ctx)
}

func (r *TicketRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domainTicket.Ticket, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, subject, priority, status, created_at, updated_at
		FROM tickets
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*domainTicket.Ticket
	for rows.Next() {
		var t domainTicket.Ticket
		var priority, status string
		if err := rows.Scan(&t.ID, &t.UserID, &t.Subject, &priority, &status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Priority = domainTicket.TicketPriority(priority)
		t.Status = domainTicket.TicketStatus(status)
		tickets = append(tickets, &t)
	}
	return tickets, rows.Err()
}

func (r *TicketRepository) ListAll(ctx context.Context, limit, offset int) ([]*domainTicket.Ticket, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, subject, priority, status, created_at, updated_at
		FROM tickets
		ORDER BY updated_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*domainTicket.Ticket
	for rows.Next() {
		var t domainTicket.Ticket
		var priority, status string
		if err := rows.Scan(&t.ID, &t.UserID, &t.Subject, &priority, &status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Priority = domainTicket.TicketPriority(priority)
		t.Status = domainTicket.TicketStatus(status)
		tickets = append(tickets, &t)
	}
	return tickets, rows.Err()
}

func (r *TicketRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainTicket.Ticket, error) {
	var t domainTicket.Ticket
	var priority, status string
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, subject, priority, status, created_at, updated_at
		FROM tickets
		WHERE id = $1
	`, id).Scan(&t.ID, &t.UserID, &t.Subject, &priority, &status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.Priority = domainTicket.TicketPriority(priority)
	t.Status = domainTicket.TicketStatus(status)

	msgRows, err := r.pool.Query(ctx, `
		SELECT id, ticket_id, sender_type, sender_id, message, created_at
		FROM ticket_messages
		WHERE ticket_id = $1
		ORDER BY created_at ASC
	`, id)
	if err != nil {
		return &t, nil
	}
	defer msgRows.Close()

	for msgRows.Next() {
		var m domainTicket.Message
		if err := msgRows.Scan(&m.ID, &m.TicketID, &m.SenderType, &m.SenderID, &m.Message, &m.CreatedAt); err == nil {
			t.Messages = append(t.Messages, m)
		}
	}

	return &t, nil
}

func (r *TicketRepository) AddMessage(ctx context.Context, ticketID uuid.UUID, senderType string, senderID uuid.UUID, message string) (*domainTicket.Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	msgID := uuid.Must(uuid.NewV7())
	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `
		INSERT INTO ticket_messages (id, ticket_id, sender_type, sender_id, message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, msgID, ticketID, senderType, senderID, message, now)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `UPDATE tickets SET updated_at = $1 WHERE id = $2`, now, ticketID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &domainTicket.Message{
		ID:         msgID,
		TicketID:   ticketID,
		SenderType: senderType,
		SenderID:   senderID,
		Message:    message,
		CreatedAt:  now,
	}, nil
}

func (r *TicketRepository) UpdateStatus(ctx context.Context, ticketID uuid.UUID, status domainTicket.TicketStatus) error {
	_, err := r.pool.Exec(ctx, `UPDATE tickets SET status = $1, updated_at = $2 WHERE id = $3`, string(status), time.Now().UTC(), ticketID)
	return err
}
