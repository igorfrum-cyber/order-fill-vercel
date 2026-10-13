package matching

import (
	"encoding/json"
	"math"
	"os"
	"slices"
	"testing"

	"order-fill/backend/services/matching-service/internal/domain"
)

func TestSelectorsMatchPinnedMainOracle(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("testdata/matching_main_1e2baa1.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Cases []struct {
			Name, Kind, BlankName, BlankUnit string
			Items                            []domain.Item
			Expected                         struct {
				ID       string
				Score    float64
				Category domain.Category
			}
		}
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if len(corpus.Cases) != 46 {
		t.Fatalf("incomplete oracle corpus: %d", len(corpus.Cases))
	}
	for _, tc := range corpus.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			if tc.Kind == "candidate" || tc.Kind == "fallback" {
				var got scored
				var ok bool
				if tc.Kind == "candidate" {
					got, ok = chooseCandidate(tc.Items, tc.BlankName, tc.BlankUnit)
				} else {
					got, ok = chooseNameFallback(tc.Items, tc.BlankName, tc.BlankUnit)
				}
				if ok != (tc.Expected.ID != "") || got.item.ID != tc.Expected.ID || math.Abs(got.score-tc.Expected.Score) > 1e-12 {
					t.Fatalf("selector: got id=%s ok=%v score=%v; main=%+v", got.item.ID, ok, got.score, tc.Expected)
				}
			}
			opts := Options{Mode: domain.ModeStandard, Brand: "angiopharm"}
			items := slices.Clone(tc.Items)
			blank := domain.Item{ID: "blank", Name: tc.BlankName, Volume: tc.BlankUnit}
			switch tc.Kind {
			case "candidate":
				blank.Article = "X"
				for i := range items {
					items[i].Article = "X"
				}
			case "sothys", "novacutan":
				opts.Brand = tc.Kind
			}
			got := New().Match([]domain.Item{blank}, items, opts)[0]
			wantScore := tc.Expected.Score
			if tc.Expected.ID == "" {
				wantScore = 0
			}
			if got.SourceItemID != tc.Expected.ID || math.Abs(got.Score-wantScore) > 1e-12 || got.Category != tc.Expected.Category {
				t.Fatalf("standard: got %+v; main=%+v", got, tc.Expected)
			}
		})
	}
}
