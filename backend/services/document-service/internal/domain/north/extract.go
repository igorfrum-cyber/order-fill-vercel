package north

import (
	"cmp"
	"fmt"
	"math"
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
	return cityFromText(name)
}

func cityFromText(value string) (key, label string, ok bool) {
	lower := normalize.NormalizeHeader(value)
	for _, city := range cityLabels {
		for _, needle := range city.needles {
			if strings.Contains(lower, needle) {
				return city.key, city.label, true
			}
		}
	}
	return "", "", false
}

// CityFromWorkbook mirrors the prototype's lookup order: sheet names and the
// first cells are authoritative, while the file name remains a fallback.
func CityFromWorkbook(workbook spreadsheet.Workbook, fileName string) (key, label string, ok bool) {
	for _, sheet := range workbook.Sheets() {
		if key, label, ok := cityFromText(sheet.Name()); ok {
			return key, label, true
		}
		bounds := sheet.Bounds()
		for row := 1; row <= min(bounds.MaxRow, 50); row++ {
			for column := 1; column <= bounds.MaxColumn; column++ {
				if key, label, ok := cityFromText(sheet.Value(row, column)); ok {
					return key, label, true
				}
			}
		}
	}
	return CityFromFileName(fileName)
}

func VariantFromRole(role string) string {
	switch role {
	case "blank-home":
		return "home"
	case "blank-proff":
		return "proff"
	default:
		return ""
	}
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
	Row, QuantityColumn, PriceColumn                      int
	Prices                                                []orderfill.BudgetPrice
	City, Article, BaseArticle, ArticleRaw, Name, Variant string
	Qty, Price                                            float64
	Minimum, SupplierUnitSize, BoxSize                    float64
	HasBoxSize                                            bool
	// Line is the CHRISTINA PROFF line this blank row belongs to (nil otherwise),
	// so the budget planner can apply the set discount on the North flow too.
	Line *orderfill.ChristinaLine
}

type Stock struct {
	SourceRow                                                    int
	Article, Name, Category                                      string
	Stock, InTransit, Target, MonthlyDemand, Recommended, Actual float64
	HasActual                                                    bool
	WarehouseStock, WarehouseTransit                             float64
	HasWarehouseStock                                            bool
}

func NeedsFromBlank(workbook spreadsheet.Workbook, rule brand.RuleConfig, city string, variants ...string) ([]Need, error) {
	detection, err := detectNorthBlank(workbook, rule)
	if err != nil {
		return nil, err
	}
	variant := ""
	if len(variants) > 0 {
		variant = variants[0]
	}
	if rule.BlankLayout != "" {
		return specializedNeeds(detection, rule, city, variant)
	}
	qtyCol := detection.Columns[orderfill.ColumnQuantity]
	priceCol := northPriceColumn(detection)
	boxCol := orderfill.BlankBoxColumn(detection, rule)
	articleCol := detection.Columns[orderfill.ColumnArticle]
	nameCol := detection.Columns[orderfill.ColumnName]
	// Attach CHRISTINA PROFF line membership so PROFF rows earn the set discount
	// in the budget planner, mirroring the Tyumen flow and origin/main north.
	var linesByRow map[int]*orderfill.ChristinaLine
	if rule.Key == "christina" {
		linesByRow = orderfill.ChristinaProffLinesByRow(detection.Sheet)
	}
	bounds := detection.Sheet.Bounds()
	out := make([]Need, 0)
	for row := detection.HeaderRow + 1; row <= bounds.MaxRow; row++ {
		articleRaw := ""
		if articleCol > 0 {
			articleRaw = strings.TrimSpace(detection.Sheet.Value(row, articleCol))
		}
		rawQuantity := strings.TrimSpace(detection.Sheet.Value(row, qtyCol))
		qty, ok := normalize.ParseNumber(rawQuantity)
		if rawQuantity == "" {
			qty, ok = 0, true
		}
		quantityHeader := normalize.NormalizeHeader(rawQuantity)
		if articleRaw == "" && (quantityHeader == "заказ" || quantityHeader == "кол во" || quantityHeader == "количество") {
			continue
		}
		if !ok || qty < 0 || math.IsNaN(qty) || math.IsInf(qty, 0) {
			return nil, fmt.Errorf("%w: в строке %d указано неверное количество", orderfill.ErrInvalidInput, row)
		}
		name := ""
		if nameCol > 0 {
			name = strings.TrimSpace(detection.Sheet.Value(row, nameCol))
		}
		base := normalize.NormalizeArticle(articleRaw, brand.ArticleNormalizeOptions(rule))
		if base == "" {
			base = nameKey(name, "")
		}
		if base == "" {
			continue
		}
		key := base
		if variant != "" {
			key = variant + ":" + base
		}
		price := 0.0
		if priceCol > 0 {
			price, _ = orderfill.BudgetCellPrice(detection.Sheet, row, priceCol)
		}
		box, hasBox := normalize.ParseNumber(detection.Sheet.Value(row, boxCol))
		out = append(out, Need{Row: row, QuantityColumn: qtyCol, PriceColumn: priceCol, Prices: orderfill.BudgetPrices(detection.Sheet, detection.HeaderRow, row, qtyCol), City: city, Article: key, BaseArticle: base, ArticleRaw: articleRaw, Name: name, Variant: variant, Qty: qty, Price: price, Line: linesByRow[row], BoxSize: box, HasBoxSize: hasBox, SupplierUnitSize: 1})
	}
	return out, nil
}

