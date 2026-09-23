package middleware

import (
	"crypto/subtle"
	"net/http"
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

		clientToken := r.Header.Get("X-CSRF-Token")
		if clientToken == "" || sess.CSRFToken == "" || subtle.ConstantTimeCompare([]byte(clientToken), []byte(sess.CSRFToken)) != 1 {
			httpError(w, r, http.StatusForbidden, "INVALID_CSRF_TOKEN", "errors.invalid_csrf_token", "invalid or missing CSRF token")
			return
		}

		next.ServeHTTP(w, r)
	})
}
