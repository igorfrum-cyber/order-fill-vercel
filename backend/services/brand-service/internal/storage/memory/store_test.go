package memory

import (
	"reflect"
	"testing"

	"order-fill/backend/services/brand-service/internal/domain"
)

func TestStoreGetAndSave(t *testing.T) {
	t.Parallel()

	store := New()
	if _, ok, err := store.Get(t.Context(), "klapp"); err != nil || ok {
		t.Fatalf("Get(missing) = %v, %t", err, ok)
	}
	policy := domain.Policy{Key: "klapp", Label: "KLAPP", Multiple: 3}
	if err := store.Save(t.Context(), policy, "actor"); err != nil {
		t.Fatal(err)
	}
	got, ok, err := store.Get(t.Context(), "klapp")
	if err != nil || !ok || !reflect.DeepEqual(got, policy) {
		t.Fatalf("Get(saved) = %+v, %t, %v; want %+v", got, ok, err, policy)
	}

	updated := domain.Policy{Key: "klapp", Label: "KLAPP Updated", Multiple: 6}
	if err := store.Save(t.Context(), updated, "actor"); err != nil {
		t.Fatal(err)
	}
	got, ok, err = store.Get(t.Context(), "klapp")
	if err != nil || !ok || !reflect.DeepEqual(got, updated) {
		t.Fatalf("Get(updated) = %+v, %t, %v; want %+v", got, ok, err, updated)
	}
}
