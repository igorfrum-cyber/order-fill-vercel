package grpcapi_test

import (
	"context"
	"testing"
	"time"

	auditv1 "order-fill/backend/proto/gen/go/orderfill/audit/v1"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	"order-fill/backend/services/audit-service/internal/clients/identity"
	"order-fill/backend/services/audit-service/internal/service/audit"
	"order-fill/backend/services/audit-service/internal/storage/memory"
	"order-fill/backend/services/audit-service/internal/transport/grpcapi"
)

type fakeActors struct{ actor identity.Actor }

func (f fakeActors) Actor(context.Context, string) (identity.Actor, error) {
	return f.actor, nil
}

func TestListEventsEmptyCompanyRequiresAdmin(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	svc := audit.New(memory.New(), func() time.Time { return now })
	if _, err := svc.Record(t.Context(), "login", "u1", "co-1", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Record(t.Context(), "login", "u2", "co-2", "", ""); err != nil {
		t.Fatal(err)
	}
	open := grpcapi.NewServer(svc, nil)
	if _, err := open.ListEvents(t.Context(), &auditv1.ListEventsRequest{}); err == nil {
		t.Fatal("empty company without identity must fail")
	}
	buyer := grpcapi.NewServer(svc, fakeActors{actor: identity.Actor{UserID: "u1", CompanyID: "co-1", Role: "purchaser"}})
	got, err := buyer.ListEvents(t.Context(), &auditv1.ListEventsRequest{
		Meta: &commonv1.RequestMeta{ActorUserId: "u1"}, CompanyId: "co-2",
	})
	if err != nil || len(got.GetEvents()) != 1 || got.GetEvents()[0].GetCompanyId() != "co-1" {
		t.Fatalf("purchaser must be scoped to own company: %v %+v", err, got)
	}
	admin := grpcapi.NewServer(svc, fakeActors{actor: identity.Actor{UserID: "a1", Role: "platform_admin"}})
	all, err := admin.ListEvents(t.Context(), &auditv1.ListEventsRequest{Meta: &commonv1.RequestMeta{ActorUserId: "a1"}})
	if err != nil || len(all.GetEvents()) != 2 {
		t.Fatalf("platform admin must list all: %v %+v", err, all)
	}
}
