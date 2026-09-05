package north

import "testing"

func TestCityFromFileName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, key string
		ok        bool
	}{
		{"Сургут.xlsx", "surgut", true},
		{"blank-urengoy-HOME.xlsx", "urengoy", true},
		{"Тюмень остатки.xlsx", "tyumen", true},
		{"Нижневартовск.xlsx", "nizhnevartovsk", true},
		{"blank.xlsx", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			key, _, ok := CityFromFileName(tc.name)
			if ok != tc.ok || key != tc.key {
				t.Fatalf("key=%s ok=%v want %s/%v", key, ok, tc.key, tc.ok)
			}
		})
	}
}
