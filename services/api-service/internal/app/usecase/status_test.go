package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
)

func TestStatusSnapshotHidesFromCompanyUsers(t *testing.T) {
	status := NewStatus(nil)
	_, err := status.Snapshot(context.Background(), identity.User{Role: identity.RoleCompanyAdmin})
	if !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestStatusSnapshotReportsProbeResults(t *testing.T) {
	status := NewStatus(nil).WithProbes([]StatusProbe{
		{ID: port.ComponentAPI, Ping: nil},
		{ID: port.ComponentQueue, Ping: func(context.Context) error { return errors.New("down") }},
		{ID: port.ComponentFiles, Ping: func(context.Context) error { return nil }},
	})
	items, err := status.Snapshot(context.Background(), identity.User{Role: identity.RolePlatformAdmin})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d components", len(items))
	}
	if !items[0].OK || items[1].OK || !items[2].OK {
		t.Fatalf("got %+v", items)
	}
}

func TestHTTPGetProbeRejectsEmptyURL(t *testing.T) {
	err := HTTPGetProbe(nil, "")(context.Background())
	if err == nil {
		t.Fatal("empty health url should fail")
	}
}

func TestStatusSnapshotHonorsTimeout(t *testing.T) {
	status := NewStatus(func() time.Duration { return 20 * time.Millisecond }).WithProbes([]StatusProbe{
		{ID: port.ComponentWorker, Ping: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}},
	})
	items, err := status.Snapshot(context.Background(), identity.User{Role: identity.RolePlatformAdmin})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].OK {
		t.Fatal("hung probe should be reported down")
	}
}
