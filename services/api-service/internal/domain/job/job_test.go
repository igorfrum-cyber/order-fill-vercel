package job

import (
	"errors"
	"testing"
	"time"
)

func TestNewJobAllowsOrderFillWithoutBrandAndMonth(t *testing.T) {
	entity, err := NewJob("job-1", TypeOrderFill, "", "", time.Now(), []InputFile{{ID: "in", Role: RoleSource, Name: "s.xlsx"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entity.Brand != "" || entity.OrderMonth != "" {
		t.Fatalf("expected empty identity, got %+v", entity)
	}
}

func TestNewJobStillRequiresBrandForNorthMerge(t *testing.T) {
	_, err := NewJob("job-1", TypeNorthMerge, "", "", time.Now(), []InputFile{{ID: "in", Role: RoleBlank, Name: "b.xlsx"}})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid job, got %v", err)
	}
}

func TestValidateUploadsRejectsMoreThanTwoOrderFillBlanks(t *testing.T) {
	err := ValidateUploads(TypeOrderFill, []Upload{
		{Role: RoleSource, Name: "s.xlsx", Content: []byte("s")},
		{Role: RoleBlank, Name: "a.xlsx", Content: []byte("a")},
		{Role: RoleBlank, Name: "b.xlsx", Content: []byte("b")},
		{Role: RoleBlank, Name: "c.xlsx", Content: []byte("c")},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid job, got %v", err)
	}
}
