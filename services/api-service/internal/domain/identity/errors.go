package identity

import "errors"

var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrInvalidLoginSlug = errors.New("invalid login slug")
)
