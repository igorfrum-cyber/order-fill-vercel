package grpcapi_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"order-fill/backend/pkg/grpcutil"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	jobsv1 "order-fill/backend/proto/gen/go/orderfill/jobs/v1"
	"order-fill/backend/services/job-service/internal/domain"
	"order-fill/backend/services/job-service/internal/queue"
	"order-fill/backend/services/job-service/internal/service/jobs"
	"order-fill/backend/services/job-service/internal/storage/memory"
	"order-fill/backend/services/job-service/internal/transport/grpcapi"
)

type fakeActors struct{ actor domain.Actor }

func (f fakeActors) Actor(context.Context, string) (domain.Actor, error) {
	return f.actor, nil
}

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

func TestCreateJobRejectsNonWorkbookAsInvalidArgument(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	files := fakeFiles{
		"src":   {ID: "src", Kind: string(domain.RoleSource), Name: "qa-not-excel.txt"},
		"blank": {ID: "blank", Kind: string(domain.RoleBlank), Name: "blank.xlsx"},
	}
	svc := jobs.New(memory.NewStore(), files, nil, queue.NewRedis(), func() time.Time { return now })
	actor := domain.Actor{UserID: "u1", CompanyID: "co", Role: domain.RolePurchaser}
	srv := grpcapi.NewServer(svc, fakeActors{actor: actor}, "")
	_, err := srv.CreateJob(t.Context(), &jobsv1.CreateJobRequest{
		Meta:         &commonv1.RequestMeta{ActorUserId: actor.UserID, CompanyId: actor.CompanyID},
		Type:         string(domain.TypeOrderFill),
		InputFileIds: []string{"src", "blank"},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code=%v err=%v want InvalidArgument", status.Code(err), err)
	}
	if !strings.Contains(status.Convert(err).Message(), "must be .xlsx or .xlsm") {
		t.Fatalf("message=%q", err)
	}
}

func TestGetJobIgnoresSpoofedPlatformAdminRole(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	svc := jobs.New(store, nil, nil, queue.NewRedis(), func() time.Time { return now })
	owner := domain.Actor{UserID: "u1", CompanyID: "co", Role: domain.RolePurchaser}
	job := domain.Job{
		ID: "job-1", Type: domain.TypeOrderFill, Status: domain.StatusQueued,
		OwnerUserID: owner.UserID, CompanyID: owner.CompanyID, CreatedAt: now, UpdatedAt: now,
	}
	if err := store.Create(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	other := domain.Job{
		ID: "job-2", Type: domain.TypeOrderFill, Status: domain.StatusQueued,
		OwnerUserID: "u2", CompanyID: "co", CreatedAt: now, UpdatedAt: now,
	}
	if err := store.Create(t.Context(), other); err != nil {
		t.Fatal(err)
	}
	srv := grpcapi.NewServer(svc, fakeActors{actor: owner}, "")
	ctx := grpcutil.WithActorRole(t.Context(), string(domain.RolePlatformAdmin))
	md, _ := metadata.FromOutgoingContext(ctx)
	ctx = metadata.NewIncomingContext(t.Context(), md)
	if _, err := srv.GetJob(ctx, &jobsv1.GetJobRequest{
		Meta: &commonv1.RequestMeta{ActorUserId: owner.UserID, CompanyId: owner.CompanyID}, JobId: other.ID,
	}); err == nil {
		t.Fatal("purchaser must not read another user's job via spoofed role")
	}
}

func TestCompleteJobRequiresWorkerToken(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	svc := jobs.New(store, nil, nil, queue.NewRedis(), func() time.Time { return now })
	job := domain.Job{
		ID: "job-1", Type: domain.TypeOrderFill, Status: domain.StatusProcessing,
		OwnerUserID: "u1", CompanyID: "co", CreatedAt: now, UpdatedAt: now,
	}
	if err := store.Create(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	srv := grpcapi.NewServer(svc, nil, "worker-secret-token")
	if _, err := srv.CompleteJob(t.Context(), &jobsv1.CompleteJobRequest{JobId: job.ID}); err == nil {
		t.Fatal("complete without worker token must fail")
	}
	ctx := grpcutil.WithWorkerToken(t.Context(), "worker-secret-token")
	md, _ := metadata.FromOutgoingContext(ctx)
	ctx = metadata.NewIncomingContext(t.Context(), md)
	if _, err := srv.CompleteJob(ctx, &jobsv1.CompleteJobRequest{JobId: job.ID}); err != nil {
		t.Fatal(err)
	}
}
