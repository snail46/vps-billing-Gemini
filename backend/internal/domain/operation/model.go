package operation

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusQueued          Status = "queued"
	StatusRunning         Status = "running"
	StatusWaitingProvider Status = "waiting_provider"
	StatusWaitingResource Status = "waiting_resource"
	StatusVerifying       Status = "verifying"
	StatusRetrying        Status = "retrying"
	StatusSucceeded       Status = "succeeded"
	StatusFailed          Status = "failed"
	StatusCancelled       Status = "cancelled"
)

type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepSucceeded StepStatus = "succeeded"
	StepFailed    StepStatus = "failed"
	StepSkipped   StepStatus = "skipped"
)

type Operation struct {
	ID                  uuid.UUID        `json:"id"`
	Type                string           `json:"type"`
	ResourceType        string           `json:"resource_type"`
	ResourceID          uuid.UUID        `json:"resource_id"`
	Status              Status           `json:"status"`
	Phase               *string          `json:"phase,omitempty"`
	Progress            int              `json:"progress"`
	MessageKey          *string          `json:"message_key,omitempty"`
	ProviderID          *uuid.UUID       `json:"provider_id,omitempty"`
	ProviderOperationID *string          `json:"provider_operation_id,omitempty"`
	IdempotencyKey      string           `json:"idempotency_key"`
	Retryable           bool             `json:"retryable"`
	RetryCount          int              `json:"retry_count"`
	MaxRetries          int              `json:"max_retries"`
	ErrorCode           *string          `json:"error_code,omitempty"`
	ErrorMessage        *string          `json:"error_message,omitempty"`
	TraceID             string           `json:"trace_id"`
	StartedAt           *time.Time       `json:"started_at,omitempty"`
	FinishedAt          *time.Time       `json:"finished_at,omitempty"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
	Steps               []*OperationStep `json:"steps,omitempty"`
}

type OperationStep struct {
	ID           uuid.UUID  `json:"id"`
	OperationID  uuid.UUID  `json:"operation_id"`
	StepKey      string     `json:"step_key"`
	StepOrder    int        `json:"step_order"`
	Status       StepStatus `json:"status"`
	Progress     int        `json:"progress"`
	Attempt      int        `json:"attempt"`
	ErrorCode    *string    `json:"error_code,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
