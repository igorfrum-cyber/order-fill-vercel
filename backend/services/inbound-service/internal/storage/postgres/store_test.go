package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"order-fill/backend/services/inbound-service/internal/domain"
)

func TestMapCompanyInboundUpsertErrorUniqueViolation(t *testing.T) {
	t.Parallel()
	err := mapCompanyInboundUpsertError(&pgconn.PgError{Code: "23505"})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("unique sender must map to ErrInvalid, got %v", err)
	}
}

func TestMapCompanyInboundUpsertErrorOther(t *testing.T) {
	t.Parallel()
	orig := errors.New("connection reset")
	err := mapCompanyInboundUpsertError(orig)
	if errors.Is(err, domain.ErrInvalid) {
		t.Fatal("non-unique errors must not become ErrInvalid")
	}
	if !errors.Is(err, orig) {
		t.Fatalf("got %v", err)
	}
}