func northPriceColumn(detection orderfill.Detection) int {
	preferred, discounted, fallback := 0, 0, 0
	for column := 1; column <= detection.Sheet.Bounds().MaxColumn; column++ {
		header := normalize.NormalizeHeader(detection.Sheet.Value(detection.HeaderRow, column))
		if !strings.Contains(header, "цена") || strings.Contains(header, "сумма") || strings.Contains(header, "итого") {
			continue
		}
		if header == "закупочная цена" || header == "цена закупки" {
			preferred = column
		} else if strings.Contains(header, "скид") {
			discounted = column
		} else if fallback == 0 {
			fallback = column
		}
	}
	if preferred > 0 {
		return preferred
	}
	if discounted > 0 {
		return discounted
	}
	return fallback
}

type SupplierLine struct {
	Key, Article, Name, Unit string
	Quantity                 float64
}

// WriteSupplierQuantities updates a selected summary blank using the same keys
// that were used during extraction. Unmentioned positions are cleared and
// ordered positions missing from the selected summary are appended at the end.
func WriteSupplierQuantities(workbook spreadsheet.Workbook, rule brand.RuleConfig, variant string, lines []SupplierLine) error {
	detection, err := detectNorthBlank(workbook, rule)
	if err != nil {
		return err
	}
	if rule.BlankLayout != "" {
		return writeSpecializedQuantities(detection, rule, variant, lines)
	}
	quantities := make(map[string]float64, len(lines))
	byKey := make(map[string]SupplierLine, len(lines))
	for _, line := range lines {
		if line.Quantity <= 0 {
			continue
		}
		quantities[line.Key] = line.Quantity
		byKey[line.Key] = line
	}
	articleCol := detection.Columns[orderfill.ColumnArticle]
	nameCol := detection.Columns[orderfill.ColumnName]
	unitCol := detection.Columns[orderfill.ColumnUnit]
	quantityCol := detection.Columns[orderfill.ColumnQuantity]
	written := make(map[string]bool, len(lines))
	for row := detection.HeaderRow + 1; row <= detection.Sheet.Bounds().MaxRow; row++ {
		article := ""
		if articleCol > 0 {
			article = normalize.NormalizeArticle(detection.Sheet.Value(row, articleCol), brand.ArticleNormalizeOptions(rule))
		}
		if article == "" && nameCol > 0 {
			article = nameKey(detection.Sheet.Value(row, nameCol), "")
		}
		if article == "" {
			continue
		}
		key := article
		if variant != "" {
			key = variant + ":" + article
		}
		written[key] = true
		quantity := quantities[key]
		if quantity > 0 {
			detection.Sheet.SetNumber(row, quantityCol, quantity)
		} else {
			detection.Sheet.ClearValue(row, quantityCol)
		}
	}
	nextRow := detection.Sheet.Bounds().MaxRow + 1
	for _, line := range lines {
		if line.Quantity <= 0 || written[line.Key] || variant != "" && !strings.HasPrefix(line.Key, variant+":") {
			continue
		}
		if variant == "" && (strings.HasPrefix(line.Key, "home:") || strings.HasPrefix(line.Key, "proff:")) {
			continue
		}
		if articleCol > 0 {
			detection.Sheet.SetText(nextRow, articleCol, cmp.Or(line.Article, BaseKey(line.Key)))
		}
		if nameCol > 0 {
			detection.Sheet.SetText(nextRow, nameCol, line.Name)
		}
		if unitCol > 0 {
			detection.Sheet.SetText(nextRow, unitCol, cmp.Or(line.Unit, "шт"))
		}
		detection.Sheet.SetNumber(nextRow, quantityCol, line.Quantity)
		nextRow++
	}
	return nil
}

