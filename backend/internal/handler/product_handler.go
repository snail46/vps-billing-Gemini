package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	domainCommerce "vps-billing/internal/domain/commerce"
	"vps-billing/internal/httputil"
	serviceCommerce "vps-billing/internal/service/commerce"
)

type ProductHandler struct {
	productSvc *serviceCommerce.ProductService
}

func NewProductHandler(productSvc *serviceCommerce.ProductService) *ProductHandler {
	return &ProductHandler{productSvc: productSvc}
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.productSvc.ListActiveProducts(r.Context())
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}
	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"products": products,
	})
}

func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.validation_failed", "invalid product id format")
		return
	}

	product, err := h.productSvc.GetProductByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainCommerce.ErrProductNotFound) {
			httputil.Error(w, r, http.StatusNotFound, "PRODUCT_NOT_FOUND", "errors.not_found", "product not found")
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"product": product,
	})
}
