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

type WalletInvoiceHandler struct {
	walletSvc *serviceCommerce.WalletService
}

func NewWalletInvoiceHandler(walletSvc *serviceCommerce.WalletService) *WalletInvoiceHandler {
	return &WalletInvoiceHandler{walletSvc: walletSvc}
}

func (h *WalletInvoiceHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	currency := r.URL.Query().Get("currency")
	if currency == "" {
		currency = "USD"
	}

	wallet, err := h.walletSvc.GetUserWallet(r.Context(), userID, currency)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"wallet": wallet,
	})
}

type DepositRequest struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

func (h *WalletInvoiceHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	var req DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_REQUEST", "errors.validation_failed", err.Error())
		return
	}

	if req.AmountMinor <= 0 {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_AMOUNT", "errors.validation_failed", "amount must be greater than zero")
		return
	}

	if req.Currency == "" {
		req.Currency = "USD"
	}

	wallet, err := h.walletSvc.Deposit(r.Context(), userID, req.AmountMinor, req.Currency)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"wallet": wallet,
	})
}

func (h *WalletInvoiceHandler) ListUserLedger(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	entries, err := h.walletSvc.ListUserLedgerEntries(r.Context(), userID)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"ledger_entries": entries,
	})
}

func (h *WalletInvoiceHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	invoices, err := h.walletSvc.ListUserInvoices(r.Context(), userID)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"invoices": invoices,
	})
}

func (h *WalletInvoiceHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user authentication required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.validation_failed", "invalid invoice id")
		return
	}

	inv, err := h.walletSvc.GetInvoiceByID(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, domainCommerce.ErrInvoiceNotFound) {
			httputil.Error(w, r, http.StatusNotFound, "INVOICE_NOT_FOUND", "errors.not_found", "invoice not found")
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"invoice": inv,
	})
}
