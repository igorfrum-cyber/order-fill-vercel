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

func (f fakeFiles) Describe(_ context.Context, _ domain.Actor, ids []string) ([]domain.FileRef, error) {
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

type tenantFiles struct {
	files   fakeFiles
	company map[string]string
}

func (f tenantFiles) Describe(ctx context.Context, actor domain.Actor, ids []string) ([]domain.FileRef, error) {
	for _, id := range ids {
		if company, ok := f.company[id]; ok && company != actor.CompanyID {
			return nil, domain.ErrNotFound
		}
	}
	return f.files.Describe(ctx, actor, ids)
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
	if len(msgs) != 1 || msgs[0].Version != queue.Version || msgs[0].JobID != job.ID || msgs[0].Brand != "angiopharm" || msgs[0].CompanyID != "co" {
		t.Fatalf("messages=%v", msgs)
	}
}

type errPublisher struct{}

func (errPublisher) Publish(context.Context, queue.Message) error {
	return errors.New("xadd failed")
}

func TestCreateMarksFailedWhenEnqueueFails(t *testing.T) {
	t.Parallel()
	store := memory.NewStore()
	svc := jobs.New(store, orderFillFiles(), fakeCompanies{}, errPublisher{}, nil)
	actor := domain.Actor{UserID: "u1", CompanyID: "co", Role: domain.RolePurchaser}
	if _, err := svc.Create(t.Context(), actor, domain.TypeOrderFill, []string{"src", "blank"}, ""); err == nil {
		t.Fatal("expected enqueue error")
	}
	items, err := store.List(t.Context())
	if err != nil || len(items) != 1 || items[0].Status != domain.StatusFailed {
		t.Fatalf("items=%v err=%v", items, err)
	}
}

func createJobWithCompanyMode(t *testing.T, mode domain.MatchingMode) (domain.Job, *queue.Publisher) {
	t.Helper()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	pub := queue.NewRedis()
	svc := jobs.New(memory.NewStore(), orderFillFiles(), fakeCompanies{mode: mode}, pub, func() time.Time { return now })
	actor := domain.Actor{UserID: "u1", CompanyID: "co", Role: domain.RolePurchaser}
	job, err := svc.Create(t.Context(), actor, domain.TypeOrderFill, []string{"src", "blank"}, "angiopharm")
	if err != nil {
		t.Fatal(err)
	}
	return job, pub
}

func TestCreateSnapshotsCompanyMatchingModeIntoQueue(t *testing.T) {
	t.Parallel()
	job, publisher := createJobWithCompanyMode(t, domain.MatchingModeSmart)
	if job.MatchingMode != domain.MatchingModeSmart {
		t.Fatalf("job mode = %q", job.MatchingMode)
	}
	if got := publisher.Messages()[0].MatchingMode; got != "smart" {
		t.Fatalf("queue mode = %q", got)
	}
}

func TestCreateRejectsForeignCompanyFile(t *testing.T) {
	t.Parallel()
	files := tenantFiles{
		files:   orderFillFiles(),
		company: map[string]string{"src": "other"},
	}
	svc := jobs.New(memory.NewStore(), files, fakeCompanies{}, queue.NewRedis(), nil)
	actor := domain.Actor{UserID: "u1", CompanyID: "co", Role: domain.RolePurchaser}
	_, err := svc.Create(t.Context(), actor, domain.TypeOrderFill, []string{"src", "blank"}, "")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v", err)
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

func TestCreateRejectsPlatformAdmin(t *testing.T) {
	t.Parallel()
	svc := jobs.New(memory.NewStore(), orderFillFiles(), fakeCompanies{}, queue.NewRedis(), nil)
	_, err := svc.Create(t.Context(), domain.Actor{UserID: "p1", Role: domain.RolePlatformAdmin}, domain.TypeOrderFill, []string{"src", "blank"}, "")
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

func TestSubmitEditsRejectsConcurrentFinalize(t *testing.T) {
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
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := svc.SubmitEdits(t.Context(), actor, job.ID, nil)
			errs <- err
		}()
	}
	var conflict, ok int
	for range 2 {
		err := <-errs
		if err == nil {
			ok++
			continue
		}
		if errors.Is(err, domain.ErrConflict) {
			conflict++
			continue
		}
		t.Fatalf("unexpected: %v", err)
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("ok=%d conflict=%d messages=%d", ok, conflict, len(pub.Messages()))
	}
	if len(pub.Messages()) != 2 {
		t.Fatalf("messages=%d", len(pub.Messages()))
	}
}
