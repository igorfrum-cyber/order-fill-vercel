package domain

import (
	"errors"
	"testing"
)

func TestParseRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role Role
	}{
		{name: "platform admin", role: RolePlatformAdmin},
		{name: "company owner", role: RoleCompanyOwner},
		{name: "company admin", role: RoleCompanyAdmin},
		{name: "purchaser", role: RolePurchaser},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseRole(string(tt.role))
			if err != nil || got != tt.role {
				t.Fatalf("ParseRole() = %q, %v; want %q", got, err, tt.role)
			}
		})
	}
	if _, err := ParseRole("unknown"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unsupported role error = %v, want ErrInvalid", err)
	}
}

func TestParseChristinaProffModeDefaultsAndOptions(t *testing.T) {
	t.Parallel()

	for raw, want := range map[string]ChristinaProffMode{
		"standard": ChristinaProffModeStandard,
		"fast":     ChristinaProffModeFast,
		"compare":  ChristinaProffModeCompare,
		"unknown":  ChristinaProffModeStandard,
	} {
		if got := ParseChristinaProffMode(raw); got != want {
			t.Errorf("ParseChristinaProffMode(%q) = %q, want %q", raw, got, want)
		}
	}
}
