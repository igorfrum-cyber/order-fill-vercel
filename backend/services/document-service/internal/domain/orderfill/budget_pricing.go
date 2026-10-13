package orderfill

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

// BudgetPricing is the agreed base column and discount applied to an export.
// Each finalize reloads the original filled workbook, avoiding compounded prices.
type BudgetPricing struct {
	RowKey      string  `json:"row_key"`
	PriceColumn int     `json:"price_column"`
	BasePrice   float64 `json:"base_price"`
	Discount    float64 `json:"discount"`
	Price       float64 `json:"price"`
}

// BudgetPrice describes a selectable supplier price column in the review report.
type BudgetPrice struct {
	Column int     `json:"column"`
	Label  string  `json:"label"`
	Price  float64 `json:"price"`
}

func BudgetPrices(sheet spreadsheet.Sheet, header, row, quantityColumn int) []BudgetPrice {
	prices := []BudgetPrice{}
	for column := 1; column <= sheet.Bounds().MaxColumn; column++ {
		label := sheet.Value(header, column)
		if !budgetPriceHeader(label) {
			continue
		}
		price, _ := BudgetCellPrice(sheet, row, column)
		prices = append(prices, BudgetPrice{Column: column, Label: label, Price: price})
	}
	return prices
}

func budgetPriceHeader(label string) bool {
	label = strings.ToLower(label)
	return strings.Contains(label, "цена") || strings.Contains(label, "price")
}

var directBudgetDiscount = regexp.MustCompile(`(?i)^([A-Z]+[0-9]+)\*\(1-([A-Z]+[0-9]+)(/100)?\)$`)

// ApplyBudgetPricing mirrors main's invoice-column selection, shared discount
// update and independent-base fallback. CHRISTINA keeps supplier club prices.
func ApplyBudgetPricing(sheet spreadsheet.Sheet, headerRow int, brand string, rows []ReportRow, entries []BudgetPricing) error {
	if len(entries) == 0 {
		return nil
	}
	byKey := map[string]ReportRow{}
	for _, row := range rows {
		byKey[row.Key] = row
	}
	type preparedEntry struct {
		entry     BudgetPricing
		row       ReportRow
		effective int
	}
	prepared := []preparedEntry{}
	for _, entry := range entries {
		row, found := byKey[entry.RowKey]
		if !found {
			continue
		}
		if math.IsNaN(entry.Discount) || math.IsInf(entry.Discount, 0) || entry.Discount < 0 || entry.Discount > 100 {
			return fmt.Errorf("invalid budget discount")
		}
		quantity, _ := normalize.ParseNumber(sheet.Value(row.BlankRow, row.BlankQuantityColumn))
		if quantity > 0 && (!(entry.Price > 0) || entry.PriceColumn < 1) {
			return fmt.Errorf("не для всех заказанных позиций определена цена")
		}
		if entry.Price <= 0 || entry.PriceColumn < 1 {
			continue
		}
		if entry.PriceColumn > sheet.Bounds().MaxColumn || !budgetPriceHeader(sheet.Value(headerRow, entry.PriceColumn)) {
			return fmt.Errorf("не найдена выбранная колонка цены")
		}
		// Trust neither price nor base supplied by the browser: verify the original column.
		base, ok := BudgetCellPrice(sheet, row.BlankRow, entry.PriceColumn)
		if !ok || math.Abs(base-entry.BasePrice) > 0.000001 {
			return fmt.Errorf("цена бланка изменилась: %s", row.BlankName)
		}
		expected := math.Floor((base*(1-entry.Discount/100)+2.220446049250313e-16)*100+0.5) / 100
		if brand != "christina" && math.Abs(expected-entry.Price) > 0.000001 {
			return fmt.Errorf("цена не совпала с расчетом: %s", row.BlankName)
		}
		if brand == "christina" {
			continue
		}
		effective, err := budgetInvoiceColumn(sheet, headerRow, row, entry.PriceColumn)
		if err != nil {
			return err
		}
		prepared = append(prepared, preparedEntry{entry, row, effective})
	}
	if brand == "christina" {
		return nil
	}
	writer, ok := sheet.(spreadsheet.FormulaWriter)
	if !ok {
		return fmt.Errorf("workbook adapter cannot write price formulas")
	}
	formulas := map[[2]int]string{}
	for _, f := range writer.Formulas() {
		formulas[[2]int{f.Row, f.Column}] = f.Text
	}
	changes := map[[2]int]float64{}
	for _, p := range prepared {
		source := spreadsheet.ColumnName(p.entry.PriceColumn) + strconv.Itoa(p.row.BlankRow)
		f := strings.NewReplacer(" ", "", "$", "", "\t", "").Replace(formulas[[2]int{p.row.BlankRow, p.effective}])
		if parts := budgetRound.FindStringSubmatch(f); len(parts) > 0 && parts[2] == "2" {
			f = parts[1]
		}
		direct := directBudgetDiscount.FindStringSubmatch(f)
		if len(direct) == 0 || direct[1] != source {
			continue
		}
		ref := budgetReference.FindStringSubmatch(direct[2])
		r, _ := strconv.Atoi(ref[2])
		c := spreadsheet.ParseColumnName(ref[1])
		if r >= headerRow {
			continue
		}
		value := p.entry.Discount
		if direct[3] == "" {
			value /= 100
		}
		key := [2]int{r, c}
		if prior, found := changes[key]; found && prior != value {
			return fmt.Errorf("разные скидки ссылаются на одну ячейку бланка")
		}
		changes[key] = value
	}
	for ref, value := range changes {
		sheet.SetNumber(ref[0], ref[1], value)
	}
	baseColumn := sheet.Bounds().MaxColumn + 2
	blocks := map[int]int{}
	for _, p := range prepared {
		actual, ok := BudgetCellPrice(sheet, p.row.BlankRow, p.effective)
		if ok && math.Abs(actual-p.entry.Price) <= 0.000001 {
			continue
		}
		offset, found := blocks[p.row.BlankQuantityColumn]
		if !found {
			offset = len(blocks) * 2
			blocks[p.row.BlankQuantityColumn] = offset
		}
		b := baseColumn + offset
		d := b + 1
		r := p.row.BlankRow
		sheet.SetText(headerRow, b, "Цена до нашей скидки")
		sheet.SetText(headerRow, d, "Наша скидка, %")
		sheet.SetNumber(r, b, p.entry.BasePrice)
		sheet.SetNumber(r, d, p.entry.Discount)
		writer.SetFormula(r, p.effective, fmt.Sprintf("ROUND(%s%d*(1-%s%d/100),2)", spreadsheet.ColumnName(b), r, spreadsheet.ColumnName(d), r), p.entry.Price)
		if p.effective == p.entry.PriceColumn {
			sheet.SetText(headerRow, p.effective, "Цена с учетом нашей скидки")
		}
	}
	for _, f := range writer.Formulas() {
		if value, ok := BudgetCellPrice(sheet, f.Row, f.Column); ok {
			writer.SetFormulaCache(f.Row, f.Column, value)
		}
	}
	for _, p := range prepared {
		value, ok := BudgetCellPrice(sheet, p.row.BlankRow, p.effective)
		if !ok || math.Abs(value-p.entry.Price) > 0.000001 {
			return fmt.Errorf("цена в бланке не совпала с расчетом: %s", p.row.BlankName)
		}
	}
	return nil
}

