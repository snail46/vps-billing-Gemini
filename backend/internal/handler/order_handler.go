package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	domainCommerce "vps-billing/internal/domain/commerce"
	"vps-billing/internal/httputil"
	"vps-billing/internal/middleware"
	serviceCommerce "vps-billing/internal/service/commerce"
)

type OrderHandler struct {
	orderSvc   *serviceCommerce.OrderService
	paymentSvc *serviceCommerce.PaymentService
}

func NewOrderHandler(
	orderSvc *serviceCommerce.OrderService,
	paymentSvc *serviceCommerce.PaymentService,
) *OrderHandler {
	return &OrderHandler{
		orderSvc:   orderSvc,
		paymentSvc: paymentSvc,
	}
}

type CreateOrderItemReq struct {
	PlanID   uuid.UUID `json:"plan_id"`
	Quantity int       `json:"quantity"`
}

type CreateOrderRequest struct {
	Items []CreateOrderItemReq `json:"items"`
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, r, http.StatusUnprocessableEntity, "INVALID_REQUEST", "errors.validation_failed", "invalid json body")
		return
	}

	inputs := make([]serviceCommerce.CreateOrderItemInput, len(req.Items))
	for i, it := range req.Items {
		inputs[i] = serviceCommerce.CreateOrderItemInput{
			PlanID:   it.PlanID,
			Quantity: it.Quantity,
		}
	}

	order, err := h.orderSvc.CreateOrder(r.Context(), userID, inputs)
	if err != nil {
		if errors.Is(err, domainCommerce.ErrEmptyOrderItems) || errors.Is(err, domainCommerce.ErrPlanInactive) {
			httputil.Error(w, r, http.StatusBadRequest, "INVALID_ORDER_ITEMS", "errors.validation_failed", err.Error())
			return
		}
		if errors.Is(err, domainCommerce.ErrPlanNotFound) || errors.Is(err, domainCommerce.ErrProductNotFound) {
			httputil.Error(w, r, http.StatusNotFound, "ITEM_NOT_FOUND", "errors.not_found", err.Error())
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusCreated, map[string]any{
		"order": order,
	})
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	orders, err := h.orderSvc.ListUserOrders(r.Context(), userID)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"orders": orders,
	})
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.validation_failed", "invalid order id")
		return
	}

	order, err := h.orderSvc.GetOrderByID(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, domainCommerce.ErrOrderNotFound) {
			httputil.Error(w, r, http.StatusNotFound, "ORDER_NOT_FOUND", "errors.not_found", "order not found")
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"order": order,
	})
}

type PayOrderRequest struct {
	Gateway string `json:"gateway"`
}

func (h *OrderHandler) Pay(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.validation_failed", "invalid order id")
		return
	}

	var req PayOrderRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Gateway == "" {
		req.Gateway = "fake"
	}

	res, err := h.paymentSvc.InitiatePayment(r.Context(), id, req.Gateway)
	if err != nil {
		if errors.Is(err, domainCommerce.ErrOrderNotFound) {
			httputil.Error(w, r, http.StatusNotFound, "ORDER_NOT_FOUND", "errors.not_found", "order not found")
			return
		}
		if errors.Is(err, domainCommerce.ErrInvalidOrderStatus) {
			httputil.Error(w, r, http.StatusBadRequest, "INVALID_STATUS", "errors.invalid_status", "order is not in pending state")
			return
		}
		if errors.Is(err, domainCommerce.ErrPaymentAlreadyProcessed) {
			httputil.Error(w, r, http.StatusConflict, "ALREADY_PAID", "errors.already_paid", "order has already been paid")
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"payment":      res.Payment,
		"checkout_url": res.CheckoutURL,
	})
}

func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.validation_failed", "invalid order id")
		return
	}

	order, err := h.orderSvc.CancelOrder(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, domainCommerce.ErrOrderNotFound) {
			httputil.Error(w, r, http.StatusNotFound, "ORDER_NOT_FOUND", "errors.not_found", "order not found")
			return
		}
		if errors.Is(err, domainCommerce.ErrInvalidOrderStatus) {
			httputil.Error(w, r, http.StatusBadRequest, "INVALID_STATUS", "errors.invalid_status", "order cannot be cancelled")
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"order": order,
	})
}
