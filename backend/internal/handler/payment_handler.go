package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	domainCommerce "vps-billing/internal/domain/commerce"
	"vps-billing/internal/httputil"
	serviceCommerce "vps-billing/internal/service/commerce"
)

type PaymentHandler struct {
	paymentSvc *serviceCommerce.PaymentService
}

func NewPaymentHandler(paymentSvc *serviceCommerce.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentSvc: paymentSvc}
}

func (h *PaymentHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	gateway := chi.URLParam(r, "gateway")
	if gateway == "" {
		httputil.Error(w, r, http.StatusBadRequest, "MISSING_GATEWAY", "errors.validation_failed", "missing gateway param")
		return
	}

	sig := r.Header.Get("X-Signature")
	if sig == "" {
		sig = r.Header.Get("X-Webhook-Signature")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "errors.validation_failed", "failed to read body")
		return
	}

	res, err := h.paymentSvc.ProcessWebhook(r.Context(), gateway, body, sig)
	if err != nil {
		if errors.Is(err, domainCommerce.ErrInvalidWebhookSignature) {
			httputil.Error(w, r, http.StatusUnauthorized, "INVALID_SIGNATURE", "errors.unauthorized", "invalid webhook signature")
			return
		}
		if errors.Is(err, domainCommerce.ErrPaymentNotFound) {
			httputil.Error(w, r, http.StatusNotFound, "PAYMENT_NOT_FOUND", "errors.not_found", err.Error())
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "WEBHOOK_FAILED", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"result": res,
	})
}

type SimulateRequest struct {
	PaymentNo string `json:"payment_no"`
}

func (h *PaymentHandler) SimulateFakePayment(w http.ResponseWriter, r *http.Request) {
	var req SimulateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PaymentNo == "" {
		paymentNo := r.URL.Query().Get("payment_no")
		if paymentNo != "" {
			req.PaymentNo = paymentNo
		} else {
			httputil.Error(w, r, http.StatusBadRequest, "MISSING_PAYMENT_NO", "errors.validation_failed", "missing payment_no")
			return
		}
	}

	body, sig, err := h.paymentSvc.GenerateFakeSimulatedWebhook(r.Context(), req.PaymentNo)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "SIMULATION_FAILED", "errors.internal_error", err.Error())
		return
	}

	res, err := h.paymentSvc.ProcessWebhook(r.Context(), "fake", body, sig)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "PROCESS_FAILED", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"result": res,
	})
}
