package audit

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ActorType string

const (
	ActorTypeUser   ActorType = "user"
	ActorTypeAdmin  ActorType = "admin"
	ActorTypeSystem ActorType = "system"
)

type AuditEvent struct {
	ID           uuid.UUID       `json:"id"`
	ActorType    ActorType       `json:"actor_type"`
	ActorID      *uuid.UUID      `json:"actor_id,omitempty"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   *uuid.UUID      `json:"resource_id,omitempty"`
	BeforeData   json.RawMessage `json:"before_data,omitempty"`
	AfterData    json.RawMessage `json:"after_data,omitempty"`
	IPAddress    string          `json:"ip_address,omitempty"`
	UserAgent    string          `json:"user_agent,omitempty"`
	RequestID    string          `json:"request_id,omitempty"`
	TraceID      string          `json:"trace_id,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}
