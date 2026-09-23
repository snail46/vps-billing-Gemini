package handler

import (
	"net/http"
	"strconv"

	"vps-billing/internal/httputil"
	auditService "vps-billing/internal/service/audit"
)

type AdminAuditHandler struct {
	auditSvc *auditService.Service
}

func NewAdminAuditHandler(auditSvc *auditService.Service) *AdminAuditHandler {
	return &AdminAuditHandler{
		auditSvc: auditSvc,
	}
}

func (h *AdminAuditHandler) List(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := int32(50)
	offset := int32(0)

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = int32(l)
	}
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = int32(o)
	}

	events, err := h.auditSvc.List(r.Context(), limit, offset)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"items":  events,
		"limit":  limit,
		"offset": offset,
	})
}
