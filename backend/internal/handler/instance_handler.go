package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	domainInfrastructure "vps-billing/internal/domain/infrastructure"
	domainOperation "vps-billing/internal/domain/operation"
	"vps-billing/internal/httputil"
	"vps-billing/internal/middleware"
	serviceOperation "vps-billing/internal/service/operation"
	serviceSubscription "vps-billing/internal/service/subscription"
)

type InstanceHandler struct {
	infraRepo domainInfrastructure.InfrastructureRepository
	subSvc    *serviceSubscription.SubscriptionService
	opSvc     *serviceOperation.Service
	mu        sync.Mutex
}

func NewInstanceHandler(
	infraRepo domainInfrastructure.InfrastructureRepository,
	subSvc *serviceSubscription.SubscriptionService,
	opSvc *serviceOperation.Service,
) *InstanceHandler {
	return &InstanceHandler{
		infraRepo: infraRepo,
		subSvc:    subSvc,
		opSvc:     opSvc,
	}
}

func (h *InstanceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	subs, err := h.subSvc.ListUserSubscriptions(r.Context(), userID)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	var instances []*domainInfrastructure.Instance
	for _, sub := range subs {
		inst, err := h.infraRepo.GetInstanceBySubscriptionID(r.Context(), sub.ID)
		if err == nil && inst != nil {
			instances = append(instances, inst)
		}
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"instances": instances,
	})
}

func (h *InstanceHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.validation_failed", "invalid instance id")
		return
	}

	inst, err := h.infraRepo.GetInstanceByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainInfrastructure.ErrInstanceNotFound) {
			httputil.Error(w, r, http.StatusNotFound, "INSTANCE_NOT_FOUND", "errors.not_found", "instance not found")
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	// Verify ownership via subscription
	sub, err := h.subSvc.GetSubscriptionByID(r.Context(), userID, inst.SubscriptionID)
	if err != nil || sub == nil {
		httputil.Error(w, r, http.StatusNotFound, "INSTANCE_NOT_FOUND", "errors.not_found", "instance not found")
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"instance": inst,
	})
}

func (h *InstanceHandler) handleAction(w http.ResponseWriter, r *http.Request, actionType, desiredState string, steps []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	userID := middleware.UserIDFromContext(r.Context())

	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.validation_failed", "invalid instance id")
		return
	}

	inst, err := h.infraRepo.GetInstanceByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainInfrastructure.ErrInstanceNotFound) {
			httputil.Error(w, r, http.StatusNotFound, "INSTANCE_NOT_FOUND", "errors.not_found", "instance not found")
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	// Ownership check
	sub, err := h.subSvc.GetSubscriptionByID(r.Context(), userID, inst.SubscriptionID)
	if err != nil || sub == nil {
		httputil.Error(w, r, http.StatusNotFound, "INSTANCE_NOT_FOUND", "errors.not_found", "instance not found")
		return
	}

	// Prevent duplicate or conflicting actions: check for active operation (HTTP 409 Conflict)
	existingOps, err := h.opSvc.ListByResource(r.Context(), "instance", inst.ID)
	if err == nil {
		for _, o := range existingOps {
			if o.Status == domainOperation.StatusQueued || o.Status == domainOperation.StatusRunning || o.Status == domainOperation.StatusRetrying {
				httputil.Error(w, r, http.StatusConflict, "OPERATION_CONFLICT", "errors.operation_conflict", "an operation is already in progress on this instance")
				return
			}
		}
	}

	// Create operation (HTTP 202 Accepted)
	idemKey := fmt.Sprintf("%s-%s-%d", actionType, inst.ID.String(), r.Context().Value("timestamp"))
	op, err := h.opSvc.CreateOperation(r.Context(), serviceOperation.CreateOperationInput{
		Type:           actionType,
		ResourceType:   "instance",
		ResourceID:     inst.ID,
		IdempotencyKey: idemKey,
		Retryable:      true,
		MaxRetries:     3,
		TraceID:        fmt.Sprintf("trace-%s", inst.ID.String()[:8]),
		Steps:          steps,
	})
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	// Update instance desired state
	_ = h.infraRepo.UpdateInstanceStates(r.Context(), inst.ID, desiredState, inst.ObservedState, nil)

	httputil.JSON(w, r, http.StatusAccepted, map[string]any{
		"operation_id": op.ID,
		"operation":    op,
	})
}

func (h *InstanceHandler) Start(w http.ResponseWriter, r *http.Request) {
	h.handleAction(w, r, "start_instance", "running", []string{"validate", "submit_provider", "wait_provider", "verify_running", "finish"})
}

func (h *InstanceHandler) Stop(w http.ResponseWriter, r *http.Request) {
	h.handleAction(w, r, "stop_instance", "stopped", []string{"validate", "submit_provider", "wait_provider", "verify_stopped", "finish"})
}

func (h *InstanceHandler) Restart(w http.ResponseWriter, r *http.Request) {
	h.handleAction(w, r, "restart_instance", "running", []string{"validate", "submit_provider", "wait_provider", "verify_running", "finish"})
}

type ReinstallRequest struct {
	Image string `json:"image"`
}

func (h *InstanceHandler) Reinstall(w http.ResponseWriter, r *http.Request) {
	var req ReinstallRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	h.handleAction(w, r, "reinstall_instance", "running", []string{"validate", "lock", "submit_provider", "wait_provider", "sync_network", "verify_running", "finish"})
}
