package domain

import "errors"

var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrInvalid          = errors.New("invalid")
	ErrInvalidLoginSlug = errors.New("invalid login slug")
	ErrInvalidLogo      = errors.New("invalid logo")
)

type MatchingMode string

const (
	MatchingModeStandard MatchingMode = "standard"
	MatchingModeSmart    MatchingMode = "smart"
)

func ParseMatchingMode(raw string) MatchingMode {
	if raw == string(MatchingModeSmart) {
		return MatchingModeSmart
	}
	return MatchingModeStandard
}
