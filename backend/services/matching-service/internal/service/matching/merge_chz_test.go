package matching

import (
	"order-fill/backend/services/matching-service/internal/domain"
	"testing"
)

func TestLeviSsimeMainIdentity(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		items []domain.Item
		count int
	}{
		{"numeric alias", []domain.Item{{ID: "1", Article: "MT100", Name: "LEV Крем осветляющая 50 мл"}, {ID: "2", Article: "100", Name: "ЧЗ LeviSsime Крем отбеливающая 50 мл"}}, 1},
		{"English identity", []domain.Item{{ID: "1", Article: "100", Name: "LEV Уход первый 50 мл / WHITE PEARL CREAM"}, {ID: "2", Article: "100", Name: "ЧЗ Другой уход 50 мл / WHITE PEARL CREAM / описание"}}, 1},
		{"volume conflict", []domain.Item{{ID: "1", Article: "100", Name: "LEV Уход первый 50 мл / WHITE PEARL CREAM"}, {ID: "2", Article: "100", Name: "ЧЗ Другой уход 30 мл / WHITE PEARL CREAM"}}, 0},
		{"normal duplicate", []domain.Item{{ID: "1", Article: "MT100", Name: "LEV Крем осветляющая 50 мл"}, {ID: "2", Article: "100", Name: "LeviSsime Крем отбеливающая 50 мл"}}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := New().MergeChz(tc.items, Options{PrefixAliases: []string{"MT"}})
			if len(got) != tc.count {
				t.Fatalf("got %+v; count want %d", got, tc.count)
			}
		})
	}
}

func TestCombinedChzIsNormalTarget(t *testing.T) {
	t.Parallel()
	got := New().MergeChz([]domain.Item{{ID: "1", Article: "100", Name: "ЧЗ + LEV Крем осветляющая 50 мл"}, {ID: "2", Article: "100", Name: "ЧЗ LEV Крем осветляющая 50 мл"}}, Options{PrefixAliases: []string{"MT"}})
	if len(got) != 1 || got[0].TargetID != "1" {
		t.Fatalf("combined row lost: %+v", got)
	}
}

func TestLeviSsimeNonbreakingVolumeSpaceMatchesMain(t *testing.T) {
	t.Parallel()
	got := New().MergeChz([]domain.Item{{ID: "1", Article: "MT100", Name: "LEV Крем осветляющая 50\u00a0мл"}, {ID: "2", Article: "100", Name: "LeviSsime Крем отбеливающая 50 мл"}}, Options{PrefixAliases: []string{"MT"}})
	if len(got) != 1 {
		t.Fatalf("main merges NBSP volume identity: %+v", got)
	}
}
