package calculation_test

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"order-fill/backend/services/calculation-service/internal/domain"
	"order-fill/backend/services/calculation-service/internal/service/calculation"
)

func TestBudgetMatchesPinnedMainOracle(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("testdata/budget_main_1e2baa1.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Cases []struct {
			Name     string
			Input    []domain.BudgetRow
			Target   float64
			Options  domain.BudgetOptions
			Expected struct {
				Quantities    []float64
				Before, Total float64
				Complete      bool
				Reason        string
				LineSteps     []domain.CompletionStep
			}
		}
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if len(corpus.Cases) != 80 {
		t.Fatalf("incomplete oracle corpus: %d", len(corpus.Cases))
	}
	for _, tc := range corpus.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			for _, mode := range []string{domain.ChristinaProffStandard, domain.ChristinaProffFast} {
				opts := tc.Options
				opts.ChristinaProffMode = mode
				got, err := calculation.PlanBudget(tc.Input, tc.Target, opts)
				if err != nil {
					t.Fatal(err)
				}
				qty := make([]float64, len(got.Rows))
				for i, row := range got.Rows {
					qty[i] = row.Quantity
				}
				if !slices.Equal(qty, tc.Expected.Quantities) || got.Before != tc.Expected.Before || got.Total != tc.Expected.Total || got.Complete != tc.Expected.Complete || got.Reason != tc.Expected.Reason || !slices.Equal(got.LineSteps, tc.Expected.LineSteps) {
					t.Fatalf("%s: qty=%v before=%v total=%v complete=%v reason=%q steps=%v; main=%+v", mode, qty, got.Before, got.Total, got.Complete, got.Reason, got.LineSteps, tc.Expected)
				}
			}
		})
	}
}
