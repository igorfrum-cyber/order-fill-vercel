package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"order-fill/backend/services/job-service/internal/domain"
)

func TestMapConflict(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "unique violation", err: &pgconn.PgError{Code: "23505"}, want: domain.ErrConflict},
		{name: "other error", err: errors.New("boom"), want: errors.New("boom")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := mapConflict(tt.err)
			if tt.want == domain.ErrConflict {
				if !errors.Is(got, domain.ErrConflict) {
					t.Fatalf("got=%v", got)
				}
				return
			}
			if got.Error() != tt.want.Error() {
				t.Fatalf("got=%v want=%v", got, tt.want)
			}
		})
	}
}
