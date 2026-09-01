package orderfill

import (
	"errors"
	"testing"
)

func TestDetectBrandReadsNomenclatureGroupFilter(t *testing.T) {
	workbook := newFakeWorkbook("Лист_1", [][]string{
		{"Параметры:"},
		{"", "", `Номенклатура В группе "Кристина" И`},
	})
	got, err := DetectBrand(workbook)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "christina" {
		t.Fatalf("got %q, want christina", got)
	}
}

func TestDetectBrandMapsAngiopharmSkinAndKlappGroups(t *testing.T) {
	cases := []struct {
		filter string
		want   string
	}{
		{`Номенклатура В группе "Ангиофарм " И`, "angiopharm"},
		{`Номенклатура В группе "SKIN SYNERGY" И`, "skin_synergy"},
		{`Номенклатура В группе "KLAPP" И`, "klapp"},
	}
	for _, tc := range cases {
		workbook := newFakeWorkbook("Лист_1", [][]string{{"", "", tc.filter}})
		got, err := DetectBrand(workbook)
		if err != nil {
			t.Fatalf("%s: %v", tc.filter, err)
		}
		if got != tc.want {
			t.Fatalf("%s: got %q, want %q", tc.filter, got, tc.want)
		}
	}
}

func TestDetectBrandRejectsUnknownGroup(t *testing.T) {
	workbook := newFakeWorkbook("Лист_1", [][]string{
		{`Номенклатура В группе "Неизвестный" И`},
	})
	_, err := DetectBrand(workbook)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestDetectBrandRejectsMissingFilter(t *testing.T) {
	workbook := newFakeWorkbook("Лист_1", [][]string{{"Период: 01.08.2025 - 31.07.2026"}})
	_, err := DetectBrand(workbook)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestInferOrderMonthUsesMainPeriodEndPlusTwoMonths(t *testing.T) {
	workbook := newFakeWorkbook("Лист_1", [][]string{
		{"Период: 01.08.2025 - 31.07.2026"},
		{"Прошлый период: 01.08.2025 - 31.10.2025"},
	})
	got, info, err := InferOrderMonth(workbook)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2026-09" {
		t.Fatalf("got %q, want 2026-09", got)
	}
	if info.OrderMonthLabel != "сентябрь 2026" {
		t.Fatalf("unexpected label %q", info.OrderMonthLabel)
	}
}

func TestInferOrderMonthRejectsMismatchedPreviousPeriod(t *testing.T) {
	workbook := newFakeWorkbook("Лист_1", [][]string{
		{"Период: 01.08.2025 - 31.07.2026"},
		{"Прошлый период: 01.01.2025 - 31.03.2025"},
	})
	_, _, err := InferOrderMonth(workbook)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestPlanBlanksRequiresTwoChristinaFiles(t *testing.T) {
	_, err := PlanBlanks("christina", []string{"HOME.xlsx"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestPlanBlanksLabelsChristinaHomeAndProffFromFileNames(t *testing.T) {
	got, err := PlanBlanks("christina", []string{"Актуальный_бланк PROFF.xlsx", "Бланк HOME.xlsx"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d plans", len(got))
	}
	if got[0].ID != "blank-1" || got[0].Label != "PROFF" {
		t.Fatalf("first blank: %+v", got[0])
	}
	if got[1].ID != "blank-2" || got[1].Label != "HOME" {
		t.Fatalf("second blank: %+v", got[1])
	}
}

func TestPlanBlanksRejectsTwoBlanksForSingleBlankBrand(t *testing.T) {
	_, err := PlanBlanks("angiopharm", []string{"a.xlsx", "b.xlsx"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestPlanBlanksFallsBackToUploadOrderWhenFileNamesAreNeutral(t *testing.T) {
	got, err := PlanBlanks("christina", []string{"бланк-1.xlsx", "бланк-2.xlsx"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].Label != "HOME" || got[1].Label != "PROFF" {
		t.Fatalf("expected HOME then PROFF, got %+v", got)
	}
}

func TestPlanBlanksKeepsSingleBlankFileName(t *testing.T) {
	got, err := PlanBlanks("klapp", []string{"Бланк Заказа KLAPP.xlsx"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Label != "Бланк Заказа KLAPP.xlsx" || got[0].ID != "blank-1" {
		t.Fatalf("unexpected plan %+v", got)
	}
}
