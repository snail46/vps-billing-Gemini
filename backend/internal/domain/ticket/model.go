package ticket

import (
	"time"

	"github.com/google/uuid"
)

type TicketPriority string

const (
	PriorityLow    TicketPriority = "low"
	PriorityMedium TicketPriority = "medium"
	PriorityHigh   TicketPriority = "high"
)

type TicketStatus string

const (
	StatusOpen   TicketStatus = "open"
	StatusClosed TicketStatus = "closed"
)

type Ticket struct {
	ID        uuid.UUID      `json:"id"`
	UserID    uuid.UUID      `json:"user_id"`
	Subject   string         `json:"subject"`
	Priority  TicketPriority `json:"priority"`
	Status    TicketStatus   `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Messages  []Message      `json:"messages,omitempty"`
}

type Message struct {
	ID         uuid.UUID `json:"id"`
	TicketID   uuid.UUID `json:"ticket_id"`
	SenderType string    `json:"sender_type"`
	SenderID   uuid.UUID `json:"sender_id"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}
