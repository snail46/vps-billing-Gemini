package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	domainOperation "vps-billing/internal/domain/operation"
	"vps-billing/internal/httputil"
	serviceOperation "vps-billing/internal/service/operation"
)

type OperationHandler struct {
	opSvc *serviceOperation.Service
}

func NewOperationHandler(opSvc *serviceOperation.Service) *OperationHandler {
	return &OperationHandler{opSvc: opSvc}
}

// Get handles polling for an operation status.
func (h *OperationHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.validation_failed", "invalid operation id")
		return
	}

	op, err := h.opSvc.GetOperation(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainOperation.ErrOperationNotFound) {
			httputil.Error(w, r, http.StatusNotFound, "OPERATION_NOT_FOUND", "errors.not_found", "operation not found")
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"operation": op,
	})
}

// Events handles Server-Sent Events (SSE) for streaming real-time operation progress.
func (h *OperationHandler) Events(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.validation_failed", "invalid operation id")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httputil.Error(w, r, http.StatusInternalServerError, "STREAMING_UNSUPPORTED", "errors.internal_error", "streaming not supported")
		return
	}

	// Fetch current state
	initialOp, err := h.opSvc.GetOperation(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainOperation.ErrOperationNotFound) {
			httputil.Error(w, r, http.StatusNotFound, "OPERATION_NOT_FOUND", "errors.not_found", "operation not found")
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send initial state
	data, _ := json.Marshal(initialOp)
	fmt.Fprintf(w, "event: operation\ndata: %s\n\n", data)
	flusher.Flush()

	// If already in terminal state, finish immediately
	if initialOp.Status == domainOperation.StatusSucceeded ||
		initialOp.Status == domainOperation.StatusFailed ||
		initialOp.Status == domainOperation.StatusCancelled {
		return
	}

	// Subscribe to live events
	events, unsubscribe := h.opSvc.Subscribe(r.Context(), id)
	defer unsubscribe()

	ticker := time.NewTicker(15 * time.Second) // keepalive heartbeat
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case update, ok := <-events:
			if !ok {
				return
			}
			updateData, err := json.Marshal(update)
			if err == nil {
				fmt.Fprintf(w, "event: operation\ndata: %s\n\n", updateData)
				flusher.Flush()
			}
			// Close stream once finished
			if update.Status == domainOperation.StatusSucceeded ||
				update.Status == domainOperation.StatusFailed ||
				update.Status == domainOperation.StatusCancelled {
				return
			}
		}
	}
}
