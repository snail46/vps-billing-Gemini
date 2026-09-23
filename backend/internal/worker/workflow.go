package worker

import (
	"context"

	domainOperation "vps-billing/internal/domain/operation"
)

type Workflow interface {
	Type() string
	Execute(ctx context.Context, op *domainOperation.Operation) error
}

type WorkflowRegistry struct {
	handlers map[string]Workflow
}

func NewWorkflowRegistry() *WorkflowRegistry {
	return &WorkflowRegistry{
		handlers: make(map[string]Workflow),
	}
}

func (r *WorkflowRegistry) Register(wf Workflow) {
	r.handlers[wf.Type()] = wf
}

func (r *WorkflowRegistry) Get(opType string) (Workflow, bool) {
	wf, ok := r.handlers[opType]
	return wf, ok
}
