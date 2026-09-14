package usecase

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"order-fill/backend/services/document-service/internal/app/port"
	"order-fill/backend/services/document-service/internal/clients/calculation"
	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/north"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

func (u *ProcessJob) processNorth(ctx context.Context, message port.JobMessage, progress *jobProgress) error {
	if u.calc == nil {
		return fmt.Errorf("%w: calculation-service is required for north merge", orderfill.ErrInvalidInput)
	}
	if u.brands == nil {
		return fmt.Errorf("%w: brand-service is required for north merge", orderfill.ErrInvalidInput)
	}
	progress.Set(ctx, 0.1, "Читаю бланки городов")
	var source port.MessageFile
	var warehouse port.MessageFile
	blanks := make([]port.MessageFile, 0)
	for _, input := range message.Inputs {
		switch input.Role {
		case port.RoleSource:
			source = input
		case port.RoleWarehouse:
			warehouse = input
		case port.RoleBlank, port.RoleBlankHome, port.RoleBlankProff:
			blanks = append(blanks, input)
		}
	}
	if len(blanks) == 0 {
		return fmt.Errorf("%w: добавьте хотя бы один бланк города", orderfill.ErrInvalidInput)
	}
	brandKey := cmp.Or(message.Brand, "angiopharm")
	rule, err := u.brands.Policy(ctx, brandKey, "")
	if err != nil {
		return fmt.Errorf("brand policy: %w", err)
	}
	needs := make([]north.Need, 0)
	calcNeeds := make([]calculation.NorthNeed, 0)
	groupsByCity := map[string][]string{}
	outputs := make([]port.OutputFile, 0, len(blanks))
	workbooks := make([]spreadsheet.Workbook, 0, len(blanks))
	seenCityVariant := make(map[string]bool)
	for _, blank := range blanks {
		workbook, err := u.loadWorkbook(ctx, blank.StorageKey, nil)
		if err != nil {
			return err
		}
		cityKey, _, ok := north.CityFromWorkbook(workbook, blank.Name)
		if !ok {
			return fmt.Errorf("%w: не узнали город по имени файла %q. Назовите файл городом: Сургут, Вартовск, Уренгой или Тюмень", orderfill.ErrInvalidInput, blank.Name)
		}
		variant := north.VariantFromRole(blank.Role)
		cityVariant := cityKey + ":" + cmp.Or(variant, "default")
		if seenCityVariant[cityVariant] {
			return fmt.Errorf("%w: загружено несколько бланков %s %s; оставьте один файл на город и тип бланка", orderfill.ErrInvalidInput, north.Label(cityKey), strings.ToUpper(variant))
		}
		seenCityVariant[cityVariant] = true
		extracted, err := north.NeedsFromBlank(workbook, rule, cityKey, variant)
		if err != nil {
			return err
		}
		needs = append(needs, extracted...)
		for _, need := range extracted {
			calcNeeds = append(calcNeeds, calculation.NorthNeed{City: need.City, Article: need.Article, Name: need.Name, Qty: need.Qty})
		}
		label := orderfill.LabelChristinaBlank(workbook, blank.Name)
		if variant != "" {
			label = strings.ToUpper(variant)
		}
		groupsByCity[cityKey] = append(groupsByCity[cityKey], label)
		output, err := u.saveWorkbook(ctx, message.JobID, workbook, orderfill.BlankOutputFileName(blank.Name, ""), "Скачать бланк города")
		if err != nil {
			return err
		}
		outputs = append(outputs, output)
		workbooks = append(workbooks, workbook)
	}

	var stock []north.Stock
	var calcStock []calculation.TyumenStock
	deliveryWeeks := 1.0
	if source.StorageKey != "" {
		progress.Set(ctx, 0.45, "Читаю таблицу Тюмени")
		workbook, err := u.loadWorkbook(ctx, source.StorageKey, nil)
		if err != nil {
			return err
		}
		if city, _, ok := north.CityFromWorkbook(workbook, source.Name); !ok || city != "tyumen" {
			return fmt.Errorf("%w: в поле таблицы Тюмени нужна именно тюменская таблица", orderfill.ErrInvalidInput)
		}
		deliveryWeeks = orderfill.DeliveryWeeks(workbook)
		stock, err = north.StockFromSource(workbook, rule)
		if err != nil {
			return err
		}
	}
	var warehouseStock []north.Stock
	if warehouse.StorageKey != "" {
		progress.Set(ctx, 0.55, "Читаю таблицу склада доставки")
		workbook, err := u.loadWorkbook(ctx, warehouse.StorageKey, nil)
		if err != nil {
			return err
		}
		warehouseStock, err = north.StockFromSource(workbook, rule)
		if err != nil {
			return err
		}
	}
	stock, calcStock = combineTyumenLocations(stock, warehouseStock)
	stockKeys := make(map[string]bool, len(stock))
	for _, item := range stock {
		stockKeys[item.Article] = true
	}
	needs = slices.DeleteFunc(needs, func(need north.Need) bool {
		return need.Qty == 0 && !stockKeys[need.BaseArticle]
	})
	calcNeeds = slices.DeleteFunc(calcNeeds, func(need calculation.NorthNeed) bool {
		return need.Qty == 0 && !stockKeys[north.BaseKey(need.Article)]
	})
	stockByArticle := make(map[string]north.Stock, len(stock))
	for _, item := range stock {
		stockByArticle[item.Article] = item
	}
	seenTyumenNeed := make(map[string]bool)
	for _, need := range calcNeeds {
		if need.City == "tyumen" {
			seenTyumenNeed[need.Article] = true
		}
	}
	seenNeedKey := make(map[string]bool)
	for _, need := range needs {
		if seenNeedKey[need.Article] || seenTyumenNeed[need.Article] {
			continue
		}
		seenNeedKey[need.Article] = true
		item, ok := stockByArticle[need.BaseArticle]
		if !ok {
			continue
		}
		planned := item.Recommended
		if item.HasActual {
			planned = item.Actual
		}
		calcNeeds = append(calcNeeds, calculation.NorthNeed{City: "tyumen", Article: need.Article, Name: need.Name, Qty: max(0, planned)})
	}

	progress.Set(ctx, 0.7, "Считаю план")
	if err := u.jobs.SetIdentity(ctx, message.JobID, brandKey, "", u.now()); err != nil {
		return fmt.Errorf("save detected brand: %w", err)
	}
	plannedRows, err := u.calc.NorthPlan(ctx, brandKey, calcNeeds, calcStock)
	if err != nil {
		return fmt.Errorf("north plan: %w", err)
	}
	planned := make([]north.Planned, 0, len(plannedRows))
	for _, row := range plannedRows {
		planned = append(planned, north.Planned{
			Article: row.Article, Name: row.Name, Variant: row.Variant, Comment: row.Comment,
			TyumenQty: row.TyumenQty, TransferQty: row.TransferQty, SupplierQty: row.SupplierQty,
			TyumenStock: row.TyumenStock, TyumenTransit: row.TyumenTransit, TyumenTarget: row.TyumenTarget,
			UnitSize: row.UnitSize, NovacutanMin: row.NovacutanMin, BoxSize: row.BoxSize, HasBoxSize: row.HasBoxSize,
			WarehouseStock: row.WarehouseStock, WarehouseTransit: row.WarehouseTransit, HasWarehouseStock: row.HasWarehouseStock,
		})
	}

	groups := make([]north.ConfirmationGroup, 0)
	for city, variants := range groupsByCity {
		if len(variants) < 2 {
			continue
		}
		groups = append(groups, north.ConfirmationGroup{
			City:     north.CityQty{Key: city, Label: north.Label(city)},
			Variants: variants,
		})
	}
	report := north.BuildReport(brandKey, needs, stock, planned, groups, deliveryWeeks)
	payload, err := json.Marshal(report)
	if err != nil {
		return err
	}
	progress.Set(ctx, 0.9, "Сохраняю отчёт")
	if err := u.storage.Put(ctx, "jobs/"+message.JobID+"/report.json", "application/json", payload); err != nil {
		return fmt.Errorf("save north report: %w", err)
	}
	outputs = assignOutputIDs(outputs)
	progress.Set(ctx, 0.92, "Готовлю превью")
	if err := u.writePreviews(ctx, message.JobID, outputs, workbooks); err != nil {
		return err
	}
	if err := u.jobs.SaveResult(ctx, message.JobID, "needs_review", outputs, u.now()); err != nil {
		return fmt.Errorf("save job result: %w", err)
	}
	return nil
}

