package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	domainIdentity "vps-billing/internal/domain/identity"
	"vps-billing/internal/logger"
	serviceIdentity "vps-billing/internal/service/identity"
	"vps-billing/internal/service/session"
)

type userContextKey string
type adminContextKey string
type sessionContextKey string

const (
	UserCtxKey    userContextKey    = "user_ctx"
	AdminCtxKey   adminContextKey   = "admin_ctx"
	SessionCtxKey sessionContextKey = "session_ctx"
)

func UserAuth(sessMgr *session.Manager, userSvc *serviceIdentity.UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string
			if cookie, err := r.Cookie(session.UserSessionCookieName); err == nil && cookie.Value != "" {
				token = cookie.Value
			} else if authHeader := r.Header.Get("Authorization"); len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			}

			if token == "" {
				httpError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "authentication required")
				return
			}

			sess, err := sessMgr.GetSession(r.Context(), token)
			if err != nil || sess == nil || sess.ActorType != "user" {
				// Admin session cannot access User endpoints
				httpError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "invalid user session")
				return
			}

			user, err := userSvc.GetMe(r.Context(), sess.ActorID)
			if err != nil {
				if err == domainIdentity.ErrAccountSuspended {
					httpError(w, r, http.StatusForbidden, "ACCOUNT_SUSPENDED", "errors.account_suspended", "account suspended")
					return
				}
				if err == domainIdentity.ErrAccountDisabled {
					httpError(w, r, http.StatusForbidden, "ACCOUNT_DISABLED", "errors.account_disabled", "account disabled")
					return
				}
				httpError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "user account inactive or not found")
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, UserCtxKey, user)
			ctx = context.WithValue(ctx, SessionCtxKey, sess)
			ctx = context.WithValue(ctx, logger.ActorKey, user.ID.String())

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminAuth(sessMgr *session.Manager, adminSvc *serviceIdentity.AdminService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string
			if cookie, err := r.Cookie(session.AdminSessionCookieName); err == nil && cookie.Value != "" {
				token = cookie.Value
			} else if authHeader := r.Header.Get("Authorization"); len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			}

			if token == "" {
				httpError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "admin authentication required")
				return
			}

			sess, err := sessMgr.GetSession(r.Context(), token)
			if err != nil || sess == nil || sess.ActorType != "admin" {
				// User session cannot access Admin endpoints
				httpError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "invalid admin session")
				return
			}

			admin, roles, perms, err := adminSvc.GetMe(r.Context(), sess.ActorID)
			if err != nil {
				httpError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "admin account inactive or not found")
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, AdminCtxKey, admin)
			ctx = context.WithValue(ctx, SessionCtxKey, sess)
			ctx = context.WithValue(ctx, logger.ActorKey, admin.ID.String())

			// Update roles and perms in session if fresh
			sess.Roles = roles
			sess.Permissions = perms

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequirePermission(adminSvc *serviceIdentity.AdminService, requiredPermission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			admin, ok := r.Context().Value(AdminCtxKey).(*domainIdentity.Admin)
			if !ok || admin == nil {
				httpError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "errors.unauthorized", "unauthenticated admin")
				return
			}

			hasPerm, err := adminSvc.HasPermission(r.Context(), admin.ID, requiredPermission)
			if err != nil || !hasPerm {
				httpError(w, r, http.StatusForbidden, "FORBIDDEN", "errors.forbidden", "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func GetContextUser(ctx context.Context) *domainIdentity.User {
	if u, ok := ctx.Value(UserCtxKey).(*domainIdentity.User); ok {
		return u
	}
	return nil
}

func UserIDFromContext(ctx context.Context) uuid.UUID {
	u := GetContextUser(ctx)
	if u == nil {
		return uuid.Nil
	}
	return u.ID
}

func GetContextAdmin(ctx context.Context) *domainIdentity.Admin {
	if a, ok := ctx.Value(AdminCtxKey).(*domainIdentity.Admin); ok {
		return a
	}
	return nil
}

func AdminIDFromContext(ctx context.Context) uuid.UUID {
	a := GetContextAdmin(ctx)
	if a == nil {
		return uuid.Nil
	}
	return a.ID
}

func GetContextSession(ctx context.Context) *session.Session {
	if s, ok := ctx.Value(SessionCtxKey).(*session.Session); ok {
		return s
	}
	return nil
}

func httpError(w http.ResponseWriter, r *http.Request, statusCode int, code, messageKey, details string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	resp := map[string]any{
		"success": false,
		"error": map[string]any{
			"code":        code,
			"message_key": messageKey,
			"details":     details,
		},
		"request_id": logger.GetRequestID(r.Context()),
	}

	_ = jsonEncode(w, resp)
}
