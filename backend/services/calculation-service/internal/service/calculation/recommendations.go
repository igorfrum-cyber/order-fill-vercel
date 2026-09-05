package calculation

import (
	"math"
	"sort"
	"strings"

	"order-fill/backend/services/calculation-service/internal/domain"
)

const defaultDeliveryWeeks = 4.0

type Service struct{}

func New() *Service { return &Service{} }

func (s *Service) Recommend(brand string, rows []domain.OrderRow) []domain.OrderRow {
	return s.RecommendWithWeeks(brand, rows, defaultDeliveryWeeks)
}

func (s *Service) RecommendWithWeeks(brand string, rows []domain.OrderRow, deliveryWeeks float64) []domain.OrderRow {
	safeWeeks := math.Max(1, deliveryWeeks)
	deliveryCoefficient := 0.25 * safeWeeks
	totalRevenue := 0.0
	for _, row := range rows {
		totalRevenue += row.Revenue
	}
	ranked := append([]domain.OrderRow{}, rows...)
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].Revenue > ranked[j].Revenue })
	type metric struct {
		category string
	}
	metrics := map[string]metric{}
	cumulative := 0.0
	for _, row := range ranked {
		cumulative += row.Revenue
		percent := 100.0
		if totalRevenue > 0 {
			percent = cumulative / totalRevenue * 100
		}
		metrics[row.ID] = metric{category: categoryFromCumulative(percent)}
	}
	out := make([]domain.OrderRow, len(rows))
	for i, row := range rows {
		m := metrics[row.ID]
		if m.category == "" {
			m.category = "C"
		}
		novelty := newProductCalculation(row.MonthlySales, safeWeeks)
		monthlyNeed := calculateTargetNew(row.MonthlySales)
		category := m.category
		targetStock := 0.0
		if novelty != nil {
			category = m.category + "/New"
			targetStock = novelty.targetStock
		} else {
			targetStock = monthlyNeed*categoryCoefficient(category, brand) + monthlyNeed*deliveryCoefficient
		}
		recommended := math.Max(0, targetStock-row.Stock-row.InTransit)
		row.ABCCategory = category
		row.TargetStock = roundTo2(targetStock)
		row.Recommended = roundTo2(recommended)
		out[i] = row
	}
	return out
}

type noveltyCalculation struct {
	targetStock float64
}

func newProductCalculation(values []float64, deliveryWeeks float64) *noveltyCalculation {
	suffix := 0
	for index := len(values) - 1; index >= 0; index-- {
		if values[index] <= 0 {
			break
		}
		suffix++
	}
	if suffix < 1 || suffix > 3 {
		return nil
	}
	firstNovelty := len(values) - suffix
	for _, value := range values[:firstNovelty] {
		if value > 0 {
			return nil
		}
	}
	maxMonth := 0.0
	for _, value := range values[firstNovelty:] {
		maxMonth = math.Max(maxMonth, value)
	}
	if maxMonth <= 0 {
		return nil
	}
	deliveryCoefficient := 0.25 * math.Max(1, deliveryWeeks)
	return &noveltyCalculation{targetStock: maxMonth*1.5 + maxMonth*deliveryCoefficient}
}

func calculateTargetNew(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	threshold := total / 24
	stable := true
	lowest := math.Inf(1)
	for _, value := range values {
		if value <= 0 {
			stable = false
		}
		lowest = math.Min(lowest, value)
	}
	if stable && lowest > threshold {
		average := total / float64(len(values))
		secondMonth := 0.0
		if len(values) > 1 {
			secondMonth = values[1]
		}
		firstThree := 0.0
		for index := range min(3, len(values)) {
			firstThree += values[index]
		}
		return (average + secondMonth + firstThree/3) / 3
	}
	filteredTotal := 0.0
	filteredCount := 0
	for _, value := range values {
		if value > 0 && value > threshold {
			filteredTotal += value
			filteredCount++
		}
	}
	if filteredCount == 0 {
		return 0
	}
	return filteredTotal / float64(filteredCount)
}

func categoryFromCumulative(percent float64) string {
	switch {
	case percent <= 50:
		return "A+"
	case percent <= 80:
		return "A"
	case percent <= 95:
		return "B"
	default:
		return "C"
	}
}

func categoryCoefficient(category, brand string) float64 {
	base, _ := strings.CutSuffix(category, "/New")
	if brand == "novacutan" {
		if base == "C" {
			return 1.5
		}
		if base == "A+" || base == "A" || base == "B" {
			return 2
		}
	}
	switch base {
	case "A+":
		return 2
	case "A":
		return 1.75
	case "B":
		return 1.5
	default:
		return 1
	}
}

func roundTo2(value float64) float64 {
	return math.Round(value*100) / 100
}

func CategoryFromCumulative(percent float64) string { return categoryFromCumulative(percent) }
func DeliveryCoefficient(weeks float64) float64     { return 0.25 * math.Max(1, weeks) }
func CategoryCoefficient(category, brand string) float64 {
	return categoryCoefficient(category, brand)
}
