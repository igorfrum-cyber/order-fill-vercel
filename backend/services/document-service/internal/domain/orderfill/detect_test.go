package orderfill

import (
	"errors"
	"strings"
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
	if !strings.Contains(err.Error(), "1С") {
		t.Fatalf("expected a human missing-source message, got %v", err)
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

func TestPlanBlanksAcceptsOneChristinaFile(t *testing.T) {
	got, err := PlanBlanks("christina", []string{"Актуальный_бланк PROFF.xlsx"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "blank-1" || got[0].Label != "Актуальный_бланк PROFF.xlsx" {
		t.Fatalf("unexpected plan %+v", got)
	}
}

func TestPlanBlanksRejectsTwoChristinaFiles(t *testing.T) {
	_, err := PlanBlanks("christina", []string{"HOME.xlsx", "PROFF.xlsx"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestPlanBlanksRejectsTwoBlanksForSingleBlankBrand(t *testing.T) {
	_, err := PlanBlanks("angiopharm", []string{"a.xlsx", "b.xlsx"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestLabelChristinaBlankReadsProfessionalCareSection(t *testing.T) {
	workbook := newFakeWorkbook("Лист1", [][]string{
		{"БЛАНК ЗАКАЗА НА ПРОДУКЦИЮ CHRISTINA"},
		{"Код", "Наименование препарата", "Форма выпуска (мл)", "Кол-во"},
		{"", "MUSE – ЛИНИЯ С ЭКЗОСОМАМИ"},
		{"", "Профессиональный уход"},
		{"CHR884", "Muse Milky Cleanser", "300", ""},
	})
	got := LabelChristinaBlank(workbook, "бланк.xlsx")
	if got != "PROFF" {
		t.Fatalf("got %q, want PROFF", got)
	}
}

func TestLabelChristinaBlankReadsHomeCareSection(t *testing.T) {
	workbook := newFakeWorkbook("Лист1", [][]string{
		{"Код", "Наименование препарата", "Форма выпуска (мл)", "Кол-во"},
		{"", "Домашний уход"},
		{"CHR967", "Muse Regenerating Cream", "50", ""},
	})
	got := LabelChristinaBlank(workbook, "бланк.xlsx")
	if got != "HOME" {
		t.Fatalf("got %q, want HOME", got)
	}
}

func TestLabelChristinaBlankUsesFileNameWhenSectionsAreMissing(t *testing.T) {
	workbook := newFakeWorkbook("Лист1", [][]string{
		{"Код", "Наименование", "Кол-во"},
		{"CHR050", "Post Peel Cream", ""},
	})
	got := LabelChristinaBlank(workbook, "Актуальный_бланк PROFF.xlsx")
	if got != "PROFF" {
		t.Fatalf("got %q, want PROFF", got)
	}
}

func TestLabelChristinaBlankPrefersWorkbookOverFileName(t *testing.T) {
	workbook := newFakeWorkbook("Лист1", [][]string{
		{"Код", "Наименование препарата", "Кол-во"},
		{"", "Профессиональный уход"},
		{"CHR884", "Muse Milky Cleanser", ""},
	})
	got := LabelChristinaBlank(workbook, "Бланк HOME.xlsx")
	if got != "PROFF" {
		t.Fatalf("got %q, want PROFF from the blank itself, got from file name", got)
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
