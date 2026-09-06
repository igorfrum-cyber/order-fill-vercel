package orderfill

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"order-fill/backend/services/document-service/internal/domain/brand"
)

type exactMatcher struct{}

func (exactMatcher) Match(_ context.Context, blank, source []MatchItem, opts MatchOptions) ([]MatchResult, error) {
	index := map[string][]MatchItem{}
	rule := brand.RuleConfig{ArticlePrefixAliases: opts.PrefixAliases}
	for _, item := range source {
		for _, key := range articleKeys(item.Article, rule) {
			index[key] = append(index[key], item)
		}
	}
	out := make([]MatchResult, 0, len(blank))
	for _, item := range blank {
		seen := map[string]bool{}
		found := make([]MatchItem, 0)
		for _, key := range articleKeys(item.Article, rule) {
			for _, cand := range index[key] {
				if seen[cand.ID] {
					continue
				}
				seen[cand.ID] = true
				found = append(found, cand)
			}
		}
		if len(found) == 0 {
			out = append(out, MatchResult{BlankID: item.ID, Category: CategoryNotInSource})
			continue
		}
		ids := make([]string, len(found))
		for i, cand := range found {
			ids[i] = cand.ID
		}
		result := MatchResult{BlankID: item.ID, SourceID: found[0].ID, Category: CategoryToOrder, Score: 1, CandidateIDs: ids}
		if len(found) > 1 {
			result.Reasons.Duplicates = "chosen_best"
			if opts.Mode == "smart" {
				result.Category = CategoryNeedsDecision
				result.Reasons.Duplicates = "needs_choice"
			}
		}
		out = append(out, result)
	}
	return out, nil
}

type pickNameMatcher struct{ contains string }

func (m pickNameMatcher) Match(_ context.Context, blank, source []MatchItem, _ MatchOptions) ([]MatchResult, error) {
	chosen := source[0]
	ids := make([]string, len(source))
	for i, item := range source {
		ids[i] = item.ID
		if strings.Contains(item.Name, m.contains) {
			chosen = item
		}
	}
	return []MatchResult{{
		BlankID: blank[0].ID, SourceID: chosen.ID, Category: CategoryToOrder, Score: 1, CandidateIDs: ids,
	}}, nil
}

