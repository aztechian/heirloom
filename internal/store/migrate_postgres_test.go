package store

import (
	"database/sql"
	"os"
	"testing"

	// The pgx stdlib driver is imported here, in a test file, and nowhere else.
	// That is the point of the arrangement: the Postgres *migrations* are
	// verified against a real Postgres from commit one, while the application
	// still refuses the dialect, because a driver is the easy part and the query
	// layer is not. See README.md, "Before Postgres can be turned on".
	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestPostgresMigrate applies the Postgres migrations to a real server.
//
// Skipped unless HEIRLOOM_TEST_POSTGRES_DSN is set, so local `go test ./...`
// needs no infrastructure. CI sets it against a service container, which is what
// makes the second dialect more than a comment: without this, a migration added
// only to the SQLite directory in the right shape but the wrong SQL would pass
// TestMigrationParity and fail years later.
func TestPostgresMigrate(t *testing.T) {
	dsn := os.Getenv("HEIRLOOM_TEST_POSTGRES_DSN")
	if dsn == "" {
		// go test reports a skip as success, so a skip in CI would turn the
		// dialect-drift job into a green no-op -- exactly the failure the job
		// exists to prevent. Under CI, an unset DSN is a configuration bug.
		if os.Getenv("CI") != "" {
			t.Fatal("HEIRLOOM_TEST_POSTGRES_DSN must be set in CI")
		}
		t.Skip("HEIRLOOM_TEST_POSTGRES_DSN not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	if err := Migrate(db, DialectPostgres); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Idempotent, as on SQLite.
	if err := Migrate(db, DialectPostgres); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	// The constraints that carry meaning must behave the same on both dialects,
	// not merely exist. Placeholders are $1 here and ? on SQLite -- which is
	// itself item 1 on the list of things blocking a shared query layer.
	if _, err := db.Exec(`INSERT INTO collections (id, name, slug, created_at, updated_at)
		VALUES ('c1', 'Test', 'test', '2026-09-07T00:00:00Z', '2026-09-07T00:00:00Z')`); err != nil {
		t.Fatalf("insert collection: %v", err)
	}

	insert := `INSERT INTO assets
		(id, collection_id, content_hash, original_filename, media_type,
		 size_bytes, kind, status, sensitivity, storage_key, uploaded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	hash := "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

	if _, err := db.Exec(insert, "a1", "c1", hash, "a.jpg", "image/jpeg",
		100, "photo", "pending", "normal", "k1", "2026-09-07T00:00:00Z"); err != nil {
		t.Fatalf("insert first asset: %v", err)
	}

	if _, err := db.Exec(insert, "a2", "c1", hash, "b.jpg", "image/jpeg",
		100, "photo", "pending", "normal", "k2", "2026-09-07T00:00:01Z"); err == nil {
		t.Error("duplicate (collection_id, content_hash) was accepted")
	}

	// Foreign keys are enforced by default in Postgres, unlike SQLite -- assert
	// anyway, so the two dialect tests make the same claims.
	if _, err := db.Exec(insert, "a3", "nope", hash, "a.jpg", "image/jpeg",
		100, "photo", "pending", "normal", "k3", "2026-09-07T00:00:00Z"); err == nil {
		t.Error("asset with a dangling collection_id was accepted")
	}

	// A distinct hash, so a failure here is the sensitivity CHECK and not the
	// dedupe UNIQUE firing first.
	other := "0000000000000000000000000000000000000000000000000000000000000001"
	if _, err := db.Exec(insert, "a4", "c1", other, "a.jpg", "image/jpeg",
		100, "photo", "pending", "nope", "k4", "2026-09-07T00:00:00Z"); err == nil {
		t.Error("invalid sensitivity was accepted")
	}
}
