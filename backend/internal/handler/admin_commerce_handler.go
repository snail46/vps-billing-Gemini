package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	domainCommerce "vps-billing/internal/domain/commerce"
	"vps-billing/internal/httputil"
	serviceCommerce "vps-billing/internal/service/commerce"
)

type AdminCommerceHandler struct {
	productSvc *serviceCommerce.ProductService
	orderSvc   *serviceCommerce.OrderService
	walletSvc  *serviceCommerce.WalletService
}

func NewAdminCommerceHandler(
	productSvc *serviceCommerce.ProductService,
	orderSvc *serviceCommerce.OrderService,
	walletSvc *serviceCommerce.WalletService,
) *AdminCommerceHandler {
	return &AdminCommerceHandler{
		productSvc: productSvc,
		orderSvc:   orderSvc,
		walletSvc:  walletSvc,
	}
}

func (h *AdminCommerceHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.productSvc.ListAllProducts(r.Context())
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}
	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"products": products,
	})
}

func (h *AdminCommerceHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var p domainCommerce.Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		httputil.Error(w, r, http.StatusUnprocessableEntity, "INVALID_REQUEST", "errors.validation_failed", "invalid json body")
		return
	}

	created, err := h.productSvc.CreateProduct(r.Context(), &p)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusCreated, map[string]any{
		"product": created,
	})
}

func (h *AdminCommerceHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var p domainCommerce.Plan
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		httputil.Error(w, r, http.StatusUnprocessableEntity, "INVALID_REQUEST", "errors.validation_failed", "invalid json body")
		return
	}

	created, err := h.productSvc.CreatePlan(r.Context(), &p)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusCreated, map[string]any{
		"plan": created,
	})
}

func (h *AdminCommerceHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.orderSvc.ListAllOrders(r.Context())
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}
	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"orders": orders,
	})
}

func (h *AdminCommerceHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	invoices, err := h.walletSvc.ListAllInvoices(r.Context())
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}
	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"invoices": invoices,
	})
}

func (h *AdminCommerceHandler) ListLedger(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	txs, err := h.walletSvc.ListLedgerTransactions(r.Context(), limit, offset)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"transactions": txs,
	})
}
