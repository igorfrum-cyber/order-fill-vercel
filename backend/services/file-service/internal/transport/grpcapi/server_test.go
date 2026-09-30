package grpcapi_test

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
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

func TestCreateAndFinalizeUpload(t *testing.T) {
	t.Parallel()

	service := files.New(objectstore.NewS3(), memory.NewMeta())
	actor := identity.Actor{UserID: "u1", CompanyID: "co-1", Role: "company_admin"}
	server := grpcapi.NewServer(service, fakeActors{actor: actor}, "worker-secret-token")
	created, err := server.CreateUpload(t.Context(), &filesv1.CreateUploadRequest{
		Meta: &commonv1.RequestMeta{ActorUserId: actor.UserID}, Name: "input.xlsx", ContentType: "application/xlsx",
	})
	if err != nil || created.GetUploadId() == "" {
		t.Fatalf("CreateUpload() = %+v, %v", created, err)
	}
	finalized, err := server.FinalizeUpload(t.Context(), &filesv1.FinalizeUploadRequest{
		Meta:     &commonv1.RequestMeta{ActorUserId: actor.UserID, CompanyId: actor.CompanyID},
		UploadId: created.GetUploadId(), Body: []byte("workbook"),
	})
	if err != nil || finalized.GetObject().GetName() != "input.xlsx" {
		t.Fatalf("FinalizeUpload() = %+v, %v", finalized, err)
	}
	got, err := server.GetObject(t.Context(), &filesv1.GetObjectRequest{
		Meta: &commonv1.RequestMeta{ActorUserId: actor.UserID}, Id: finalized.GetObject().GetId(),
	})
	if err != nil || string(got.GetBody()) != "workbook" {
		t.Fatalf("GetObject(finalized upload) = %+v, %v", got, err)
	}
}

func TestWorkerUploadRequiresCompanyAndAuthorizedWorker(t *testing.T) {
	t.Parallel()

	service := files.New(objectstore.NewS3(), memory.NewMeta())
	server := grpcapi.NewServer(service, nil, "worker-secret-token")
	if _, err := server.FinalizeUpload(workerCtx(t, "worker-secret-token"), &filesv1.FinalizeUploadRequest{UploadId: "missing", Body: []byte("x")}); err == nil {
		t.Fatal("worker finalization without company metadata must fail")
	}
	if _, err := server.CreateUpload(t.Context(), &filesv1.CreateUploadRequest{Meta: &commonv1.RequestMeta{ActorUserId: "u1"}, Name: "input.xlsx"}); err == nil {
		t.Fatal("unverified actor must not create upload")
	}
}

func TestCreateArchiveCombinesCompanyObjects(t *testing.T) {
	t.Parallel()

	service := files.New(objectstore.NewS3(), memory.NewMeta())
	actor := identity.Actor{UserID: "u1", CompanyID: "co-1", Role: "company_admin"}
	server := grpcapi.NewServer(service, fakeActors{actor: actor}, "worker-secret-token")
	ctx := t.Context()
	meta := &commonv1.RequestMeta{ActorUserId: actor.UserID, CompanyId: actor.CompanyID}
	first, err := server.PutObject(ctx, &filesv1.PutObjectRequest{Meta: meta, Key: "input/one", Name: "one.xlsx", Body: []byte("first")})
	if err != nil {
		t.Fatal(err)
	}
	second, err := server.PutObject(ctx, &filesv1.PutObjectRequest{Meta: meta, Key: "input/two", Name: "two.xlsx", Body: []byte("second")})
	if err != nil {
		t.Fatal(err)
	}
	archive, err := server.CreateArchive(ctx, &filesv1.CreateArchiveRequest{
		Meta: meta, ObjectIds: []string{first.GetObject().GetId(), second.GetObject().GetKey()}, Name: "orders.zip",
	})
	if err != nil || archive.GetObject().GetContentType() != "application/zip" {
		t.Fatalf("CreateArchive() = %+v, %v", archive, err)
	}
	response, err := server.GetObject(ctx, &filesv1.GetObjectRequest{Meta: meta, Id: archive.GetObject().GetId()})
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(response.GetBody()), int64(len(response.GetBody())))
	if err != nil || len(reader.File) != 2 {
		t.Fatalf("archive entries = %v, err = %v", reader, err)
	}
	contents := make(map[string]string)
	for _, entry := range reader.File {
		file, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := io.ReadAll(file)
		_ = file.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		contents[entry.Name] = string(body)
	}
	if contents["one.xlsx"] != "first" || contents["two.xlsx"] != "second" {
		t.Fatalf("archive contents = %v", contents)
	}
}

func TestCreateArchiveRejectsEmptyAndUnauthorizedRequests(t *testing.T) {
	t.Parallel()

	service := files.New(objectstore.NewS3(), memory.NewMeta())
	admin := identity.Actor{UserID: "a1", Role: "platform_admin"}
	server := grpcapi.NewServer(service, fakeActors{actor: admin}, "worker-secret-token")
	if _, err := server.CreateArchive(t.Context(), &filesv1.CreateArchiveRequest{Meta: &commonv1.RequestMeta{ActorUserId: admin.UserID}}); err == nil {
		t.Fatal("archive without object refs must fail")
	}
	if _, err := server.CreateArchive(workerCtx(t, "worker-secret-token"), &filesv1.CreateArchiveRequest{ObjectIds: []string{"id"}}); err == nil {
		t.Fatal("worker archive without company must fail")
	}
}
