package securecfg

import "testing"

func TestPostgresAllowsLocalDefault(t *testing.T) {
	t.Parallel()
	if err := Postgres("local", "postgres://order_fill:order_fill@postgres/db?sslmode=disable"); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRejectsDefaultPasswordOutsideLocal(t *testing.T) {
	t.Parallel()
	err := Postgres("production", "postgres://order_fill:order_fill@postgres/db?sslmode=require")
	if err == nil {
		t.Fatal("expected default password error")
	}
}

func TestPostgresRejectsDisableSSLOutsideLocal(t *testing.T) {
	t.Parallel()
	err := Postgres("production", "postgres://user:secret@postgres/db?sslmode=disable")
	if err == nil {
		t.Fatal("expected sslmode error")
	}
}

func TestPostgresAcceptsRequire(t *testing.T) {
	t.Parallel()
	if err := Postgres("production", "postgres://user:secret@postgres/db?sslmode=require"); err != nil {
		t.Fatal(err)
	}
}

func TestRedisRejectsMissingPasswordOutsideLocal(t *testing.T) {
	t.Parallel()
	if err := Redis("production", "redis://redis:6379/0"); err == nil {
		t.Fatal("expected password error")
	}
}

func TestRedisAcceptsPassword(t *testing.T) {
	t.Parallel()
	if err := Redis("production", "redis://:secret@redis:6379/0"); err != nil {
		t.Fatal(err)
	}
}