func ApplySupplierDiscount(workbook spreadsheet.Workbook, rule brand.RuleConfig, discount float64) error {
	if discount <= 0 {
		return nil
	}
	if discount >= 100 || math.IsNaN(discount) || math.IsInf(discount, 0) {
		return fmt.Errorf("%w: скидка должна быть от 0 до 99,99%%", orderfill.ErrInvalidInput)
	}
	detection, err := detectNorthBlank(workbook, rule)
	if err != nil {
		return err
	}
	if rule.BlankLayout != "" {
		return discountSpecializedPrices(detection, rule, discount)
	}
	priceColumn := northPriceColumn(detection)
	if priceColumn == 0 {
		return fmt.Errorf("%w: не найдена закупочная цена для применения скидки", orderfill.ErrInvalidInput)
	}
	rows := []orderfill.ReportRow{}
	entries := []orderfill.BudgetPricing{}
	for row := detection.HeaderRow + 1; row <= detection.Sheet.Bounds().MaxRow; row++ {
		base, ok := orderfill.BudgetCellPrice(detection.Sheet, row, priceColumn)
		if !ok || base <= 0 {
			continue
		}
		qty, _ := normalize.ParseNumber(detection.Sheet.Value(row, detection.Columns[orderfill.ColumnQuantity]))
		key := fmt.Sprintf("north:%d", row)
		rows = append(rows, orderfill.ReportRow{Key: key, BlankRow: row, BlankQuantityColumn: detection.Columns[orderfill.ColumnQuantity], Inserted: new(qty), BlankName: detection.Sheet.Value(row, detection.Columns[orderfill.ColumnName])})
		price := math.Floor(base*(1-discount/100)*100+0.5) / 100
		entries = append(entries, orderfill.BudgetPricing{RowKey: key, PriceColumn: priceColumn, BasePrice: base, Discount: discount, Price: price})
	}
	return orderfill.ApplyBudgetPricing(detection.Sheet, detection.HeaderRow, rule.Key, rows, entries)
}

func BaseKey(key string) string {
	if strings.HasPrefix(key, "home:") || strings.HasPrefix(key, "proff:") {
		_, base, _ := strings.Cut(key, ":")
		return base
	}
	return key
}

func detectNorthBlank(workbook spreadsheet.Workbook, rule brand.RuleConfig) (orderfill.Detection, error) {
	if rule.BlankLayout != "" {
		return orderfill.DetectOrderBlank(workbook, rule)
	}
	if detection, err := orderfill.DetectBlankColumns(workbook, rule); err == nil {
		return detection, nil
	}
	for _, sheet := range workbook.Sheets() {
		bounds := sheet.Bounds()
		for row := 1; row <= min(bounds.MaxRow, 120); row++ {
			columns := map[string]int{}
			for column := 1; column <= bounds.MaxColumn; column++ {
				header := normalize.NormalizeHeader(sheet.Value(row, column))
				switch {
				case strings.Contains(header, "артикул") || header == "арт" || header == "код":
					columns[orderfill.ColumnArticle] = column
				case strings.Contains(header, "наименование") || strings.Contains(header, "название") || strings.Contains(header, "описание") || header == "товар":
					columns[orderfill.ColumnName] = column
				case header == "заказ" || header == "количество" || strings.Contains(header, "кол во") || strings.Contains(header, "количество заказа"):
					if !strings.Contains(header, "короб") && !strings.Contains(header, "упак") {
						columns[orderfill.ColumnQuantity] = column
					}
				}
			}
			if columns[orderfill.ColumnName] > 0 && columns[orderfill.ColumnQuantity] > 0 {
				return orderfill.Detection{Sheet: sheet, SheetName: sheet.Name(), HeaderRow: row, Columns: columns}, nil
			}
		}
	}
	return orderfill.Detection{}, fmt.Errorf("%w: не удалось распознать товар и колонку заказа в бланке", orderfill.ErrInvalidInput)
}

