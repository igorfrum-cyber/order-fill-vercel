package orderfill

import (
	"sort"
	"strconv"

	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

// fakeWorkbook is an in-memory spreadsheet used to exercise the engine without
// touching the xlsx adapter.
type fakeWorkbook struct {
	sheets []*fakeSheet
}

type fakeSheet struct {
	name  string
	cells map[[2]int]string
}

func newFakeWorkbook(name string, grid [][]string) *fakeWorkbook {
	sheet := &fakeSheet{name: name, cells: map[[2]int]string{}}
	for rowIndex, row := range grid {
		for columnIndex, value := range row {
			if value == "" {
				continue
			}
			sheet.cells[[2]int{rowIndex + 1, columnIndex + 1}] = value
		}
	}
	return &fakeWorkbook{sheets: []*fakeSheet{sheet}}
}

func (w *fakeWorkbook) Sheets() []spreadsheet.Sheet {
	sheets := make([]spreadsheet.Sheet, 0, len(w.sheets))
	for _, sheet := range w.sheets {
		sheets = append(sheets, sheet)
	}
	return sheets
}

func (w *fakeWorkbook) Sheet(name string) (spreadsheet.Sheet, bool) {
	for _, sheet := range w.sheets {
		if sheet.name == name {
			return sheet, true
		}
	}
	return nil, false
}

func (w *fakeWorkbook) Save() ([]byte, error) { return nil, nil }

func (s *fakeSheet) Name() string { return s.name }

func (s *fakeSheet) Bounds() spreadsheet.Bounds {
	bounds := spreadsheet.Bounds{}
	for reference := range s.cells {
		bounds.MaxRow = max(bounds.MaxRow, reference[0])
		bounds.MaxColumn = max(bounds.MaxColumn, reference[1])
	}
	return bounds
}

func (s *fakeSheet) Value(row int, column int) string {
	return s.cells[[2]int{row, column}]
}

func (s *fakeSheet) SetNumber(row int, column int, value float64) {
	s.cells[[2]int{row, column}] = strconv.FormatFloat(value, 'f', -1, 64)
}

func (s *fakeSheet) ClearValue(row int, column int) {
	delete(s.cells, [2]int{row, column})
}

func (s *fakeSheet) SetText(row int, column int, value string) {
	if value == "" {
		delete(s.cells, [2]int{row, column})
		return
	}
	s.cells[[2]int{row, column}] = value
}

func (s *fakeSheet) DeleteRows(rows []int) {
	removed := map[int]bool{}
	ordered := make([]int, 0, len(rows))
	for _, row := range rows {
		if removed[row] {
			continue
		}
		removed[row] = true
		ordered = append(ordered, row)
	}
	sort.Ints(ordered)

	moved := map[[2]int]string{}
	for reference, value := range s.cells {
		if removed[reference[0]] {
			continue
		}
		moved[[2]int{reference[0] - sort.SearchInts(ordered, reference[0]), reference[1]}] = value
	}
	s.cells = moved
}
