package middleware

import (
	"crypto/subtle"
	"net/http"

	"vps-billing/internal/service/session"
)

func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only check mutating methods
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		sess := GetContextSession(r.Context())
		if sess == nil {
			// If request is unauthenticated, CSRF is not required for public auth (login, register)
			next.ServeHTTP(w, r)
			return
		}

		// Check if request is authenticated via an explicit Authorization: Bearer token header.
		// Under OWASP CSRF prevention standards, requests authenticated via explicit custom headers
		// (such as Bearer tokens stored in JS/localStorage) cannot be forged cross-site by ambient browser cookies.
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			bearerToken := authHeader[7:]
			if subtle.ConstantTimeCompare([]byte(bearerToken), []byte(sess.Token)) == 1 {
				next.ServeHTTP(w, r)
				return
			}
		}

		// For cookie-based ambient sessions, validate CSRF token from header or cookie
		clientToken := r.Header.Get("X-CSRF-Token")
		if clientToken == "" {
			clientToken = r.Header.Get("X-XSRF-Token")
		}

		// Fallback check against actor-specific cookie
		if clientToken == "" {
			if sess.ActorType == "admin" {
				if c, err := r.Cookie(session.AdminCSRFCookieName); err == nil && c.Value != "" {
					clientToken = c.Value
				}
			} else {
				if c, err := r.Cookie(session.UserCSRFCookieName); err == nil && c.Value != "" {
					clientToken = c.Value
				}
			}
			if clientToken == "" {
				if c, err := r.Cookie(session.CSRFCookieName); err == nil && c.Value != "" {
					clientToken = c.Value
				}
			}
		}

		if clientToken == "" || sess.CSRFToken == "" || subtle.ConstantTimeCompare([]byte(clientToken), []byte(sess.CSRFToken)) != 1 {
			httpError(w, r, http.StatusForbidden, "INVALID_CSRF_TOKEN", "errors.invalid_csrf_token", "invalid or missing CSRF token")
			return
		}

		next.ServeHTTP(w, r)
	})
}
