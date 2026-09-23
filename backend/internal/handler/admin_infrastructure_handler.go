package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	domainInfrastructure "vps-billing/internal/domain/infrastructure"
	"vps-billing/internal/httputil"
	serviceIdentity "vps-billing/internal/service/identity"
	serviceOperation "vps-billing/internal/service/operation"
)

type AdminInfrastructureHandler struct {
	infraRepo domainInfrastructure.InfrastructureRepository
	opSvc     *serviceOperation.Service
	userSvc   *serviceIdentity.UserService
	adminSvc  *serviceIdentity.AdminService
}

func NewAdminInfrastructureHandler(
	infraRepo domainInfrastructure.InfrastructureRepository,
	opSvc *serviceOperation.Service,
	userSvc *serviceIdentity.UserService,
	adminSvc *serviceIdentity.AdminService,
) *AdminInfrastructureHandler {
	return &AdminInfrastructureHandler{
		infraRepo: infraRepo,
		opSvc:     opSvc,
		userSvc:   userSvc,
		adminSvc:  adminSvc,
	}
}

func (h *AdminInfrastructureHandler) ListNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.infraRepo.ListNodes(r.Context())
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}
	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"nodes": nodes,
	})
}

func (h *AdminInfrastructureHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.infraRepo.ListProviders(r.Context())
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}
	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"providers": providers,
	})
}

func (h *AdminInfrastructureHandler) ListInstances(w http.ResponseWriter, r *http.Request) {
	instances, err := h.infraRepo.ListInstances(r.Context())
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}
	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"instances": instances,
	})
}

func (h *AdminInfrastructureHandler) ListOperations(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	ops, err := h.opSvc.ListRecentOperations(r.Context(), limit)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}
	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"operations": ops,
	})
}

func (h *AdminInfrastructureHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	users, err := h.userSvc.ListUsers(r.Context(), limit)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}
	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"users": users,
	})
}

func (h *AdminInfrastructureHandler) ListAdmins(w http.ResponseWriter, r *http.Request) {
	admins, err := h.adminSvc.ListAdmins(r.Context())
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}
	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"admins": admins,
	})
}

type CreateProviderRequest struct {
	Name         string `json:"name"`
	ProviderType string `json:"provider_type"`
	Status       string `json:"status"`
}

func (h *AdminInfrastructureHandler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	var req CreateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, r, http.StatusUnprocessableEntity, "INVALID_REQUEST", "errors.validation_failed", "invalid json body")
		return
	}

	p := &domainInfrastructure.Provider{
		ID:           uuid.Must(uuid.NewV7()),
		Name:         req.Name,
		ProviderType: req.ProviderType,
		Status:       "active",
		Config:       []byte(`{}`),
		Capabilities: []byte(`{"create_instance":true,"start_instance":true,"stop_instance":true,"restart_instance":true,"reinstall_instance":true}`),
	}

	created, err := h.infraRepo.CreateProvider(r.Context(), p)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusCreated, map[string]any{
		"provider": created,
	})
}

type CreateNodeRequest struct {
	Name          string `json:"name"`
	ProviderID    string `json:"provider_id"`
	Region        string `json:"region"`
	CPUTotal      int    `json:"cpu_total"`
	MemoryTotalMB int64  `json:"memory_total_mb"`
	DiskTotalGB   int64  `json:"disk_total_gb"`
}

func (h *AdminInfrastructureHandler) CreateNode(w http.ResponseWriter, r *http.Request) {
	var req CreateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, r, http.StatusUnprocessableEntity, "INVALID_REQUEST", "errors.validation_failed", "invalid json body")
		return
	}

	provID, _ := uuid.Parse(req.ProviderID)
	if provID == uuid.Nil {
		provs, _ := h.infraRepo.ListProviders(r.Context())
		if len(provs) > 0 {
			provID = provs[0].ID
		}
	}

	node := &domainInfrastructure.Node{
		ID:            uuid.Must(uuid.NewV7()),
		ProviderID:    provID,
		Name:          req.Name,
		Region:        req.Region,
		Status:        "active",
		CPUTotal:      float64(req.CPUTotal),
		MemoryTotalMB: req.MemoryTotalMB,
		DiskTotalGB:   req.DiskTotalGB,
		Weight:        100,
		Capabilities:  []byte(`{}`),
	}

	created, err := h.infraRepo.CreateNode(r.Context(), node)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusCreated, map[string]any{
		"node": created,
	})
}

func (h *AdminInfrastructureHandler) GetOperation(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.validation_failed", "invalid operation id")
		return
	}

	op, err := h.opSvc.GetOperation(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, http.StatusNotFound, "NOT_FOUND", "errors.not_found", "operation not found")
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"operation": op,
	})
}
