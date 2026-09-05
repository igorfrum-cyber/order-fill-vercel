package domain

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrConflict     = errors.New("conflict")
	ErrInvalidTOTP  = errors.New("invalid totp")
	ErrLocked       = errors.New("locked")
)
