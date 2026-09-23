package operation

import "errors"

var (
	ErrOperationNotFound      = errors.New("operation not found")
	ErrOperationConflict      = errors.New("operation conflict: an active operation already exists for this resource")
	ErrInvalidOperationStatus = errors.New("invalid operation status transition")
	ErrOperationCancelled     = errors.New("operation cancelled")
)
