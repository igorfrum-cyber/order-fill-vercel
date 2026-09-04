package orderfill

import (
	"errors"
	"strings"
	"testing"

	"order-fill/services/document-service/internal/domain/brand"
)

func TestDetectSourceColumnsExplainsWrongFile(t *testing.T) {
	workbook := newFakeWorkbook("Лист_1", [][]string{{"Не то", "совсем"}})
	_, err := DetectSourceColumns(workbook)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if strings.Contains(err.Error(), "article") || strings.Contains(err.Error(), "orderedFact") {
		t.Fatalf("leaked technical column keys: %v", err)
	}
	if !strings.Contains(err.Error(), "1С") {
		t.Fatalf("expected a human 1C message, got %v", err)
	}
}

func TestDetectBlankColumnsExplainsWrongFile(t *testing.T) {
	workbook := newFakeWorkbook("Лист_1", [][]string{{"Не то", "совсем"}})
	_, err := DetectBlankColumns(workbook, brand.Rule("klapp"))
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if strings.Contains(err.Error(), "article") || strings.Contains(err.Error(), "quantity") {
		t.Fatalf("leaked technical column keys: %v", err)
	}
	if !strings.Contains(err.Error(), "бланк") {
		t.Fatalf("expected a human blank message, got %v", err)
	}
}
