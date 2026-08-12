package pgengine_test

import (
	"database/sql"
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
)

func TestPostgresPersistence_NewIsPersistent(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}
	defer eng.Close()

	if !eng.Profile().IsPersistent() {
		t.Error("Postgres engine should always be persistent")
	}

	if eng.Profile().IsVolatile() {
		t.Error("Postgres engine should never be volatile")
	}
}

func TestPostgresPersistence_FromDBIsPersistent(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("pgx", pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}
	defer db.Close()

	eng, err := pgengine.NewFromDB(db)
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}
	defer eng.Close()

	if !eng.Profile().IsPersistent() {
		t.Error("Postgres engine from a caller-owned DB should be persistent")
	}
}
