package preview

import (
	"regexp"
	"strconv"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

var (
	formulaRangePattern = regexp.MustCompile(`(?i)\$?[A-Z]+\$?[1-9][0-9]*:\$?[A-Z]+\$?[1-9][0-9]*`)
	formulaCellPattern  = regexp.MustCompile(`(?i)\$?[A-Z]+\$?[1-9][0-9]*`)
)

const maxFormulaRangeCells = 4000

func captureFormulas(sheet spreadsheet.Sheet, meta *SheetMeta) {
	formulated, ok := sheet.(spreadsheet.Formulated)
	if !ok {
		return
	}
	formulas := formulated.Formulas()
	if len(formulas) == 0 {
		return
	}
	meta.Formulas = make([]SheetFormula, 0, len(formulas))
	values := map[string]string{}
	for _, item := range formulas {
		if item.Row < 1 || item.Column < 1 || strings.TrimSpace(item.Text) == "" {
			continue
		}
		meta.Formulas = append(meta.Formulas, SheetFormula{Row: item.Row, Column: item.Column, Text: item.Text})
		for _, ref := range referencedCells(item.Text) {
			if ref[0] == item.Row && ref[1] == item.Column {
				continue
			}
			value := sheet.Value(ref[0], ref[1])
			if value == "" {
				continue
			}
			values[cellValueKey(ref[0], ref[1])] = value
		}
	}
	if len(values) > 0 {
		meta.FormulaValues = values
	}
}

func referencedCells(formula string) [][2]int {
	text := strings.TrimSpace(strings.TrimPrefix(formula, "="))
	seen := map[[2]int]bool{}
	out := make([][2]int, 0)
	add := func(row int, column int) {
		if row < 1 || column < 1 {
			return
		}
		key := [2]int{row, column}
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, key)
	}
	leftover := formulaRangePattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := strings.Split(match, ":")
		if len(parts) != 2 {
			return " "
		}
		start, okStart := parseA1(parts[0])
		end, okEnd := parseA1(parts[1])
		if !okStart || !okEnd {
			return " "
		}
		rowFrom, rowTo := start[0], end[0]
		if rowTo < rowFrom {
			rowFrom, rowTo = rowTo, rowFrom
		}
		colFrom, colTo := start[1], end[1]
		if colTo < colFrom {
			colFrom, colTo = colTo, colFrom
		}
		count := (rowTo - rowFrom + 1) * (colTo - colFrom + 1)
		if count > maxFormulaRangeCells {
			return " "
		}
		for row := rowFrom; row <= rowTo; row++ {
			for column := colFrom; column <= colTo; column++ {
				add(row, column)
			}
		}
		return " "
	})
	for _, match := range formulaCellPattern.FindAllString(leftover, -1) {
		if ref, ok := parseA1(match); ok {
			add(ref[0], ref[1])
		}
	}
	return out
}

func parseA1(text string) ([2]int, bool) {
	trimmed := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(text), "$", ""))
	split := 0
	for split < len(trimmed) && trimmed[split] >= 'A' && trimmed[split] <= 'Z' {
		split++
	}
	if split == 0 || split == len(trimmed) {
		return [2]int{}, false
	}
	column := spreadsheet.ParseColumnName(trimmed[:split])
	row := 0
	for _, char := range trimmed[split:] {
		if char < '0' || char > '9' {
			return [2]int{}, false
		}
		row = row*10 + int(char-'0')
	}
	if column < 1 || row < 1 {
		return [2]int{}, false
	}
	return [2]int{row, column}, true
}

func cellValueKey(row int, column int) string {
	return strconv.Itoa(row) + ":" + strconv.Itoa(column)
}