func combineTyumenLocations(office, warehouse []north.Stock) ([]north.Stock, []calculation.TyumenStock) {
	byArticle := make(map[string]north.Stock, len(office)+len(warehouse))
	for _, item := range office {
		byArticle[item.Article] = item
	}
	warehouseByArticle := make(map[string]north.Stock, len(warehouse))
	for _, item := range warehouse {
		warehouseByArticle[item.Article] = item
		combined := byArticle[item.Article]
		combined.Article = item.Article
		if combined.Name == "" {
			combined.Name = item.Name
		}
		combined.Stock += item.Stock
		combined.InTransit += item.InTransit
		combined.Target += item.Target
		byArticle[item.Article] = combined
	}
	articles := make([]string, 0, len(byArticle))
	for article := range byArticle {
		articles = append(articles, article)
	}
	slices.Sort(articles)
	combined := make([]north.Stock, 0, len(articles))
	calculationRows := make([]calculation.TyumenStock, 0, len(articles))
	for _, article := range articles {
		item := byArticle[article]
		warehouseItem, hasWarehouse := warehouseByArticle[article]
		combined = append(combined, item)
		calculationRows = append(calculationRows, calculation.TyumenStock{
			Article: item.Article, Name: item.Name, Stock: item.Stock, InTransit: item.InTransit, Target: item.Target,
			WarehouseStock: warehouseItem.Stock, WarehouseTransit: warehouseItem.InTransit, HasWarehouseStock: hasWarehouse,
		})
	}
	return combined, calculationRows
}

