package calculation_test

import (
	"testing"

	"order-fill/backend/services/calculation-service/internal/domain"
	"order-fill/backend/services/calculation-service/internal/service/calculation"
)

// Expectations come from main 1e2baa1 recalculateSourceTable's Number(toFixed(2)).
// Multiplying by 100 before rounding changes the result for these binary floats.
func TestRecommendationsMatchMainDecimalRounding(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		sale, want float64
	}{
		{"binary below half", 1.07, 2.67},
		{"multiplication crosses half", 1.13, 2.82},
		{"exact half rounds up", 1.05, 2.63},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := calculation.New().RecommendWithWeeks("angiopharm", []domain.OrderRow{{
				ID: "novel", Revenue: 10, MonthlySales: []float64{0, 0, 0, tc.sale},
			}}, 4)
			if out[0].TargetStock != tc.want || out[0].Recommended != tc.want || out[0].ABCCategory != "C/New" {
				t.Fatalf("got %+v; want target/recommended=%v, category=C/New", out[0], tc.want)
			}
		})
	}
}

func TestUrengoyCategoryMatchesMainNormalization(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, category, brand string
		want                  float64
	}{
		{"Cyrillic A", "А", "angiopharm", 37.5},
		{"lowercase and spaces", " a + ", "angiopharm", 40},
		{"Cyrillic B", "В", "angiopharm", 35},
		{"Novacutan Cyrillic C", "с", "novacutan", 35},
		{"novel category uses main fallback", "A/New", "angiopharm", 30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := calculation.New().RecommendUrengoy(tc.brand, []domain.OrderRow{{
				ID: "urengoy", ABCCategory: tc.category, MonthlySales: []float64{10, 4},
			}}, 4)
			if out[0].Recommended != tc.want {
				t.Fatalf("recommended=%v; main=%v for category %q", out[0].Recommended, tc.want, tc.category)
			}
		})
	}
}
