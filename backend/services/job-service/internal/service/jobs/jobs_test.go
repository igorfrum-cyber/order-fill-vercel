package jobs_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"order-fill/backend/services/job-service/internal/domain"
	"order-fill/backend/services/job-service/internal/queue"
	"order-fill/backend/services/job-service/internal/service/jobs"
	"order-fill/backend/services/job-service/internal/storage/memory"
)

type fakeFiles map[string]domain.FileRef

func (f fakeFiles) Describe(_ context.Context, ids []string) ([]domain.FileRef, error) {
	out := make([]domain.FileRef, 0, len(ids))
	for _, id := range ids {
		file, ok := f[id]
		if !ok {
			return nil, domain.ErrNotFound
		}
		out = append(out, file)
	}
	return out, nil
}

type fakeCompanies struct{ mode domain.MatchingMode }

func (f fakeCompanies) MatchingMode(context.Context, domain.Actor) (domain.MatchingMode, error) {
	return f.mode, nil
}

func orderFillFiles() fakeFiles {
	return fakeFiles{
		"src":   {ID: "src", Kind: string(domain.RoleSource), Name: "source.xlsx"},
		"blank": {ID: "blank", Kind: string(domain.RoleBlank), Name: "blank.xlsx"},
	}
}

func TestCreatePublishesOneVersionedMessage(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	pub := queue.NewRedis()
	svc := jobs.New(memory.NewStore(), orderFillFiles(), fakeCompanies{mode: domain.MatchingModeSmart}, pub, func() time.Time { return now })
	actor := domain.Actor{UserID: "u1", CompanyID: "co", Role: domain.RolePurchaser}
	job, err := svc.Create(t.Context(), actor, domain.TypeOrderFill, []string{"src", "blank"}, "angiopharm")
	if err != nil {
		t.Fatal(err)
	}
	if job.MatchingMode != domain.MatchingModeSmart {
		t.Fatalf("mode=%s", job.MatchingMode)
	}
	msgs := pub.Messages()
	if len(msgs) != 1 || msgs[0].Version != queue.Version || msgs[0].JobID != job.ID || msgs[0].Brand != "angiopharm" {
		t.Fatalf("messages=%v", msgs)
	}
}

func TestCreateRequiresOwner(t *testing.T) {
	t.Parallel()
	svc := jobs.New(memory.NewStore(), orderFillFiles(), fakeCompanies{}, queue.NewRedis(), nil)
	_, err := svc.Create(t.Context(), domain.Actor{Role: domain.RolePurchaser}, domain.TypeOrderFill, []string{"src", "blank"}, "")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestListHidesOtherPurchaserJobs(t *testing.T) {
	t.Parallel()
	store := memory.NewStore()
	svc := jobs.New(store, orderFillFiles(), fakeCompanies{}, queue.NewRedis(), nil)
	buyer := domain.Actor{UserID: "u1", CompanyID: "co", Role: domain.RolePurchaser}
	if _, err := svc.Create(t.Context(), buyer, domain.TypeOrderFill, []string{"src", "blank"}, ""); err != nil {
		t.Fatal(err)
	}
	other := domain.Actor{UserID: "u2", CompanyID: "co", Role: domain.RolePurchaser}
	items, err := svc.List(t.Context(), other)
	if err != nil || len(items) != 0 {
		t.Fatalf("other saw %v err=%v", items, err)
	}
	admin := domain.Actor{UserID: "a1", CompanyID: "co", Role: domain.RoleCompanyAdmin}
	items, err = svc.List(t.Context(), admin)
	if err != nil || len(items) != 1 {
		t.Fatalf("admin saw %d err=%v", len(items), err)
	}
}

func TestCompleteRecordsReportAndFiles(t *testing.T) {
	t.Parallel()
	store := memory.NewStore()
	svc := jobs.New(store, orderFillFiles(), fakeCompanies{}, queue.NewRedis(), nil)
	actor := domain.Actor{UserID: "u1", CompanyID: "co", Role: domain.RolePurchaser}
	job, err := svc.Create(t.Context(), actor, domain.TypeOrderFill, []string{"src", "blank"}, "")
	if err != nil {
		t.Fatal(err)
	}
	report := domain.Report{Summary: domain.ReportSummary{ToOrder: 1}, Rows: []domain.ReportRow{{ID: "r1", Category: domain.CategoryToOrder}}}
	files := []domain.FileRef{{ID: "out", Kind: "output", Name: "filled.xlsx"}}
	if err := svc.Complete(t.Context(), job.ID, report, files); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(t.Context(), actor, job.ID)
	if err != nil || got.Status != domain.StatusNeedsReview || len(got.OutputFiles) != 1 {
		t.Fatalf("job=%+v err=%v", got, err)
	}
	saved, err := svc.GetReport(t.Context(), actor, job.ID)
	if err != nil || saved.Summary.ToOrder != 1 {
		t.Fatalf("report=%+v err=%v", saved, err)
	}
}

func TestCompletedJobAcceptsEdits(t *testing.T) {
	t.Parallel()
	store := memory.NewStore()
	pub := queue.NewRedis()
	svc := jobs.New(store, orderFillFiles(), fakeCompanies{}, pub, nil)
	actor := domain.Actor{UserID: "u1", CompanyID: "co", Role: domain.RolePurchaser}
	job, err := svc.Create(t.Context(), actor, domain.TypeOrderFill, []string{"src", "blank"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Complete(t.Context(), job.ID, domain.Report{}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SubmitEdits(t.Context(), actor, job.ID, nil); err != nil {
		t.Fatal(err)
	}
	if len(pub.Messages()) != 2 {
		t.Fatalf("messages=%d", len(pub.Messages()))
	}
}
