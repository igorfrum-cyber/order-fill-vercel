package calculation_test

import (
	"testing"

	"order-fill/backend/services/calculation-service/internal/domain"
	"order-fill/backend/services/calculation-service/internal/service/calculation"
)

func TestDefaultNorthActualSupplierOrder(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		brand    string
		pos      domain.NorthSupplierPosition
		need     float64
		expected float64
	}{
		{"angiopharm box", "angiopharm", domain.NorthSupplierPosition{BlankBoxSize: 6, HasBoxSize: true}, 4.75, 6},
		{"levissime box", "levissime", domain.NorthSupplierPosition{BlankBoxSize: 4, HasBoxSize: true}, 4.75, 8},
		{"skin_synergy", "skin_synergy", domain.NorthSupplierPosition{}, 0.22, 1},
		{"sothys", "sothys", domain.NorthSupplierPosition{}, 1.25, 2},
		{"christina", "christina", domain.NorthSupplierPosition{}, 4.75, 6},
		{"klapp 10", "klapp", domain.NorthSupplierPosition{}, 10, 9},
		{"klapp 11", "klapp", domain.NorthSupplierPosition{}, 11, 12},
		{"novacutan 104", "novacutan", domain.NorthSupplierPosition{NovacutanMinimum: 100}, 104, 100},
		{"novacutan 105", "novacutan", domain.NorthSupplierPosition{NovacutanMinimum: 100}, 105, 110},
		{"novacutan mask", "novacutan", domain.NorthSupplierPosition{NovacutanMinimum: 10, SupplierUnitSize: 5}, 12, 12},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := calculation.DefaultNorthActualSupplierOrder(tc.brand, "", tc.pos, tc.need)
			if got != tc.expected {
				t.Fatalf("got %v want %v", got, tc.expected)
			}
		})
	}
}

func TestChristinaNorthSupplierQuantity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		need     float64
		expected float64
		ok       bool
	}{
		{"zero", 0, 0, false},
		{"fraction", 0.22, 3, true},
		{"one box", 1.25, 3, true},
		{"below multiple", 2, 3, true},
		{"exact multiple", 3, 3, true},
		{"next multiple", 4.75, 6, true},
		{"second exact multiple", 6, 6, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := calculation.ChristinaNorthSupplierQuantity(tc.need)
			if ok != tc.ok || got != tc.expected {
				t.Fatalf("need %v: got %v %v want %v %v", tc.need, got, ok, tc.expected, tc.ok)
			}
		})
	}
}

func TestDefaultNorthActualHomeVariantUsesChristina(t *testing.T) {
	t.Parallel()
	got := calculation.DefaultNorthActualSupplierOrder("angiopharm", "home", domain.NorthSupplierPosition{BlankBoxSize: 6, HasBoxSize: true}, 2)
	if got != 3 {
		t.Fatalf("home variant must use *3, got %v", got)
	}
}
