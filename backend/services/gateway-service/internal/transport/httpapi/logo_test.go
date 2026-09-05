package httpapi

import (
	"testing"
)

func TestParseLogoAcceptsPNG(t *testing.T) {
	t.Parallel()
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00}
	kind, err := parseLogo(png)
	if err != nil || kind != "image/png" {
		t.Fatalf("kind=%q err=%v", kind, err)
	}
}

func TestParseLogoRejectsUnknown(t *testing.T) {
	t.Parallel()
	if _, err := parseLogo([]byte("PK\x03\x04")); err == nil {
		t.Fatal("expected error")
	}
}
