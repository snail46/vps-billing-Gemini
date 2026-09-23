package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	domainTicket "vps-billing/internal/domain/ticket"
	"vps-billing/internal/httputil"
	"vps-billing/internal/middleware"
	serviceTicket "vps-billing/internal/service/ticket"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TicketHandler struct {
	ticketSvc *serviceTicket.Service
}

func NewTicketHandler(ticketSvc *serviceTicket.Service) *TicketHandler {
	return &TicketHandler{ticketSvc: ticketSvc}
}

type CreateTicketRequest struct {
	Subject  string `json:"subject"`
	Priority string `json:"priority"`
	Message  string `json:"message"`
}

func (h *TicketHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	h.Create(w, r)
}

func (h *TicketHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "login required")
		return
	}

	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, r, http.StatusUnprocessableEntity, "INVALID_REQUEST", "errors.validation_failed", "invalid json body")
		return
	}

	p := domainTicket.PriorityMedium
	if req.Priority == "low" {
		p = domainTicket.PriorityLow
	} else if req.Priority == "high" {
		p = domainTicket.PriorityHigh
	}

	t, err := h.ticketSvc.CreateTicket(r.Context(), userID, req.Subject, p, req.Message)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "CREATE_FAILED", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusCreated, map[string]any{"ticket": t})
}

func (h *TicketHandler) ListMyTickets(w http.ResponseWriter, r *http.Request) {
	h.ListUser(w, r)
}

func (h *TicketHandler) ListUser(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "login required")
		return
	}

	tickets, err := h.ticketSvc.ListUserTickets(r.Context(), userID)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{"tickets": tickets})
}

func (h *TicketHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	h.Get(w, r)
}

func (h *TicketHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.invalid_id", "invalid ticket id")
		return
	}

	t, err := h.ticketSvc.GetTicket(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, http.StatusNotFound, "NOT_FOUND", "errors.not_found", "ticket not found")
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{"ticket": t})
}

type ReplyTicketRequest struct {
	Message string `json:"message"`
}

func (h *TicketHandler) UserReplyTicket(w http.ResponseWriter, r *http.Request) {
	h.UserReply(w, r)
}

func (h *TicketHandler) UserReply(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "login required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.invalid_id", "invalid ticket id")
		return
	}

	var req ReplyTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, r, http.StatusUnprocessableEntity, "INVALID_REQUEST", "errors.validation_failed", "invalid json body")
		return
	}

	msg, err := h.ticketSvc.UserReply(r.Context(), id, userID, req.Message)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "REPLY_FAILED", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusCreated, map[string]any{"message": msg})
}

func (h *TicketHandler) AdminListTickets(w http.ResponseWriter, r *http.Request) {
	h.AdminList(w, r)
}

func (h *TicketHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	tickets, err := h.ticketSvc.ListAllTickets(r.Context(), limit, offset)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{"tickets": tickets})
}

func (h *TicketHandler) AdminReplyTicket(w http.ResponseWriter, r *http.Request) {
	h.AdminReply(w, r)
}

func (h *TicketHandler) AdminReply(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.AdminIDFromContext(r.Context())
	if adminID == uuid.Nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "admin login required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.invalid_id", "invalid ticket id")
		return
	}

	var req ReplyTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, r, http.StatusUnprocessableEntity, "INVALID_REQUEST", "errors.validation_failed", "invalid json body")
		return
	}

	msg, err := h.ticketSvc.AdminReply(r.Context(), id, adminID, req.Message)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "REPLY_FAILED", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusCreated, map[string]any{"message": msg})
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}

func (h *TicketHandler) AdminUpdateTicketStatus(w http.ResponseWriter, r *http.Request) {
	h.AdminUpdateStatus(w, r)
}

func (h *TicketHandler) AdminUpdateStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, http.StatusBadRequest, "INVALID_ID", "errors.invalid_id", "invalid ticket id")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, r, http.StatusUnprocessableEntity, "INVALID_REQUEST", "errors.validation_failed", "invalid json body")
		return
	}

	status := domainTicket.StatusOpen
	if req.Status == "closed" {
		status = domainTicket.StatusClosed
	}

	if err := h.ticketSvc.UpdateStatus(r.Context(), id, status); err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{"status": string(status)})
}