func budgetInvoiceColumn(sheet spreadsheet.Sheet, header int, row ReportRow, selected int) (int, error) {
	prices := map[int]bool{}
	discounted := []int{}
	for c := 1; c <= sheet.Bounds().MaxColumn; c++ {
		label := sheet.Value(header, c)
		if budgetPriceHeader(label) {
			prices[c] = true
			if strings.Contains(strings.ToLower(label), "скид") {
				discounted = append(discounted, c)
			}
		}
	}
	referenced := map[int]bool{}
	if formulated, ok := sheet.(spreadsheet.Formulated); ok {
		qtyRef := spreadsheet.ColumnName(row.BlankQuantityColumn) + strconv.Itoa(row.BlankRow)
		for _, f := range formulated.Formulas() {
			if f.Row != row.BlankRow {
				continue
			}
			refs := budgetReference.FindAllStringSubmatch(strings.ReplaceAll(f.Text, "$", ""), -1)
			qty := false
			for _, ref := range refs {
				if ref[0] == qtyRef {
					qty = true
				}
			}
			if !qty {
				continue
			}
			for _, ref := range refs {
				r, _ := strconv.Atoi(ref[2])
				c := spreadsheet.ParseColumnName(ref[1])
				if r == row.BlankRow && prices[c] {
					referenced[c] = true
				}
			}
		}
	}
	if len(referenced) == 1 {
		for c := range referenced {
			return c, nil
		}
	}
	if len(referenced) > 1 {
		return 0, fmt.Errorf("несколько цен в формуле суммы: %s", row.BlankName)
	}
	if len(discounted) == 1 {
		return discounted[0], nil
	}
	if len(prices) == 1 {
		for c := range prices {
			return c, nil
		}
	}
	if selected == row.BlankQuantityColumn-1 && prices[selected] {
		return selected, nil
	}
	return 0, fmt.Errorf("не удалось определить цену для суммы: %s", row.BlankName)
}

// RefreshBudgetCaches refreshes supported numeric caches after quantities or profile fields change.
func RefreshBudgetCaches(sheet spreadsheet.Sheet) {
	writer, ok := sheet.(spreadsheet.FormulaWriter)
	if !ok {
		return
	}
	for _, formula := range writer.Formulas() {
		if value, ok := BudgetCellPrice(sheet, formula.Row, formula.Column); ok {
			writer.SetFormulaCache(formula.Row, formula.Column, value)
		}
	}
}

// RefreshBudgetData ensures report price options describe the processed company-profile copy.
func RefreshBudgetData(book spreadsheet.Workbook, rule brand.RuleConfig, rows []ReportRow) error {
	detection, err := DetectOrderBlank(book, rule)
	if err != nil {
		return err
	}
	RefreshBudgetCaches(detection.Sheet)
	for i := range rows {
		row := &rows[i]
		if row.BlankQuantityColumn <= 0 || row.BlankRow <= detection.HeaderRow {
			continue
		}
		if rule.BlankLayout == "splitVariants" {
			col := row.BlankQuantityColumn - 1
			row.BudgetPrice = budgetPriceAt(detection.Sheet, row.BlankRow, col)
			row.BudgetPrices = []BudgetPrice{{Column: col, Label: detection.Sheet.Value(detection.HeaderRow, col) + " (по блоку объема)", Price: row.BudgetPrice}}
		} else {
			row.BudgetPrices = BudgetPrices(detection.Sheet, detection.HeaderRow, row.BlankRow, row.BlankQuantityColumn)
			row.BudgetPrice = budgetPriceAt(detection.Sheet, row.BlankRow, budgetPriceColumn(detection))
		}
	}
	return nil
}
