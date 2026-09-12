package grpcapi_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"

	"order-fill/backend/pkg/grpcutil"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	filesv1 "order-fill/backend/proto/gen/go/orderfill/files/v1"
	"order-fill/backend/services/file-service/internal/clients/identity"
	"order-fill/backend/services/file-service/internal/service/files"
	"order-fill/backend/services/file-service/internal/storage/memory"
	"order-fill/backend/services/file-service/internal/storage/objectstore"
	"order-fill/backend/services/file-service/internal/transport/grpcapi"
)

type fakeActors struct{ actor identity.Actor }

func (f fakeActors) Actor(context.Context, string) (identity.Actor, error) {
	return f.actor, nil
}

func workerCtx(t *testing.T, token string) context.Context {
	t.Helper()
	ctx := grpcutil.WithWorkerToken(t.Context(), token)
	md, _ := metadata.FromOutgoingContext(ctx)
	return metadata.NewIncomingContext(t.Context(), md)
}

func TestGetObjectDeniesOtherCompany(t *testing.T) {
	t.Parallel()
	svc := files.New(objectstore.NewS3(), memory.NewMeta())
	owner := identity.Actor{UserID: "u1", CompanyID: "co-1", Role: "purchaser"}
	srv := grpcapi.NewServer(svc, fakeActors{actor: owner}, "worker-secret-token")
	put, err := srv.PutObject(t.Context(), &filesv1.PutObjectRequest{
		Meta: &commonv1.RequestMeta{ActorUserId: "u1", CompanyId: "co-1"}, Name: "a.xlsx", Body: []byte("xlsx"),
	})
	if err != nil {
		t.Fatal(err)
	}
	other := grpcapi.NewServer(svc, fakeActors{actor: identity.Actor{UserID: "u2", CompanyID: "co-2", Role: "purchaser"}}, "worker-secret-token")
	if _, err := other.GetObject(t.Context(), &filesv1.GetObjectRequest{
		Meta: &commonv1.RequestMeta{ActorUserId: "u2"}, Id: put.GetObject().GetId(),
	}); err == nil {
		t.Fatal("other company must not read the object")
	}
	if _, err := srv.GetObject(workerCtx(t, "worker-secret-token"), &filesv1.GetObjectRequest{Id: put.GetObject().GetId()}); err != nil {
		t.Fatal(err)
	}
}

func TestGetObjectAllowsPublicLogoWithoutActor(t *testing.T) {
	t.Parallel()
	svc := files.New(objectstore.NewS3(), memory.NewMeta())
	admin := identity.Actor{UserID: "a1", CompanyID: "co-1", Role: "company_admin"}
	srv := grpcapi.NewServer(svc, fakeActors{actor: admin}, "worker-secret-token")
	if _, err := srv.PutObject(t.Context(), &filesv1.PutObjectRequest{
		Meta: &commonv1.RequestMeta{ActorUserId: "a1", CompanyId: "co-1"},
		Key:  "companies/co-1/logo", Name: "logo", ContentType: "image/png", Body: []byte("png"),
	}); err != nil {
		t.Fatal(err)
	}
	got, err := srv.GetObject(t.Context(), &filesv1.GetObjectRequest{Key: "companies/co-1/logo"})
	if err != nil || string(got.GetBody()) != "png" {
		t.Fatalf("public logo: %v %+v", err, got)
	}
}
