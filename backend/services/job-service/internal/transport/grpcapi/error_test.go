package grpcapi

import (
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"order-fill/backend/services/job-service/internal/domain"
)

func TestToStatusMapsDomainErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{
			name: "invalid workbook extension",
			err:  fmt.Errorf("%w: file %q must be .xlsx or .xlsm", domain.ErrInvalid, "qa-not-excel.txt"),
			code: codes.InvalidArgument,
		},
		{name: "not found", err: domain.ErrNotFound, code: codes.NotFound},
		{name: "conflict", err: domain.ErrConflict, code: codes.AlreadyExists},
		{name: "unauthorized", err: domain.ErrUnauthorized, code: codes.Unauthenticated},
		{name: "unknown", err: fmt.Errorf("enqueue job: xadd failed"), code: codes.Internal},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := toStatus(tc.err)
			if status.Code(got) != tc.code {
				t.Fatalf("code=%v want %v err=%v", status.Code(got), tc.code, got)
			}
			if status.Convert(got).Message() != tc.err.Error() {
				t.Fatalf("message=%q want %q", status.Convert(got).Message(), tc.err.Error())
			}
		})
	}
}

func TestToStatusNil(t *testing.T) {
	t.Parallel()
	if err := toStatus(nil); err != nil {
		t.Fatalf("got %v", err)
	}
}
