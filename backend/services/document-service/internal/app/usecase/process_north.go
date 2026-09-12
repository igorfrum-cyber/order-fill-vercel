package usecase

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"order-fill/backend/services/document-service/internal/app/port"
	"order-fill/backend/services/document-service/internal/clients/calculation"
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
		case port.RoleBlank:
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
	for _, blank := range blanks {
		cityKey, _, ok := north.CityFromFileName(blank.Name)
		if !ok {
			return fmt.Errorf("%w: не узнали город по имени файла %q. Назовите файл городом: Сургут, Вартовск, Уренгой или Тюмень", orderfill.ErrInvalidInput, blank.Name)
		}
		workbook, err := u.loadWorkbook(ctx, blank.StorageKey, nil)
		if err != nil {
			return err
		}
		extracted, err := north.NeedsFromBlank(workbook, rule, cityKey)
		if err != nil {
			return err
		}
		needs = append(needs, extracted...)
		for _, need := range extracted {
			calcNeeds = append(calcNeeds, calculation.NorthNeed{City: need.City, Article: need.Article, Name: need.Name, Qty: need.Qty})
		}
		groupsByCity[cityKey] = append(groupsByCity[cityKey], orderfill.LabelChristinaBlank(workbook, blank.Name))
		output, err := u.saveWorkbook(ctx, message.JobID, workbook, orderfill.BlankOutputFileName(blank.Name, ""), "Скачать бланк города")
		if err != nil {
			return err
		}
		outputs = append(outputs, output)
		workbooks = append(workbooks, workbook)
	}

	var stock []north.Stock
	var calcStock []calculation.TyumenStock
	if source.StorageKey != "" {
		progress.Set(ctx, 0.45, "Читаю таблицу Тюмени")
		workbook, err := u.loadWorkbook(ctx, source.StorageKey, nil)
		if err != nil {
			return err
		}
		stock, err = north.StockFromSource(workbook)
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
		warehouseStock, err = north.StockFromSource(workbook)
		if err != nil {
			return err
		}
	}
	stock, calcStock = combineTyumenLocations(stock, warehouseStock)

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
			Article: row.Article, Name: row.Name, Comment: row.Comment,
			TyumenQty: row.TyumenQty, TransferQty: row.TransferQty, SupplierQty: row.SupplierQty,
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
	report := north.BuildReport(brandKey, needs, stock, planned, groups)
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
		edits = append(edits, north.Edit{Key: edit.Key, Value: edit.Value})
	}
	if err := north.ApplyEdits(&report, edits); err != nil {
		return err
	}
	payload, err := json.Marshal(report)
	if err != nil {
		return err
	}
	if err := u.storage.Put(ctx, "jobs/"+message.JobID+"/report.json", "application/json", payload); err != nil {
		return fmt.Errorf("save north report: %w", err)
	}

	progress.Set(ctx, 0.6, "Готовлю файлы")
	outputs, err := u.jobs.Outputs(ctx, message.JobID)
	if err != nil {
		return fmt.Errorf("load job outputs: %w", err)
	}
	workbooks := make([]spreadsheet.Workbook, 0, len(outputs))
	for _, output := range outputs {
		workbook, err := u.loadWorkbook(ctx, output.StorageKey, nil)
		if err != nil {
			return err
		}
		workbooks = append(workbooks, workbook)
	}
	progress.Set(ctx, 0.85, "Готовлю превью")
	if err := u.writePreviews(ctx, message.JobID, outputs, workbooks); err != nil {
		return err
	}
	return u.jobs.SaveResult(ctx, message.JobID, "completed", outputs, u.now())
}
