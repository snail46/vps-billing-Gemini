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

type AdminAuthHandler struct {
	adminSvc *serviceIdentity.AdminService
	sessMgr  *session.Manager
}

func NewAdminAuthHandler(adminSvc *serviceIdentity.AdminService, sessMgr *session.Manager) *AdminAuthHandler {
	return &AdminAuthHandler{
		adminSvc: adminSvc,
		sessMgr:  sessMgr,
	}
}

type AdminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Code     string `json:"code,omitempty"`
}

func (h *AdminAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req AdminLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, r, http.StatusUnprocessableEntity, "INVALID_REQUEST", "errors.invalid_request", "invalid json body")
		return
	}

	ip := r.RemoteAddr
	userAgent := r.UserAgent()

	var admin *domainIdentity.Admin
	var sess *session.Session
	var err error

	if req.Code != "" {
		admin, sess, err = h.adminSvc.LoginWith2FA(r.Context(), req.Email, req.Password, req.Code, ip, userAgent)
	} else {
		admin, sess, err = h.adminSvc.Login(r.Context(), req.Email, req.Password, ip, userAgent)
	}

	if err != nil {
		if errors.Is(err, domainIdentity.ErrTwoFactorRequired) {
			httputil.Error(w, r, http.StatusUnauthorized, "TWO_FACTOR_REQUIRED", "errors.two_factor_required", "two-factor authentication code is required")
			return
		}
		if errors.Is(err, domainIdentity.ErrInvalidTwoFactorCode) {
			httputil.Error(w, r, http.StatusUnauthorized, "INVALID_TWO_FACTOR_CODE", "errors.invalid_two_factor_code", "invalid two-factor authentication code")
			return
		}
		if errors.Is(err, domainIdentity.ErrInvalidCredentials) {
			httputil.Error(w, r, http.StatusUnauthorized, "INVALID_CREDENTIALS", "errors.invalid_credentials", "invalid email or password")
			return
		}
		if errors.Is(err, domainIdentity.ErrAccountSuspended) {
			httputil.Error(w, r, http.StatusForbidden, "ACCOUNT_SUSPENDED", "errors.account_suspended", "admin account is suspended")
			return
		}
		if errors.Is(err, domainIdentity.ErrAccountDisabled) {
			httputil.Error(w, r, http.StatusForbidden, "ACCOUNT_DISABLED", "errors.account_disabled", "admin account is disabled")
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	// Set HttpOnly session cookie and CSRF cookie
	h.sessMgr.SetSessionCookie(w, session.AdminSessionCookieName, sess.Token, session.AdminSessionTTL)
	h.sessMgr.SetCSRFCookie(w, sess.CSRFToken, session.AdminSessionTTL)

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"admin": map[string]any{
			"id":                 admin.ID,
			"email":              admin.Email,
			"display_name":       admin.DisplayName,
			"status":             admin.Status,
			"two_factor_enabled": admin.TwoFactorEnabled,
		},
		"roles":       sess.Roles,
		"permissions": sess.Permissions,
		"csrf_token":  sess.CSRFToken,
	})
}

func (h *AdminAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetContextSession(r.Context())
	if sess != nil {
		_ = h.adminSvc.Logout(r.Context(), sess.Token, r.RemoteAddr, r.UserAgent())
	}

	h.sessMgr.ClearSessionCookie(w, session.AdminSessionCookieName)
	h.sessMgr.ClearCSRFCookie(w)

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"logged_out": true,
	})
}

func (h *AdminAuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetContextAdmin(r.Context())
	sess := middleware.GetContextSession(r.Context())
	if admin == nil || sess == nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "unauthenticated admin")
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"admin": map[string]any{
			"id":                 admin.ID,
			"email":              admin.Email,
			"display_name":       admin.DisplayName,
			"status":             admin.Status,
			"two_factor_enabled": admin.TwoFactorEnabled,
			"last_login_at":      admin.LastLoginAt,
			"created_at":         admin.CreatedAt,
		},
		"roles":       sess.Roles,
		"permissions": sess.Permissions,
	})
}

func (h *AdminAuthHandler) Setup2FA(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetContextAdmin(r.Context())
	if admin == nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "unauthenticated admin")
		return
	}

	secret, uri, err := h.adminSvc.SetupTwoFactor(r.Context(), admin.ID)
	if err != nil {
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"secret":      secret,
		"otpauth_uri": uri,
	})
}

type Enable2FARequest struct {
	Secret string `json:"secret"`
	Code   string `json:"code"`
}

func (h *AdminAuthHandler) Enable2FA(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetContextAdmin(r.Context())
	if admin == nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "unauthenticated admin")
		return
	}

	var req Enable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, r, http.StatusUnprocessableEntity, "INVALID_REQUEST", "errors.invalid_request", "invalid json body")
		return
	}

	if err := h.adminSvc.EnableTwoFactor(r.Context(), admin.ID, req.Secret, req.Code); err != nil {
		if errors.Is(err, domainIdentity.ErrInvalidTwoFactorCode) {
			httputil.Error(w, r, http.StatusBadRequest, "INVALID_TWO_FACTOR_CODE", "errors.invalid_two_factor_code", "invalid two-factor code")
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"success": true,
		"message": "Two-factor authentication enabled successfully",
	})
}

type Disable2FARequest struct {
	Code string `json:"code"`
}

func (h *AdminAuthHandler) Disable2FA(w http.ResponseWriter, r *http.Request) {
	admin := middleware.GetContextAdmin(r.Context())
	if admin == nil {
		httputil.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "unauthenticated admin")
		return
	}

	var req Disable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, r, http.StatusUnprocessableEntity, "INVALID_REQUEST", "errors.invalid_request", "invalid json body")
		return
	}

	if err := h.adminSvc.DisableTwoFactor(r.Context(), admin.ID, req.Code); err != nil {
		if errors.Is(err, domainIdentity.ErrInvalidTwoFactorCode) {
			httputil.Error(w, r, http.StatusBadRequest, "INVALID_TWO_FACTOR_CODE", "errors.invalid_two_factor_code", "invalid two-factor code")
			return
		}
		httputil.Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.internal_error", err.Error())
		return
	}

	httputil.JSON(w, r, http.StatusOK, map[string]any{
		"success": true,
		"message": "Two-factor authentication disabled successfully",
	})
}
