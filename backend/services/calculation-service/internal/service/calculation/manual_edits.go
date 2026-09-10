package calculation

import "order-fill/backend/services/calculation-service/internal/domain"

type ManualEdit struct {
	RowID   string
	Qty     float64
	Comment string
}

func (s *Service) ValidateManualEdits(edits []ManualEdit, rows []domain.OrderRow) (ok bool, blocking []string) {
	known := map[string]bool{}
	for _, row := range rows {
		known[row.ID] = true
	}
	for _, edit := range edits {
		if edit.Qty != 0 && edit.Comment == "" {
			blocking = append(blocking, edit.RowID)
		}
		if edit.RowID != "" && !known[edit.RowID] && len(rows) > 0 {
			blocking = append(blocking, edit.RowID)
		}
	}
	return len(blocking) == 0, blocking
}
