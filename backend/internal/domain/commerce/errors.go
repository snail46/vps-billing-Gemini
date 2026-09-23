package commerce

import "errors"

var (
	ErrProductNotFound         = errors.New("product not found")
	ErrPlanNotFound            = errors.New("plan not found")
	ErrPlanInactive            = errors.New("plan is not active")
	ErrOrderNotFound           = errors.New("order not found")
	ErrInvalidOrderStatus      = errors.New("invalid order status for operation")
	ErrEmptyOrderItems         = errors.New("order must contain at least one item")
	ErrPaymentNotFound         = errors.New("payment not found")
	ErrPaymentAlreadyProcessed = errors.New("payment has already been processed")
	ErrInvalidPaymentAmount    = errors.New("payment amount does not match order")
	ErrInvalidCurrency         = errors.New("currency mismatch")
	ErrInvalidWebhookSignature = errors.New("invalid webhook signature")
	ErrIdempotencyConflict     = errors.New("idempotency key conflict")
	ErrInsufficientBalance     = errors.New("insufficient wallet balance")
	ErrInvoiceNotFound         = errors.New("invoice not found")
	ErrWalletNotFound          = errors.New("wallet not found")
)
