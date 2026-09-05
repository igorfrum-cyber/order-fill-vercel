package north

import (
	"strings"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

var cityLabels = []struct {
	key     string
	label   string
	needles []string
}{
	{"tyumen", "Тюмень", []string{"тюмен", "tyumen"}},
	{"surgut", "Сургут", []string{"сургут", "surgut"}},
	{"nizhnevartovsk", "Вартовск", []string{"вартов", "nizhne", "nvartovsk"}},
	{"urengoy", "Уренгой", []string{"уренгой", "urengoy"}},
}

func CityFromFileName(name string) (key, label string, ok bool) {
	lower := strings.ToLower(strings.ReplaceAll(name, "ё", "е"))
	for _, city := range cityLabels {
		for _, needle := range city.needles {
			if strings.Contains(lower, needle) {
				return city.key, city.label, true
			}
		}
	}
	return "", "", false
}

func Label(key string) string {
	for _, city := range cityLabels {
		if city.key == key {
			return city.label
		}
	}
	return key
}

type Need struct {
	City, Article, Name string
	Qty                 float64
}

type Stock struct {
	Article, Name            string
	Stock, InTransit, Target float64
}

func NeedsFromBlank(workbook spreadsheet.Workbook, brandKey, city string) ([]Need, error) {
	detection, err := orderfill.DetectBlankColumns(workbook, brand.Rule(brandKey))
	if err != nil {
		return nil, err
	}
	qtyCol := detection.Columns[orderfill.ColumnQuantity]
	articleCol := detection.Columns[orderfill.ColumnArticle]
	nameCol := detection.Columns[orderfill.ColumnName]
	bounds := detection.Sheet.Bounds()
	out := make([]Need, 0)
	for row := detection.HeaderRow + 1; row <= bounds.MaxRow; row++ {
		article := strings.TrimSpace(detection.Sheet.Value(row, articleCol))
		if article == "" {
			continue
		}
		qty, ok := normalize.ParseNumber(detection.Sheet.Value(row, qtyCol))
		if !ok || qty <= 0 {
			continue
		}
		name := ""
		if nameCol > 0 {
			name = strings.TrimSpace(detection.Sheet.Value(row, nameCol))
		}
		out = append(out, Need{City: city, Article: article, Name: name, Qty: qty})
	}
	return out, nil
}

func StockFromSource(workbook spreadsheet.Workbook) ([]Stock, error) {
	detection, err := orderfill.DetectSourceColumns(workbook)
	if err != nil {
		return nil, err
	}
	articleCol := detection.Columns[orderfill.ColumnArticle]
	nameCol := detection.Columns[orderfill.ColumnName]
	stockCol := detection.Columns[orderfill.ColumnStock]
	transitCol := detection.Columns[orderfill.ColumnInTransit]
	bounds := detection.Sheet.Bounds()
	out := make([]Stock, 0)
	for row := detection.HeaderRow + 1; row <= bounds.MaxRow; row++ {
		article := strings.TrimSpace(detection.Sheet.Value(row, articleCol))
		if article == "" {
			continue
		}
		item := Stock{Article: article}
		if nameCol > 0 {
			item.Name = strings.TrimSpace(detection.Sheet.Value(row, nameCol))
		}
		if stockCol > 0 {
			item.Stock, _ = normalize.ParseNumber(detection.Sheet.Value(row, stockCol))
		}
		if transitCol > 0 {
			item.InTransit, _ = normalize.ParseNumber(detection.Sheet.Value(row, transitCol))
		}
		out = append(out, item)
	}
	return out, nil
}