func sourceGrid() [][]string {
	return [][]string{
		{"Период: 01.08.2025 - 31.07.2026"},
		{"Прошлый период: 01.08.2025 - 31.10.2025"},
		{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
		{"A100", "Крем для лица 50 мл", "10", "0", "0", "", ""},
		{"A200", "Гель для умывания 100 мл", "1.2", "4", "0", "", ""},
		{"A300", "Тоник 200 мл", "11", "1", "0", "", ""},
		{"A400", "Сыворотка 30 мл", "5", "0", "0", "", ""},
	}
}

func blankGrid() [][]string {
	return [][]string{
		{"Артикул", "Наименование", "Объем", "Кол-во", "Шт. в коробке"},
		{"A100", "Крем для лица", "50 мл", "", "3"},
		{"A200", "Гель для умывания", "100 мл", "", "3"},
		{"A300", "Тоник", "200 мл", "", "3"},
		{"A999", "Неизвестный товар", "10 мл", "", "3"},
	}
}

func testFill(cmd FillCommand) (Result, error) {
	if cmd.Matcher == nil {
		cmd.Matcher = exactMatcher{}
	}
	return Fill(cmd)
}

func fillFixture(t *testing.T) Result {
	t.Helper()
	result, err := testFill(FillCommand{
		Source:     newFakeWorkbook("Заказ", sourceGrid()),
		Blank:      newFakeWorkbook("Бланк", blankGrid()),
		OrderMonth: "2026-09",
		Brand:      "angiopharm",
		BlankID:    "blank-1",
		BlankLabel: "Бланк",
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}
	return result
}

func rowByKey(t *testing.T, result Result, key string) ReportRow {
	t.Helper()
	for _, row := range result.Rows {
		if row.Key == key {
			return row
		}
	}
	t.Fatalf("report row %q was not found", key)
	return ReportRow{}
}

func TestFillWritesRecommendedQuantityIntoBlank(t *testing.T) {
	result := fillFixture(t)
	sheet, _ := result.Blank.Sheet("Бланк")

	if got := sheet.Value(2, 4); got != "10" {
		t.Fatalf("expected quantity 10 in the blank, got %q", got)
	}
	row := rowByKey(t, result, "blank-1:2")
	if row.Status != StatusMatched {
		t.Fatalf("expected status %q, got %q", StatusMatched, row.Status)
	}
	if row.Inserted == nil || *row.Inserted != 10 {
		t.Fatalf("expected inserted 10, got %v", row.Inserted)
	}
}

func TestFillRoundsUpToBoxWhenCloseEnough(t *testing.T) {
	result := fillFixture(t)
	sheet, _ := result.Blank.Sheet("Бланк")

	if got := sheet.Value(4, 4); got != "12" {
		t.Fatalf("expected quantity rounded up to 12, got %q", got)
	}
	row := rowByKey(t, result, "blank-1:4")
	if !row.BoxAdjusted || row.AutoComment != "до коробки" {
		t.Fatalf("expected box adjustment with comment, got adjusted=%v comment=%q", row.BoxAdjusted, row.AutoComment)
	}
}

func TestFillLeavesNonPositiveRecommendationEmpty(t *testing.T) {
	result := fillFixture(t)
	sheet, _ := result.Blank.Sheet("Бланк")

	if got := sheet.Value(3, 4); got != "" {
		t.Fatalf("expected an empty quantity cell, got %q", got)
	}
	row := rowByKey(t, result, "blank-1:3")
	if row.Status != StatusLeftBlank {
		t.Fatalf("expected status %q, got %q", StatusLeftBlank, row.Status)
	}
	if row.Inserted != nil {
		t.Fatalf("expected no inserted quantity, got %v", *row.Inserted)
	}
}

func TestFillMarksBlankRowsMissingFromSource(t *testing.T) {
	result := fillFixture(t)
	row := rowByKey(t, result, "blank-1:5")

	if row.Status != StatusNotInSource {
		t.Fatalf("expected status %q, got %q", StatusNotInSource, row.Status)
	}
	if row.Editable {
		t.Fatal("a row without a source position must not be editable")
	}
	if result.Summary.Unmatched != 1 {
		t.Fatalf("expected one unmatched row, got %d", result.Summary.Unmatched)
	}
}

func TestFillReportsSourceItemsMissingFromBlank(t *testing.T) {
	result := fillFixture(t)
	found := false
	for _, row := range result.Rows {
		if row.Status == StatusNotInBlank && row.SourceArticle == "A400" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected the source position missing from the blank to be reported")
	}
}

func TestFillSummaryCountsPositions(t *testing.T) {
	result := fillFixture(t)
	if result.Summary.Filled != 2 {
		t.Fatalf("expected 2 filled rows, got %d", result.Summary.Filled)
	}
	if result.Summary.LeftBlank != 1 {
		t.Fatalf("expected 1 empty row, got %d", result.Summary.LeftBlank)
	}
	if result.Summary.Brand != "ANGIOPHARM" {
		t.Fatalf("unexpected brand %q", result.Summary.Brand)
	}
	if result.Summary.OrderMonthLabel != "сентябрь 2026" {
		t.Fatalf("unexpected order month label %q", result.Summary.OrderMonthLabel)
	}
	if result.Summary.NotInBlank != 1 {
		t.Fatalf("expected 1 source item missing from the blank, got %d", result.Summary.NotInBlank)
	}
}

func TestFillCapsMissingFromBlankRowsButKeepsTheTrueCount(t *testing.T) {
	sourceRows := [][]string{
		{"Период: 01.08.2025 - 31.07.2026"},
		{"Прошлый период: 01.08.2025 - 31.10.2025"},
		{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
		{"A100", "Крем для лица 50 мл", "10", "0", "0", "", ""},
	}
	for i := 0; i < MaxNotInBlankReportRows+3; i++ {
		sourceRows = append(sourceRows, []string{
			fmt.Sprintf("X%04d", i),
			fmt.Sprintf("Лишний товар %d", i),
			"8",
			"0",
			"0",
			"",
			"",
		})
	}
	result, err := testFill(FillCommand{
		Source:     newFakeWorkbook("Заказ", sourceRows),
		Blank:      newFakeWorkbook("Бланк", blankGrid()),
		OrderMonth: "2026-09",
		Brand:      "angiopharm",
		BlankID:    "blank-1",
		BlankLabel: "Бланк",
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}

	missing := 0
	for _, row := range result.Rows {
		if row.Status == StatusNotInBlank {
			missing++
		}
	}
	if missing != MaxNotInBlankReportRows {
		t.Fatalf("reported missing rows = %d, want cap %d", missing, MaxNotInBlankReportRows)
	}
	if result.Summary.NotInBlank != MaxNotInBlankReportRows+3 {
		t.Fatalf("summary missing count = %d, want %d", result.Summary.NotInBlank, MaxNotInBlankReportRows+3)
	}
}

func TestFillDoesNotRepeatSourceDuplicatesAlreadyFlaggedOnBlankRows(t *testing.T) {
	source := newFakeWorkbook("Заказ", [][]string{
		{"Период: 01.08.2025 - 31.07.2026"},
		{"Прошлый период: 01.08.2025 - 31.10.2025"},
		{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
		{"6014", "АН Косметичка непромокаемая", "0", "45", "0", "", ""},
		{"6014", "АН Косметичка матовая", "0", "10", "0", "", ""},
		{"ZZ99", "Только в таблице дважды", "4", "0", "0", "", ""},
		{"ZZ99", "Только в таблице дважды другой", "1", "0", "0", "", ""},
	})
	blank := newFakeWorkbook("Бланк", [][]string{
		{"Артикул", "Наименование", "Объем", "Кол-во", "Шт. в коробке"},
		{"6014", "Косметичка непромокаемая", "1", "", "3"},
	})
	result, err := testFill(FillCommand{
		Source:     source,
		Blank:      blank,
		OrderMonth: "2026-09",
		Brand:      "angiopharm",
		BlankID:    "blank-1",
		BlankLabel: "Бланк",
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}

	blankDuplicates := 0
	sourceOnlyDuplicates := 0
	for _, row := range result.Rows {
		if row.Status == StatusSourceDuplicate {
			sourceOnlyDuplicates++
			if row.BlankArticle == "6014" {
				t.Fatal("article 6014 is already flagged on the blank row and must not be repeated")
			}
		}
		if row.Duplicate && row.Status != StatusSourceDuplicate {
			blankDuplicates++
		}
	}
	if result.Summary.Duplicates != 1 {
		t.Fatalf("summary duplicates = %d, want 1 blank row", result.Summary.Duplicates)
	}
	if blankDuplicates != 1 {
		t.Fatalf("blank duplicate rows = %d, want 1", blankDuplicates)
	}
	if sourceOnlyDuplicates != 1 {
		t.Fatalf("source-only duplicate groups = %d, want 1 for ZZ99", sourceOnlyDuplicates)
	}
}

func TestFillRejectsExportBuiltForAnotherMonth(t *testing.T) {
	_, err := testFill(FillCommand{
		Source:     newFakeWorkbook("Заказ", sourceGrid()),
		Blank:      newFakeWorkbook("Бланк", blankGrid()),
		OrderMonth: "2026-10",
		Brand:      "angiopharm",
		BlankID:    "blank-1",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected an invalid input error, got %v", err)
	}
}

func TestFillRejectsUnsupportedBlankLayout(t *testing.T) {
	t.Parallel()
	for _, brandKey := range []string{"novacutan", "sothys"} {
		t.Run(brandKey, func(t *testing.T) {
			t.Parallel()
			_, err := testFill(FillCommand{
				Source:     newFakeWorkbook("Заказ", sourceGrid()),
				Blank:      newFakeWorkbook("Бланк", blankGrid()),
				OrderMonth: "2026-09",
				Brand:      brandKey,
				BlankID:    "blank-1",
			})
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected an invalid input error, got %v", err)
			}
		})
	}
}

func TestFillMatchesLevissimePrefixAlias(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		sourceArticle string
		blankArticle  string
	}{
		{name: "blank digits source MT", sourceArticle: "MT123", blankArticle: "123"},
		{name: "blank MT source digits", sourceArticle: "123", blankArticle: "MT123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := testFill(FillCommand{
				Source: newFakeWorkbook("Заказ", [][]string{
					{"Период: 01.08.2025 - 31.07.2026"},
					{"Прошлый период: 01.08.2025 - 31.10.2025"},
					{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
					{tt.sourceArticle, "Cream 50 ml", "10", "0", "0", "", ""},
				}),
				Blank: newFakeWorkbook("Бланк", [][]string{
					{"Артикул", "Наименование", "Объем", "Заказ", "Кол-во в уп."},
					{tt.blankArticle, "Cream 50 ml", "50 мл", "", "1"},
				}),
				OrderMonth: "2026-09",
				Brand:      "levissime",
				BlankID:    "blank-1",
				BlankLabel: "Бланк",
			})
			if err != nil {
				t.Fatalf("fill failed: %v", err)
			}
			sheet, _ := result.Blank.Sheet("Бланк")
			if got := sheet.Value(2, 4); got != "10" {
				t.Fatalf("quantity = %q, want 10", got)
			}
			row := rowByKey(t, result, "blank-1:2")
			if row.Status != StatusMatched {
				t.Fatalf("status = %q, want %q", row.Status, StatusMatched)
			}
		})
	}
}

func TestFillStandardPicksBestVolumeAmongDuplicates(t *testing.T) {
	t.Parallel()
	result, err := testFill(FillCommand{
		Source: newFakeWorkbook("Заказ", [][]string{
			{"Период: 01.08.2025 - 31.07.2026"},
			{"Прошлый период: 01.08.2025 - 31.10.2025"},
			{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
			{"A1", "Cream 30 ml", "8", "0", "0", "", ""},
			{"A1", "Cream 50 ml", "10", "0", "0", "", ""},
		}),
		Blank: newFakeWorkbook("Бланк", [][]string{
			{"Артикул", "Наименование", "Объем", "Кол-во", "Шт. в коробке"},
			{"A1", "Cream 50 ml", "50 мл", "", "1"},
		}),
		OrderMonth: "2026-09",
		Brand:      "angiopharm",
		BlankID:    "blank-1",
		BlankLabel: "Бланк",
		Matcher:    pickNameMatcher{contains: "50 ml"},
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}
	sheet, _ := result.Blank.Sheet("Бланк")
	if got := sheet.Value(2, 4); got != "10" {
		t.Fatalf("quantity = %q, want 10 from the 50 ml source row", got)
	}
	row := rowByKey(t, result, "blank-1:2")
	if row.Status == StatusSourceDuplicate {
		t.Fatal("standard mode must auto-fill a clear volume winner")
	}
	if row.Inserted == nil || *row.Inserted != 10 {
		t.Fatalf("inserted = %v, want 10", row.Inserted)
	}
}

func TestFillSmartLeavesAmbiguousDuplicateEmpty(t *testing.T) {
	t.Parallel()
	result, err := testFill(FillCommand{
		Source: newFakeWorkbook("Заказ", [][]string{
			{"Период: 01.08.2025 - 31.07.2026"},
			{"Прошлый период: 01.08.2025 - 31.10.2025"},
			{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
			{"A1", "Cream", "10", "0", "0", "", ""},
			{"A1", "Cream", "8", "0", "0", "", ""},
		}),
		Blank: newFakeWorkbook("Бланк", [][]string{
			{"Артикул", "Наименование", "Объем", "Кол-во", "Шт. в коробке"},
			{"A1", "Cream", "50 мл", "", "1"},
		}),
		OrderMonth:   "2026-09",
		Brand:        "angiopharm",
		BlankID:      "blank-1",
		BlankLabel:   "Бланк",
		MatchingMode: "smart",
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}
	sheet, _ := result.Blank.Sheet("Бланк")
	if got := sheet.Value(2, 4); got != "" {
		t.Fatalf("smart mode must leave quantity empty, got %q", got)
	}
	row := rowByKey(t, result, "blank-1:2")
	if row.Status != StatusSourceDuplicate {
		t.Fatalf("status = %q, want %q", row.Status, StatusSourceDuplicate)
	}
	if row.Inserted != nil {
		t.Fatalf("inserted = %v, want nil", row.Inserted)
	}
}

func TestFillSmartFillsClearVolumeWinner(t *testing.T) {
	t.Parallel()
	result, err := testFill(FillCommand{
		Source: newFakeWorkbook("Заказ", [][]string{
			{"Период: 01.08.2025 - 31.07.2026"},
			{"Прошлый период: 01.08.2025 - 31.10.2025"},
			{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
			{"A1", "Cream 30 ml", "8", "0", "0", "", ""},
			{"A1", "Cream 50 ml", "10", "0", "0", "", ""},
		}),
		Blank: newFakeWorkbook("Бланк", [][]string{
			{"Артикул", "Наименование", "Объем", "Кол-во", "Шт. в коробке"},
			{"A1", "Cream 50 ml", "50 мл", "", "1"},
		}),
		OrderMonth:   "2026-09",
		Brand:        "angiopharm",
		BlankID:      "blank-1",
		BlankLabel:   "Бланк",
		MatchingMode: "smart",
		Matcher:      pickNameMatcher{contains: "50 ml"},
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}
	sheet, _ := result.Blank.Sheet("Бланк")
	if got := sheet.Value(2, 4); got != "10" {
		t.Fatalf("quantity = %q, want 10", got)
	}
	row := rowByKey(t, result, "blank-1:2")
	if row.Status == StatusSourceDuplicate {
		t.Fatal("smart mode must still fill when the volume winner is clear")
	}
}

func TestFillUsesOrderedFactOverRecommended(t *testing.T) {
	t.Parallel()
	result, err := testFill(FillCommand{
		Source: newFakeWorkbook("Заказ", [][]string{
			{"Период: 01.08.2025 - 31.07.2026"},
			{"Прошлый период: 01.08.2025 - 31.10.2025"},
			{"Артикул", "Товар", "Рекомендуемый заказ", "Остаток", "В пути", "Заказано по факту", "Комментарий"},
			{"A100", "Крем для лица 50 мл", "10", "0", "0", "6", ""},
		}),
		Blank: newFakeWorkbook("Бланк", [][]string{
			{"Артикул", "Наименование", "Объем", "Кол-во", "Шт. в коробке"},
			{"A100", "Крем для лица", "50 мл", "", "3"},
		}),
		OrderMonth: "2026-09",
		Brand:      "angiopharm",
		BlankID:    "blank-1",
		BlankLabel: "Бланк",
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}
	sheet, _ := result.Blank.Sheet("Бланк")
	if got := sheet.Value(2, 4); got != "6" {
		t.Fatalf("quantity = %q, want 6 from ordered fact", got)
	}
}

func TestFillReportsProgressThroughThePipeline(t *testing.T) {
	var reports []float64
	var messages []string
	_, err := testFill(FillCommand{
		Source:     newFakeWorkbook("Заказ", sourceGrid()),
		Blank:      newFakeWorkbook("Бланк", blankGrid()),
		OrderMonth: "2026-09",
		Brand:      "angiopharm",
		BlankID:    "blank-1",
		BlankLabel: "Бланк",
		OnProgress: func(fraction float64, message string) {
			reports = append(reports, fraction)
			messages = append(messages, message)
		},
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}
	if len(reports) == 0 {
		t.Fatal("fill must report progress while it works")
	}
	if reports[len(reports)-1] < 0.99 {
		t.Fatalf("final progress %v, want at least 0.99", reports[len(reports)-1])
	}
	for index := 1; index < len(reports); index++ {
		if reports[index]+1e-9 < reports[index-1] {
			t.Fatalf("progress went backwards: %v -> %v", reports[index-1], reports[index])
		}
	}
	for _, message := range messages {
		if message == "" {
			t.Fatal("every progress update needs a user-facing message")
		}
	}
}

func TestFillUsesAdjuster(t *testing.T) {
	t.Parallel()
	qty := 42.0
	result, err := testFill(FillCommand{
		Source:     newFakeWorkbook("Заказ", sourceGrid()),
		Blank:      newFakeWorkbook("Бланк", blankGrid()),
		OrderMonth: "2026-09",
		Brand:      "angiopharm",
		BlankID:    "blank-1",
		BlankLabel: "Бланк",
		Adjuster:   stubAdjuster{qty: qty},
	})
	if err != nil {
		t.Fatalf("fill failed: %v", err)
	}
	row := rowByKey(t, result, "blank-1:2")
	if row.Inserted == nil || *row.Inserted != qty {
		t.Fatalf("calculation-service must own qty, got %v", row.Inserted)
	}
}

type stubAdjuster struct{ qty float64 }

func (s stubAdjuster) AdjustQuantity(context.Context, float64, float64, bool, string, brand.RuleConfig) (brand.AdjustedQuantity, error) {
	qty := s.qty
	return brand.AdjustedQuantity{Inserted: &qty, Rounded: int(qty)}, nil
}

func TestFillRequiresMatcher(t *testing.T) {
	t.Parallel()
	_, err := Fill(FillCommand{
		Source:     newFakeWorkbook("Заказ", sourceGrid()),
		Blank:      newFakeWorkbook("Бланк", blankGrid()),
		OrderMonth: "2026-09",
		Brand:      "angiopharm",
	})
	if err == nil {
		t.Fatal("fill without matching-service must fail")
	}
}