func (u *ProcessJob) finalizeNorth(ctx context.Context, message port.JobMessage) error {
	progress := newJobProgress(u.jobs, message.JobID, u.now)
	progress.Set(ctx, 0.3, "Применяю правки")
	raw, err := u.storage.Get(ctx, "jobs/"+message.JobID+"/report.json")
	if err != nil {
		return fmt.Errorf("load north report: %w", err)
	}
	var report north.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("decode north report: %w", err)
	}
	edits := make([]north.Edit, 0, len(message.Edits))
	for _, edit := range message.Edits {
		edits = append(edits, north.Edit{Key: edit.Key, Value: edit.Value, Comment: edit.Comment})
	}
	if err := north.ApplyEdits(&report, edits); err != nil {
		return err
	}
	actualByKey := make(map[string]float64, len(report.PlanRows))
	calcNeeds := make([]calculation.NorthNeed, 0)
	calcStock := make([]calculation.TyumenStock, 0)
	seenStock := make(map[string]bool)
	for _, row := range report.PlanRows {
		actualByKey[row.Key] = row.ActualSupplierOrder
		hasTyumen := false
		for _, city := range row.Cities {
			calcNeeds = append(calcNeeds, calculation.NorthNeed{City: city.Key, Article: row.Key, Name: row.Name, Qty: city.Quantity})
			hasTyumen = hasTyumen || city.Key == "tyumen"
		}
		if !hasTyumen && row.TyumenPlannedOrder > 0 {
			calcNeeds = append(calcNeeds, calculation.NorthNeed{City: "tyumen", Article: row.Key, Name: row.Name, Qty: row.TyumenPlannedOrder})
		}
		baseKey := row.Key
		if row.Variant != "" {
			_, baseKey, _ = strings.Cut(row.Key, ":")
		}
		if row.HasTyumenSource && !seenStock[baseKey] {
			seenStock[baseKey] = true
			calcStock = append(calcStock, calculation.TyumenStock{
				Article: baseKey, Name: row.Name, Stock: row.TyumenStock, InTransit: row.TyumenInTransit, Target: row.TyumenTarget,
				WarehouseStock: row.WarehouseStock, WarehouseTransit: row.WarehouseTransit, HasWarehouseStock: row.HasWarehouseStock,
			})
		}
	}
	plannedRows, err := u.calc.NorthPlan(ctx, report.Summary.Kind, calcNeeds, calcStock)
	if err != nil {
		return fmt.Errorf("recalculate north plan: %w", err)
	}
	plannedByKey := make(map[string]calculation.NorthRow, len(plannedRows))
	for _, row := range plannedRows {
		plannedByKey[row.Article] = row
	}
	report.Transfers = report.Transfers[:0]
	for index := range report.PlanRows {
		row := &report.PlanRows[index]
		planned := plannedByKey[row.Key]
		row.FromTyumen = planned.TransferQty
		row.SupplierNeed = planned.SupplierQty
		row.ActualSupplierOrder = actualByKey[row.Key]
		row.TyumenPlannedOrder = planned.TyumenQty
		row.Comment = planned.Comment
		row.NorthNeed = 0
		for _, city := range row.Cities {
			if city.Key != "tyumen" {
				row.NorthNeed += city.Quantity
				report.Transfers = append(report.Transfers, north.Transfer{Article: row.Article, Qty: city.Quantity})
			}
		}
		north.PopulateAllocation(row)
	}
	payload, err := json.Marshal(report)
	if err != nil {
		return err
	}
	if err := u.storage.Put(ctx, "jobs/"+message.JobID+"/report.json", "application/json", payload); err != nil {
		return fmt.Errorf("save north report: %w", err)
	}

	progress.Set(ctx, 0.6, "Готовлю файлы")
	rule, err := u.brands.Policy(ctx, report.Summary.Kind, "")
	if err != nil {
		return fmt.Errorf("brand policy: %w", err)
	}
	outputs, workbooks, err := u.buildNorthOutputs(ctx, message, report, rule)
	if err != nil {
		return err
	}
	outputs = assignOutputIDs(outputs)
	progress.Set(ctx, 0.85, "Готовлю превью")
	if err := u.writePreviews(ctx, message.JobID, outputs, workbooks); err != nil {
		return err
	}
	return u.jobs.SaveResult(ctx, message.JobID, "completed", outputs, u.now())
}

