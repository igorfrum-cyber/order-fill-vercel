package xlsx

import (
	"regexp"
	"strconv"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

var formulaA1Pattern = regexp.MustCompile(`(\$?)([A-Za-z]+)(\$?)([1-9][0-9]*)`)

func expandSharedFormulas(cells map[cellKey]*cell) {
	type master struct {
		key  cellKey
		text string
	}
	masters := map[string]master{}
	for key, item := range cells {
		if item == nil || item.element == nil {
			continue
		}
		formula := item.element.SelectElement("f")
		if formula == nil || !strings.EqualFold(formula.SelectAttrValue("t", ""), "shared") {
			continue
		}
		text := strings.TrimSpace(elementText(formula))
		if text == "" {
			continue
		}
		masters[formula.SelectAttrValue("si", "")] = master{key: key, text: text}
	}
	for key, item := range cells {
		if item == nil || item.element == nil {
			continue
		}
		formula := item.element.SelectElement("f")
		if formula == nil || !strings.EqualFold(formula.SelectAttrValue("t", ""), "shared") {
			continue
		}
		text := strings.TrimSpace(elementText(formula))
		if text != "" {
			item.formula = text
			continue
		}
		found, ok := masters[formula.SelectAttrValue("si", "")]
		if !ok {
			continue
		}
		item.formula = shiftFormula(found.text, found.key, key)
	}
}

func shiftFormula(formula string, from cellKey, to cellKey) string {
	deltaRow := to.row - from.row
	deltaColumn := to.column - from.column
	if deltaRow == 0 && deltaColumn == 0 {
		return formula
	}
	return formulaA1Pattern.ReplaceAllStringFunc(formula, func(match string) string {
		parts := formulaA1Pattern.FindStringSubmatch(match)
		if len(parts) != 5 {
			return match
		}
		column := spreadsheet.ParseColumnName(parts[2])
		row, err := strconv.Atoi(parts[4])
		if err != nil || column < 1 || row < 1 {
			return match
		}
		if parts[1] == "" {
			column += deltaColumn
		}
		if parts[3] == "" {
			row += deltaRow
		}
		if column < 1 || row < 1 {
			return match
		}
		out := ""
		if parts[1] != "" {
			out += "$"
		}
		out += spreadsheet.ColumnName(column)
		if parts[3] != "" {
			out += "$"
		}
		out += strconv.Itoa(row)
		return out
	})
}

// SetFormula materializes shared followers before replacing a shared master.
func (s *sheet) SetFormula(row, column int, formula string, value float64) {
	for _, item := range s.cells {
		if item.formula == "" {
			continue
		}
		f := item.element.SelectElement("f")
		if f != nil && strings.EqualFold(f.SelectAttrValue("t", ""), "shared") {
			f.SetText(item.formula)
			f.RemoveAttr("t")
			f.RemoveAttr("si")
			f.RemoveAttr("ref")
		}
	}
	target := s.findOrCreateCell(row, column)
	clearCell(target.element)
	newChild(target.element, "f", nil).SetText(formula)
	target.formula = formula
	s.SetFormulaCache(row, column, value)
}

func (s *sheet) SetFormulaCache(row, column int, value float64) {
	target := s.findOrCreateCell(row, column)
	text := strconv.FormatFloat(value, 'f', -1, 64)
	v := target.element.SelectElement("v")
	if v == nil {
		v = newChild(target.element, "v", nil)
	}
	v.SetText(text)
	target.value = text
	s.modified = true
}
