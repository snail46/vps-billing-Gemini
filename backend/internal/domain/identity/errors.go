package identity

import "errors"

var (
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrEmailAlreadyExists   = errors.New("email is already registered")
	ErrUserNotFound         = errors.New("user not found")
	ErrAdminNotFound        = errors.New("admin not found")
	ErrAccountSuspended     = errors.New("account is suspended")
	ErrAccountDisabled      = errors.New("account is disabled")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrInvalidCSRFToken     = errors.New("invalid or missing CSRF token")
	ErrInvalidSession       = errors.New("session is invalid or expired")
	ErrTwoFactorRequired    = errors.New("two-factor authentication code is required")
	ErrInvalidTwoFactorCode = errors.New("invalid two-factor authentication code")
)
