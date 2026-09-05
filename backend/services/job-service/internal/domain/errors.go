package domain

import "errors"

var (
	ErrInvalid      = errors.New("invalid job")
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrConflict     = errors.New("conflict")
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
