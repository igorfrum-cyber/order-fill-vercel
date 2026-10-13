package usecase

import (
	"context"
	"fmt"
	"order-fill/backend/services/document-service/internal/app/port"
	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
	"strings"
)

func (u *ProcessJob) processTable(ctx context.Context, message port.JobMessage, progress *jobProgress) error {
	var input, warehouse port.MessageFile
	for _, file := range message.Inputs {
		switch file.Role {
		case port.RoleSource:
			input = file
		case port.RoleWarehouse:
			warehouse = file
		}
	}
	if input.StorageKey == "" {
		return fmt.Errorf("%w: загрузите таблицу продаж", orderfill.ErrInvalidInput)
	}
	book, err := u.loadWorkbook(ctx, input.StorageKey, nil)
	if err != nil {
		return err
	}
	key, month, rule, err := u.resolveIdentity(ctx, book, "")
	if err != nil {
		return err
	}
	if message.OrderMonth != "" {
		month = message.OrderMonth
	}
	if err := u.jobs.SetIdentity(ctx, message.JobID, key, month, u.now()); err != nil {
		return fmt.Errorf("save table identity: %w", err)
	}
	if warehouse.StorageKey != "" {
		other, err := u.loadWorkbook(ctx, warehouse.StorageKey, nil)
		if err != nil {
			return err
		}
		book, err = orderfill.MergeTyumenSources(book, other, month, rule, orderfill.TyumenMergeOptions{PreserveFacts: true, Chz: chzMerger(u.match), MatchingMode: message.MatchingMode, Context: ctx})
		if err != nil {
			return err
		}
	}
	progress.Set(ctx, .55, "Пересчитываю таблицу")
	source, err := orderfill.RecalculateTable(orderfill.FillCommand{Source: book, SourceFileName: input.Name, Brand: key, OrderMonth: month, Rule: rule, Chz: chzMerger(u.match), Recommender: sourceRecommender(u.calc), Context: ctx, MatchingMode: message.MatchingMode})
	if err != nil {
		return err
	}
	rows := make([]orderfill.ReportRow, 0, len(source.Items))
	counts := map[string]int{}
	for _, item := range source.Items {
		key := item.Article
		if key == "" {
			key = normalize.NormalizeName(item.Name)
		}
		counts[key]++
	}
	for _, item := range source.Items {
		category := orderfill.CategoryOrderNotNeeded
		key := item.Article
		if key == "" {
			key = normalize.NormalizeName(item.Name)
		}
		if counts[key] > 1 || strings.Contains(item.SourceComment, "Проверить:") {
			category = orderfill.CategoryNeedsDecision
		}
		rows = append(rows, orderfill.ReportRow{Key: fmt.Sprintf("table:%d", item.RowIndex), Category: category, SourceRow: new(item.RowIndex), SourceArticle: item.ArticleRaw, SourceName: item.Name, Recommended: new(item.Recommended), OrderedFact: new(item.OrderedFact), HasOrderedFact: item.HasOrderedFact, SourceComment: item.SourceComment, Stock: item.Stock, InTransit: item.InTransit})
	}
	output, err := u.saveWorkbook(ctx, message.JobID, book, orderfill.SourceOutputFileName(input.Name), "Скачать пересчитанную таблицу")
	if err != nil {
		return err
	}
	outputs := assignOutputIDs([]port.OutputFile{output})
	if err := u.writePreviews(ctx, message.JobID, outputs, []spreadsheet.Workbook{book}); err != nil {
		return err
	}
	if err := u.reports.Save(ctx, message.JobID, orderfill.Summary{Brand: key, SourceItems: len(rows)}, rows, u.now()); err != nil {
		return fmt.Errorf("save table report: %w", err)
	}
	return u.jobs.SaveResult(ctx, message.JobID, "completed", outputs, u.now())
}
