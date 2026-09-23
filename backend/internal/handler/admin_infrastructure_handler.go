package handler

import (
	"net/http"
	"strconv"

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
