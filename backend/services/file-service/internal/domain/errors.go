package domain

import (
	"errors"
	"path"
	"strings"
)

var (
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid")
	ErrConflict = errors.New("conflict")
)

func SafeFileName(value string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	base := path.Base(normalized)
	if base == "." || base == "/" || base == "" {
		return "file"
	}
	return base
}
