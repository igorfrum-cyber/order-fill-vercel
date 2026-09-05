package domain

import (
	"errors"
	"testing"
)

func TestParseLoginSlugAcceptsLatin(t *testing.T) {
	got, err := ParseLoginSlug("Kristail")
	if err != nil {
		t.Fatal(err)
	}
	if got != "kristail" {
		t.Fatalf("got %q", got)
	}
}

func TestParseLoginSlugRejectsCyrillic(t *testing.T) {
	if _, err := ParseLoginSlug("Кристайл"); !errors.Is(err, ErrInvalidLoginSlug) {
		t.Fatalf("got %v", err)
	}
}

func TestParseLoginSlugRejectsReserved(t *testing.T) {
	if _, err := ParseLoginSlug("admin"); !errors.Is(err, ErrInvalidLoginSlug) {
		t.Fatalf("got %v", err)
	}
}

func TestParseLoginSlugRejectsEmpty(t *testing.T) {
	if _, err := ParseLoginSlug("  "); !errors.Is(err, ErrInvalidLoginSlug) {
		t.Fatalf("got %v", err)
	}
}

func TestParseLoginSlugRejectsHyphenEdges(t *testing.T) {
	if _, err := ParseLoginSlug("-acme"); !errors.Is(err, ErrInvalidLoginSlug) {
		t.Fatalf("leading: %v", err)
	}
	if _, err := ParseLoginSlug("acme-"); !errors.Is(err, ErrInvalidLoginSlug) {
		t.Fatalf("trailing: %v", err)
	}
}
