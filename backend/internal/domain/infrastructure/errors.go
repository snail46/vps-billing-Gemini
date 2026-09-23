package infrastructure

import "errors"

var (
	ErrProviderNotFound         = errors.New("provider not found")
	ErrNodeGroupNotFound        = errors.New("node group not found")
	ErrNodeNotFound             = errors.New("node not found")
	ErrNoSchedulableNode        = errors.New("no suitable node found with sufficient capacity")
	ErrResourceExhausted        = errors.New("node resources exhausted")
	ErrReservationNotFound      = errors.New("resource reservation not found")
	ErrInvalidReservationStatus = errors.New("invalid resource reservation status")
	ErrReservationExpired       = errors.New("resource reservation has expired")
	ErrInstanceNotFound         = errors.New("instance not found")
	ErrInstanceAlreadyExists    = errors.New("instance already exists")
	ErrNodeOffline              = errors.New("node is not active or offline")
)