type northSummaryCandidate struct {
	workbook spreadsheet.Workbook
	name     string
	city     string
	variant  string
}

func (u *ProcessJob) buildNorthOutputs(ctx context.Context, message port.JobMessage, report north.Report, rule brand.RuleConfig) ([]port.OutputFile, []spreadsheet.Workbook, error) {
	candidates := make(map[string]northSummaryCandidate)
	for _, input := range message.Inputs {
		variant := north.VariantFromRole(input.Role)
		if input.Role != port.RoleBlank && variant == "" {
			continue
		}
		workbook, err := u.loadWorkbook(ctx, input.StorageKey, nil)
		if err != nil {
			return nil, nil, err
		}
		city, _, _ := north.CityFromWorkbook(workbook, input.Name)
		current, exists := candidates[variant]
		if !exists || city == "tyumen" && current.city != "tyumen" {
			candidates[variant] = northSummaryCandidate{workbook: workbook, name: input.Name, city: city, variant: variant}
		}
	}
	lines := make([]north.SupplierLine, 0, len(report.PlanRows))
	for _, row := range report.PlanRows {
		lines = append(lines, north.SupplierLine{
			Key: row.Key, Article: cmp.Or(row.ArticleRaw, north.BaseKey(row.Article)), Name: row.Name,
			Unit: "шт", Quantity: row.ActualSupplierOrder,
		})
	}
	variants := []string{""}
	if _, home := candidates["home"]; home {
		variants = []string{"home", "proff"}
	}
	outputs := make([]port.OutputFile, 0)
	workbooks := make([]spreadsheet.Workbook, 0)
	for _, variant := range variants {
		candidate, ok := candidates[variant]
		if !ok {
			continue
		}
		if err := north.WriteSupplierQuantities(candidate.workbook, rule, variant, lines); err != nil {
			return nil, nil, err
		}
		discount := 0.0
		for _, row := range report.PlanRows {
			if row.Variant == variant && row.BudgetDiscount > 0 {
				discount = row.BudgetDiscount
				break
			}
		}
		if err := north.ApplySupplierDiscount(candidate.workbook, rule, discount); err != nil {
			return nil, nil, err
		}
		variantLabel := ""
		if variant != "" {
			variantLabel = " " + strings.ToUpper(variant)
		}
		name := "Север общий бланк" + variantLabel + northOutputExtension(candidate.name)
		output, err := u.saveWorkbook(ctx, message.JobID, candidate.workbook, name, "Скачать общий бланк"+variantLabel)
		if err != nil {
			return nil, nil, err
		}
		outputs = append(outputs, output)
		workbooks = append(workbooks, candidate.workbook)
	}
	creator, ok := u.codec.(spreadsheet.TableCodec)
	if !ok {
		return outputs, workbooks, nil
	}
	for _, cityKey := range []string{"surgut", "nizhnevartovsk", "urengoy"} {
		rows := make([][]any, 0)
		for _, row := range report.PlanRows {
			for _, city := range row.Cities {
				if city.Key == cityKey && city.Quantity > 0 {
					rows = append(rows, []any{cmp.Or(row.ArticleRaw, north.BaseKey(row.Article)), row.Name, "шт", city.Quantity})
				}
			}
		}
		if len(rows) == 0 {
			continue
		}
		label := north.Label(cityKey)
		workbook, err := creator.NewTable("Перемещение", []string{"Артикул", "Наименование", "Ед.", "Количество"}, rows)
		if err != nil {
			return nil, nil, fmt.Errorf("create transfer workbook: %w", err)
		}
		output, err := u.saveWorkbook(ctx, message.JobID, workbook, "Заказ на перемещение "+label+".xlsx", "Скачать перемещение в "+label)
		if err != nil {
			return nil, nil, err
		}
		outputs = append(outputs, output)
		workbooks = append(workbooks, workbook)
	}
	return outputs, workbooks, nil
}

func northOutputExtension(name string) string {
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(name)), ".xlsm") {
		return ".xlsm"
	}
	return ".xlsx"
}
