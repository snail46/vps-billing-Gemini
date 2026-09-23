package subscription

import "errors"

var (
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrInvalidSubscriptionStatus = errors.New("invalid subscription status for operation")
	ErrSubscriptionAlreadyActive = errors.New("subscription is already active")
	ErrSubscriptionExpired       = errors.New("subscription has expired or terminated")
)
