package job

import (
	"fmt"
	"strings"
)

// Report is the reviewable result of a processed job.
type Report struct {
	JobID   string
	Summary Summary
	Rows    []ReportRow
}

// Summary carries the aggregate numbers and the period metadata the UI shows
// above the table.
type Summary struct {
	Brand                  string
	OrderMonthLabel        string
	AdjustmentLabel        string
	ActualMainPeriod       string
	ActualPreviousPeriod   string
	SourceCity             string
	CityRule               string
	DeliveryWeeks          float64
	Filled                 int
	LeftBlank              int
	Suspicious             int
	Unmatched              int
	Duplicates             int
	NotInBlank             int
	BlankDuplicateArticles int
	SourceItems            int
	SourceArticles         int
	SourceSheet            string
	SourceHeaderRow        int
	BlankSheet             string
	BlankHeaderRow         int
}

// ReportRow is one reviewable position of the supplier blank.
type ReportRow struct {
	Key                 string
	Status              string
	BlankID             string
	BlankLabel          string
	BlankRow            int
	BlankQuantityColumn int
	BlankArticle        string
	BlankName           string
	BlankUnit           string
	BlankBoxSize        string
	SourceRow           *int
	SourceArticle       string
	SourceName          string
	HasOrderedFact      bool
	OrderedFact         *float64
	SourceComment       string
	Stock               string
	InTransit           string
	Recommended         *float64
	Rounded             *int
	BaseRounded         *int
	Inserted            *float64
	AutoComment         string
	AdjustmentLabel     string
	BoxAdjusted         bool
	Duplicate           bool
	DuplicateCandidates []DuplicateCandidate
	Editable            bool
	Similarity          float64
}

// DuplicateCandidate is a competing source row for the same article.
type DuplicateCandidate struct {
	SourceRow     int
	SourceArticle string
	SourceName    string
	Recommended   float64
	Rounded       int
	Stock         string
	InTransit     string
}

// ManualEdit is a reviewer correction for a single report row.
type ManualEdit struct {
	Key     string
	Value   string
	Comment string
}

// ValidateEdits enforces the shape of an edit batch. Business validation of the
// values themselves belongs to document-service, which owns the workbook rules.
func ValidateEdits(edits []ManualEdit) error {
	seen := make(map[string]bool, len(edits))
	for _, edit := range edits {
		key := strings.TrimSpace(edit.Key)
		if key == "" {
			return fmt.Errorf("%w: edit key is required", ErrInvalid)
		}
		if seen[key] {
			return fmt.Errorf("%w: duplicate edit for key %q", ErrInvalid, key)
		}
		seen[key] = true
	}
	return nil
}
