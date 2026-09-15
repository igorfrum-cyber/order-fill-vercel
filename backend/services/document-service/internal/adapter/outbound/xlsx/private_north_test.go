package xlsx_test

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"order-fill/backend/services/document-service/internal/adapter/outbound/xlsx"
	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/north"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

func TestPrivateNorthWorkbooks(t *testing.T) {
	root := os.Getenv("ORDER_FILL_PRIVATE_TESTDATA")
	if root == "" {
		root = filepath.Clean(filepath.Join("..", "..", "..", "..", "..", "..", "testdata", "private"))
	}
	if _, err := os.Stat(root); err != nil {
		t.Skip("private testdata is not available")
	}
	codec := xlsx.NewCodec()
	checkDir := func(dir string, source bool) {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".xlsx") {
				continue
			}
			name := entry.Name()
			t.Run(dir+"/"+name, func(t *testing.T) {
				content, err := os.ReadFile(filepath.Join(root, dir, name))
				if err != nil {
					t.Fatal(err)
				}
				workbook, err := codec.Load(content)
				if err != nil {
					t.Fatal(err)
				}
				brandKey := privateBrand(name)
				if source {
					if _, _, ok := north.CityFromWorkbook(workbook, name); !ok {
						t.Fatal("city was not detected")
					}
					rows, err := north.StockFromSource(workbook, brand.Rule(brandKey))
					if err != nil || len(rows) == 0 {
						t.Fatalf("stock rows=%d err=%v", len(rows), err)
					}
					budgetRows := 0
					for _, row := range rows {
						if row.Category != "" && row.MonthlyDemand > 0 {
							budgetRows++
						}
					}
					if budgetRows == 0 {
						for _, sheet := range workbook.Sheets() {
							for row := 1; row <= min(15, sheet.Bounds().MaxRow); row++ {
								values := make([]string, 0, sheet.Bounds().MaxColumn)
								for column := 1; column <= sheet.Bounds().MaxColumn; column++ {
									if value := strings.TrimSpace(sheet.Value(row, column)); value != "" {
										values = append(values, value)
									}
								}
								if len(values) > 0 {
									t.Logf("sheet=%q row=%d values=%q", sheet.Name(), row, values)
								}
							}
						}
						t.Fatal("no ABC rows with monthly demand were extracted for budget planning")
					}
					return
				}
				variant := ""
				if brandKey == "christina" {
					variant = "proff"
				}
				rows, err := north.NeedsFromBlank(workbook, brand.Rule(brandKey), "surgut", variant)
				if err != nil || len(rows) == 0 {
					if err != nil {
						for _, sheet := range workbook.Sheets() {
							if sheet.Bounds().MaxRow < 267 {
								continue
							}
							values := make([]string, 0, sheet.Bounds().MaxColumn)
							for column := 1; column <= sheet.Bounds().MaxColumn; column++ {
								values = append(values, sheet.Value(267, column))
							}
							t.Logf("sheet=%q row267=%q", sheet.Name(), values)
						}
					}
					t.Fatalf("blank rows=%d err=%v", len(rows), err)
				}
			})
		}
	}
	checkDir("Бланки", false)
	checkDir("таблицы продаж", true)
}

func TestPrivateCompanyProfileTargetsRealBlankCells(t *testing.T) {
	root := os.Getenv("ORDER_FILL_PRIVATE_TESTDATA")
	if root == "" {
		root = filepath.Clean(filepath.Join("..", "..", "..", "..", "..", "..", "testdata", "private"))
	}
	cases := []struct {
		name, brand, sheet string
		want               map[[2]int]string
	}{
		{"2026 08 25 Бланк заказа ANGIOPHARM.xlsx", "angiopharm", "Бланк", map[[2]int]string{{1, 3}: "ООО Тест", {2, 3}: "14.09.2026", {17, 3}: "32.5"}},
		{"_Бланк заказа Skin Synergy от 26.08.2026.xlsx", "skin_synergy", "Бланк заказа", map[[2]int]string{{5, 7}: "Дилер Тест", {6, 7}: "0.325"}},
		{"Бланк Заказа KLAPP август 2026 (1).xlsx", "klapp", "БЛАНК ЗАКАЗА", map[[2]int]string{{5, 4}: "Заказ от (просьба указать юр. лицо): ООО Тест", {2, 10}: "0.325"}},
	}
	codec := xlsx.NewCodec()
	for _, tc := range cases {
		t.Run(tc.brand, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(root, "Бланки", tc.name))
			if err != nil {
				t.Skip("private blank is not available")
			}
			workbook, err := codec.Load(content)
			if err != nil {
				t.Fatal(err)
			}
			orderfill.ApplyCompanyOrderProfile(workbook, tc.brand, orderfill.CompanyOrderProfile{
				LegalName: "ООО Тест", BrandTerms: []orderfill.CompanyBrandTerms{{
					Brand: tc.brand, DealerName: "Дилер Тест", DiscountBasisPoints: 3_250, DiscountSet: true,
				}},
			}, time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC))
			sheet, ok := workbook.Sheet(tc.sheet)
			if !ok {
				t.Fatalf("sheet %q not found", tc.sheet)
			}
			for cell, want := range tc.want {
				if got := sheet.Value(cell[0], cell[1]); got != want {
					t.Fatalf("cell %v = %q, want %q", cell, got, want)
				}
			}
			saved, err := workbook.Save()
			if err != nil {
				t.Fatal(err)
			}
			reloaded, err := codec.Load(saved)
			if err != nil {
				t.Fatal(err)
			}
			sheet, _ = reloaded.Sheet(tc.sheet)
			for cell, want := range tc.want {
				if got := sheet.Value(cell[0], cell[1]); got != want {
					t.Fatalf("saved cell %v = %q, want %q", cell, got, want)
				}
			}
		})
	}
}

