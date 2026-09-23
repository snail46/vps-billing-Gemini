package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	domainIdentity "vps-billing/internal/domain/identity"
	"vps-billing/internal/httputil"
	"vps-billing/internal/middleware"
	serviceIdentity "vps-billing/internal/service/identity"
	"vps-billing/internal/service/session"
)

type UserAuthHandler struct {
	userSvc *serviceIdentity.UserService
	sessMgr *session.Manager
}

func NewUserAuthHandler(userSvc *serviceIdentity.UserService, sessMgr *session.Manager) *UserAuthHandler {
	return &UserAuthHandler{
		userSvc: userSvc,
		sessMgr: sessMgr,
	}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Locale   string `json:"locale"`
	Timezone string `json:"timezone"`
}

func (h *UserAuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, r, http.StatusUnprocessableEntity, "INVALID_REQUEST", "errors.invalid_request", "invalid json body")
		return
	}

	ip := r.RemoteAddr
	userAgent := r.UserAgent()

	user, err := h.userSvc.Register(r.Context(), req.Email, req.Password, req.Locale, req.Timezone, ip, userAgent)
	if err != nil {
		if errors.Is(err, domainIdentity.ErrEmailAlreadyExists) {
			httputil.Error(w, r, http.StatusConflict, "EMAIL_ALREADY_EXISTS", "errors.email_already_exists", err.Error())
			return
		}
		httputil.Error(w, r, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "errors.validation_failed", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusCreated, map[string]any{
		"user": map[string]any{
			"id":       user.ID,
			"email":    user.Email,
			"status":   user.Status,
			"locale":   user.Locale,
			"timezone": user.Timezone,
		},
	})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *UserAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, r, http.StatusUnprocessableEntity, "INVALID_REQUEST", "errors.invalid_request", "invalid json body")
		return
	}

	ip := r.RemoteAddr
	userAgent := r.UserAgent()

	user, sess, err := h.userSvc.Login(r.Context(), req.Email, req.Password, ip, userAgent)
	if err != nil {
		if errors.Is(err, domainIdentity.ErrInvalidCredentials) {
			httputil.Error(w, r, http.StatusUnauthorized, "INVALID_CREDENTIALS", "errors.invalid_credentials", "invalid email or password")
			return
		}
		if errors.Is(err, domainIdentity.ErrAccountSuspended) {
			httputil.Error(w, r, http.StatusForbidden, "ACCOUNT_SUSPENDED", "errors.account_suspended", "account is suspended")
			return
		}
		if errors.Is(err, domainIdentity.ErrAccountDisabled) {
			httputil.Error(w, r, http.StatusForbidden, "ACCOUNT_DISABLED", "errors.account_disabled", "account is disabled")
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	// Set HttpOnly session cookie and CSRF cookie
	h.sessMgr.SetSessionCookie(w, session.UserSessionCookieName, sess.Token, session.UserSessionTTL)
	h.sessMgr.SetCSRFCookie(w, sess.CSRFToken, session.UserSessionTTL)

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":       user.ID,
			"email":    user.Email,
			"status":   user.Status,
			"locale":   user.Locale,
			"timezone": user.Timezone,
		},
		"csrf_token": sess.CSRFToken,
	})
}

func (h *UserAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetContextSession(r.Context())
	if sess != nil {
		_ = h.userSvc.Logout(r.Context(), sess.Token, r.RemoteAddr, r.UserAgent())
	}

	h.sessMgr.ClearSessionCookie(w, session.UserSessionCookieName)
	h.sessMgr.ClearCSRFCookie(w)

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"logged_out": true,
	})
}

func (h *UserAuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetContextUser(r.Context())
	if user == nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "unauthenticated")
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":                user.ID,
			"email":             user.Email,
			"status":            user.Status,
			"locale":            user.Locale,
			"timezone":          user.Timezone,
			"email_verified_at": user.EmailVerifiedAt,
			"last_login_at":     user.LastLoginAt,
			"created_at":        user.CreatedAt,
		},
	})
}
