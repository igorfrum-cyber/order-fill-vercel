package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalid            = errors.New("invalid argument")
	ErrAlreadyExists      = errors.New("already exists")
	ErrFailedPrecond      = errors.New("failed precondition")
	ErrPayloadTooLarge    = errors.New("payload too large")
	ErrAttachmentTooLarge = errors.New("attachment too large")
	ErrInvalidPayload     = errors.New("invalid payload")
)
