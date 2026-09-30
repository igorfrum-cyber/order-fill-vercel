package memory

import (
	"sync"
	"testing"

	"order-fill/backend/services/audit-service/internal/domain"
)

func TestStoreRecordAndList(t *testing.T) {
	t.Parallel()

	store := New()
	if got, err := store.List(t.Context(), "company-a"); err != nil || len(got) != 0 {
		t.Fatalf("empty List() = %v, %v", got, err)
	}
	events := []domain.Event{
		{ID: "a1", CompanyID: "company-a"},
		{ID: "b1", CompanyID: "company-b"},
		{ID: "a2", CompanyID: "company-a"},
	}
	for _, event := range events {
		if err := store.Record(t.Context(), event); err != nil {
			t.Fatal(err)
		}
	}

	all, err := store.List(t.Context(), "")
	if err != nil || len(all) != len(events) {
		t.Fatalf("all events = %v, %v", all, err)
	}
	company, err := store.List(t.Context(), "company-a")
	if err != nil || len(company) != 2 || company[0].ID != "a1" || company[1].ID != "a2" {
		t.Fatalf("company events = %v, %v", company, err)
	}
	company[0].ID = "changed"
	again, _ := store.List(t.Context(), "company-a")
	if again[0].ID != "a1" {
		t.Fatalf("mutating returned list changed stored event: %+v", again[0])
	}
}

func TestStoreConcurrentRecords(t *testing.T) {
	t.Parallel()

	store := New()
	var records sync.WaitGroup
	for range 24 {
		records.Go(func() {
			if err := store.Record(t.Context(), domain.Event{CompanyID: "company-a"}); err != nil {
				t.Error(err)
			}
		})
	}
	records.Wait()
	got, err := store.List(t.Context(), "company-a")
	if err != nil || len(got) != 24 {
		t.Fatalf("concurrent records = %d, %v", len(got), err)
	}
}