func nameKey(name, unit string) string {
	normalized := normalize.NormalizeName(strings.TrimSpace(name + " " + unit))
	if normalized == "" {
		return ""
	}
	return "name:" + normalized
}

func StockFromSource(workbook spreadsheet.Workbook, rules ...brand.RuleConfig) ([]Stock, error) {
	detection, err := orderfill.DetectSourceColumns(workbook)
	if err != nil {
		return nil, err
	}
	rule := brand.Rule("angiopharm")
	if len(rules) > 0 {
		rule = rules[0]
	}
	articleCol := detection.Columns[orderfill.ColumnArticle]
	nameCol := detection.Columns[orderfill.ColumnName]
	stockCol := detection.Columns[orderfill.ColumnStock]
	transitCol := detection.Columns[orderfill.ColumnInTransit]
	recommendedCol := detection.Columns[orderfill.ColumnRecommended]
	actualCol := detection.Columns[orderfill.ColumnOrderedFact]
	bounds := detection.Sheet.Bounds()
	targetCol := 0
	categoryCol := 0
	demandCol := 0
	warehouseStockCol, warehouseTransitCol := 0, 0
	for column := 1; column <= bounds.MaxColumn; column++ {
		header := normalize.NormalizeHeader(detection.Sheet.Value(detection.HeaderRow, column))
		if header == "склад остаток" {
			warehouseStockCol = column
		}
		if header == "склад в пути" {
			warehouseTransitCol = column
		}
		if strings.Contains(header, "целевой") && strings.Contains(header, "запас") {
			targetCol = column
		}
		if strings.Contains(header, "категор") || header == "abc" {
			categoryCol = column
		}
		if strings.Contains(header, "среднемес") || strings.Contains(header, "средн месяч") || strings.Contains(header, "среднее в месяц") {
			demandCol = column
		}
	}
	out := make([]Stock, 0)
	for row := detection.HeaderRow + 1; row <= bounds.MaxRow; row++ {
		total := false
		for col := 1; col <= min(bounds.MaxColumn, max(1, nameCol-1)); col++ {
			header := normalize.NormalizeHeader(detection.Sheet.Value(row, col))
			if header == "итого" || header == "total" {
				total = true
			}
		}
		if total {
			continue
		}
		article := normalize.NormalizeArticle(detection.Sheet.Value(row, articleCol), brand.ArticleNormalizeOptions(rule))
		item := Stock{Article: article, SourceRow: row}
		if nameCol > 0 {
			item.Name = strings.TrimSpace(detection.Sheet.Value(row, nameCol))
		}
		if normalize.AsText(detection.Sheet.Value(row, articleCol)) == "" && item.Name == "" {
			continue
		}
		if rule.Key == "novacutan" {
			item.Article = orderfill.NovacutanProductKey(item.Name)
		}
		if item.Article == "" {
			item.Article = nameKey(item.Name, "")
		}
		if item.Article == "" {
			continue
		}
		if warehouseStockCol > 0 {
			item.WarehouseStock, _ = normalize.ParseNumber(detection.Sheet.Value(row, warehouseStockCol))
			item.HasWarehouseStock = true
		}
		if warehouseTransitCol > 0 {
			item.WarehouseTransit, _ = normalize.ParseNumber(detection.Sheet.Value(row, warehouseTransitCol))
		}
		if stockCol > 0 {
			item.Stock, _ = normalize.ParseNumber(detection.Sheet.Value(row, stockCol))
		}
		if transitCol > 0 {
			item.InTransit, _ = normalize.ParseNumber(detection.Sheet.Value(row, transitCol))
		}
		if recommendedCol > 0 {
			item.Recommended, _ = normalize.ParseNumber(detection.Sheet.Value(row, recommendedCol))
		}
		if actualCol > 0 {
			raw := detection.Sheet.Value(row, actualCol)
			item.Actual, _ = normalize.ParseNumber(raw)
			item.HasActual = strings.TrimSpace(raw) != ""
		}
		if targetCol > 0 {
			item.Target, _ = normalize.ParseNumber(detection.Sheet.Value(row, targetCol))
		} else {
			item.Target = max(0, item.Stock+item.InTransit+item.Recommended)
		}
		if categoryCol > 0 {
			item.Category = strings.Split(normalize.NormalizeCategory(detection.Sheet.Value(row, categoryCol)), "/")[0]
		}
		if demandCol > 0 {
			item.MonthlyDemand, _ = normalize.ParseNumber(detection.Sheet.Value(row, demandCol))
		}
		out = append(out, item)
	}
	return out, nil
}

