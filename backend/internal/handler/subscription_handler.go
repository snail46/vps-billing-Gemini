package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	domainSubscription "vps-billing/internal/domain/subscription"
	"vps-billing/internal/httputil"
	"vps-billing/internal/middleware"
	serviceSubscription "vps-billing/internal/service/subscription"
)

type SubscriptionHandler struct {
	subSvc *serviceSubscription.SubscriptionService
}

func NewSubscriptionHandler(subSvc *serviceSubscription.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{subSvc: subSvc}
}

func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
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

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"subscriptions": subs,
	})
}

func (h *SubscriptionHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.validation_failed", "invalid subscription id")
		return
	}

	sub, err := h.subSvc.GetSubscriptionByID(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, domainSubscription.ErrSubscriptionNotFound) {
			httputil.Error(w, r, http.StatusNotFound, "SUBSCRIPTION_NOT_FOUND", "errors.not_found", "subscription not found")
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"subscription": sub,
	})
}

type CancelSubRequest struct {
	Immediate bool `json:"immediate"`
}

func (h *SubscriptionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.validation_failed", "invalid subscription id")
		return
	}

	var req CancelSubRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	sub, err := h.subSvc.CancelSubscription(r.Context(), userID, id, req.Immediate)
	if err != nil {
		if errors.Is(err, domainSubscription.ErrSubscriptionNotFound) {
			httputil.Error(w, r, http.StatusNotFound, "SUBSCRIPTION_NOT_FOUND", "errors.not_found", "subscription not found")
			return
		}
		if errors.Is(err, domainSubscription.ErrInvalidSubscriptionStatus) {
			httputil.Error(w, r, http.StatusBadRequest, "INVALID_STATUS", "errors.invalid_status", err.Error())
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"subscription": sub,
	})
}

func (h *SubscriptionHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	subs, err := h.subSvc.ListAllSubscriptions(r.Context())
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}
	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"subscriptions": subs,
	})
}