func TestPrivateNorthBudgetPairs(t *testing.T) {
	root := os.Getenv("ORDER_FILL_PRIVATE_TESTDATA")
	if root == "" {
		root = filepath.Clean(filepath.Join("..", "..", "..", "..", "..", "..", "testdata", "private"))
	}
	if _, err := os.Stat(root); err != nil {
		t.Skip("private testdata is not available")
	}
	pairs := []struct {
		name, brand, blank, source, variant string
	}{
		{"Skin Synergy", "skin_synergy", "_Бланк заказа Skin Synergy от 26.08.2026.xlsx", "Скин Синерджи Тюмень .xlsx", ""},
		{"ANGIOPHARM", "angiopharm", "2026 08 25 Бланк заказа ANGIOPHARM.xlsx", "Ангио Тюмень .xlsx", ""},
		{"KLAPP", "klapp", "Бланк Заказа KLAPP август 2026 (1).xlsx", "Клапп Тюмень .xlsx", ""},
		{"CHRISTINA PROFF", "christina", "Актуальный_бланк PROFF.xlsx", "Кристина Тюмень .xlsx", "proff"},
	}
	codec := xlsx.NewCodec()
	for _, pair := range pairs {
		t.Run(pair.name, func(t *testing.T) {
			load := func(dir, name string) spreadsheet.Workbook {
				content, err := os.ReadFile(filepath.Join(root, dir, name))
				if err != nil {
					t.Fatal(err)
				}
				workbook, err := codec.Load(content)
				if err != nil {
					t.Fatal(err)
				}
				return workbook
			}
			blank := load("Бланки", pair.blank)
			source := load("таблицы продаж", pair.source)
			rule := brand.Rule(pair.brand)
			needs, err := north.NeedsFromBlank(blank, rule, "surgut", pair.variant)
			if err != nil {
				t.Fatal(err)
			}
			stock, err := north.StockFromSource(source, rule)
			if err != nil {
				t.Fatal(err)
			}
			planned := make([]north.Planned, 0, len(needs))
			for _, need := range needs {
				planned = append(planned, north.Planned{Article: need.Article, Name: need.Name, Variant: need.Variant})
			}
			report := north.BuildReport(pair.brand, needs, stock, planned, nil)
			eligible := 0
			for _, row := range report.PlanRows {
				if row.HasBudgetData {
					eligible++
				}
			}
			if eligible == 0 {
				t.Fatalf("no budget rows for real %s pair", pair.name)
			}
		})
	}
}

func TestPrivateNorthFinalWorkbookRoundTrip(t *testing.T) {
	root := os.Getenv("ORDER_FILL_PRIVATE_TESTDATA")
	if root == "" {
		root = filepath.Clean(filepath.Join("..", "..", "..", "..", "..", "..", "testdata", "private"))
	}
	path := filepath.Join(root, "Бланки", "_Бланк заказа Skin Synergy от 26.08.2026.xlsx")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Skip("private Skin Synergy blank is not available")
	}
	codec := xlsx.NewCodec()
	workbook, err := codec.Load(content)
	if err != nil {
		t.Fatal(err)
	}
	rule := brand.Rule("skin_synergy")
	before, err := north.NeedsFromBlank(workbook, rule, "surgut")
	if err != nil {
		t.Fatal(err)
	}
	var selected north.Need
	for _, row := range before {
		if row.Price > 0 {
			selected = row
			break
		}
	}
	if selected.Article == "" {
		t.Fatal("real blank has no row with a detected price")
	}
	if err := north.WriteSupplierQuantities(workbook, rule, "", []north.SupplierLine{{
		Key: selected.Article, Article: selected.ArticleRaw, Name: selected.Name, Unit: "шт", Quantity: 7,
	}}); err != nil {
		t.Fatal(err)
	}
	if err := north.ApplySupplierDiscount(workbook, rule, 10); err != nil {
		t.Fatal(err)
	}
	result, err := workbook.Save()
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := codec.Load(result)
	if err != nil {
		t.Fatal(err)
	}
	after, err := north.NeedsFromBlank(reloaded, rule, "surgut")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range after {
		if row.Article != selected.Article {
			continue
		}
		if row.Qty != 7 || math.Abs(row.Price-selected.Price*0.9) > 0.011 {
			t.Fatalf("round trip quantity=%v price=%v, original price=%v", row.Qty, row.Price, selected.Price)
		}
		return
	}
	t.Fatal("updated row disappeared after save and reload")
}

func privateBrand(name string) string {
	lower := strings.ToLower(strings.ReplaceAll(name, "ё", "е"))
	switch {
	case strings.Contains(lower, "skin") || strings.Contains(lower, "скин"):
		return "skin_synergy"
	case strings.Contains(lower, "klapp") || strings.Contains(lower, "клапп"):
		return "klapp"
	case strings.Contains(lower, "proff") || strings.Contains(lower, "кристин"):
		return "christina"
	default:
		return "angiopharm"
	}
}