// ApplySupplierPricing uses the selected summary template as the price source;
// finalization loads that original template on every download.
func ApplySupplierPricing(workbook spreadsheet.Workbook, rule brand.RuleConfig, variant string, plans []PlanRow) error {
	detection, err := detectNorthBlank(workbook, rule)
	if err != nil {
		return err
	}
	needs, err := NeedsFromBlank(workbook, rule, "", variant)
	if err != nil {
		return err
	}
	byKey := map[string]PlanRow{}
	for _, plan := range plans {
		byKey[plan.Key] = plan
	}
	rows := []orderfill.ReportRow{}
	entries := []orderfill.BudgetPricing{}
	for _, need := range needs {
		plan, found := byKey[need.Article]
		if !found || plan.BudgetDiscount == 0 && plan.BudgetPriceColumn == 0 {
			continue
		}
		column := plan.BudgetPriceColumn
		if column == 0 {
			column = plan.BudgetDefaultPriceColumn
		}
		selectedLabel := ""
		selectedBase := 0.0
		for _, price := range plan.BudgetPrices {
			if price.Column == column {
				selectedLabel = price.Label
				selectedBase = price.Price
				break
			}
		}
		if selectedLabel != "" {
			column = 0
			for _, price := range need.Prices {
				if price.Label == selectedLabel {
					column = price.Column
					break
				}
			}
		} else if column == 0 {
			column = need.PriceColumn
		}
		valid := false
		for _, price := range need.Prices {
			if price.Column == column {
				valid = true
			}
		}
		if !valid {
			return fmt.Errorf("%w: выбранная колонка цены отсутствует в общем бланке: %s", orderfill.ErrInvalidInput, plan.Name)
		}
		base, ok := orderfill.BudgetCellPrice(detection.Sheet, need.Row, column)
		if !ok || base <= 0 {
			if plan.ActualSupplierOrder > 0 {
				return fmt.Errorf("%w: не определена цена заказанной позиции: %s", orderfill.ErrInvalidInput, plan.Name)
			}
			continue
		}
		// Report price options were taken from this same selected template. Refuse a
		// mismatch instead of exporting a different invoice from the budget preview.
		if selectedLabel != "" && math.Abs(selectedBase-base) > 0.000001 {
			return fmt.Errorf("%w: цена общего бланка изменилась: %s", orderfill.ErrInvalidInput, plan.Name)
		}
		key := fmt.Sprintf("north:%d:%d", need.Row, need.QuantityColumn)
		rows = append(rows, orderfill.ReportRow{Key: key, BlankRow: need.Row, BlankQuantityColumn: need.QuantityColumn, BlankName: need.Name, Inserted: new(plan.ActualSupplierOrder)})
		price := math.Floor((base*(1-plan.BudgetDiscount/100)+2.220446049250313e-16)*100+0.5) / 100
		entries = append(entries, orderfill.BudgetPricing{RowKey: key, PriceColumn: column, BasePrice: base, Discount: plan.BudgetDiscount, Price: price})
	}
	return orderfill.ApplyBudgetPricing(detection.Sheet, detection.HeaderRow, rule.Key, rows, entries)
}
