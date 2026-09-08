package store

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"testing"
)

var migrationVersion = regexp.MustCompile(`^(\d{5})_[a-z0-9_]+\.sql$`)

// TestMigrationParity is the guard for the failure mode this whole arrangement
// invites: adding a migration to one dialect and forgetting the other. Because
// Postgres is not reachable from the application yet, nothing else would notice.
func TestMigrationParity(t *testing.T) {
	versions := map[Dialect]map[string]string{}

	for _, d := range []Dialect{DialectSQLite, DialectPostgres} {
		versions[d] = map[string]string{}

		// path.Join, not filepath.Join: io/fs paths are always slash-separated,
		// so filepath would yield a backslash path on Windows and ReadDir would
		// return ErrNotExist.
		entries, err := fs.ReadDir(migrationsFS, path.Join("migrations", string(d)))
		if err != nil {
			t.Fatalf("read %s migrations: %v", d, err)
		}

		for _, e := range entries {
			m := migrationVersion.FindStringSubmatch(e.Name())
			if m == nil {
				t.Errorf("%s/%s: filename must match NNNNN_snake_case.sql", d, e.Name())
				continue
			}
			if prev, dup := versions[d][m[1]]; dup {
				t.Errorf("%s: duplicate version %s (%s and %s)", d, m[1], prev, e.Name())
			}
			versions[d][m[1]] = e.Name()
		}
	}

	if len(versions[DialectSQLite]) == 0 {
		t.Fatal("no migrations embedded; check the go:embed patterns")
	}

	for v, name := range versions[DialectSQLite] {
		if _, ok := versions[DialectPostgres][v]; !ok {
			t.Errorf("version %s exists for sqlite (%s) but not postgres", v, name)
		}
	}
	for v, name := range versions[DialectPostgres] {
		if _, ok := versions[DialectSQLite][v]; !ok {
			t.Errorf("version %s exists for postgres (%s) but not sqlite", v, name)
		}
	}
}

func TestParseDialect(t *testing.T) {
	for in, want := range map[string]Dialect{
		"sqlite":    DialectSQLite,
		"  SQLite ": DialectSQLite,
		"postgres":  DialectPostgres,
		"POSTGRES":  DialectPostgres,
	} {
		got, err := ParseDialect(in)
		if err != nil {
			t.Errorf("ParseDialect(%q): unexpected error %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseDialect(%q) = %q, want %q", in, got, want)
		}
	}

	// "sqlite3" and "postgresql" are the names people will reach for by habit.
	// Rejecting them is intentional -- silently accepting aliases means the
	// configuration value and the migration directory name can diverge.
	for _, in := range []string{"", "sqlite3", "postgresql", "mysql", "duckdb"} {
		if _, err := ParseDialect(in); err == nil {
			t.Errorf("ParseDialect(%q): expected error", in)
		}
	}
}

// TestOpenPostgresUnsupported pins the contract that a Postgres configuration
// fails with a distinguishable error rather than a driver panic or a confusing
// connection failure. Callers are expected to match on ErrDialectUnsupported.
func TestOpenPostgresUnsupported(t *testing.T) {
	_, err := Open(Config{Dialect: DialectPostgres, DSN: "postgres://localhost/heirloom"})
	if err == nil {
		t.Fatal("expected an error for the postgres dialect")
	}
	if !errors.Is(err, ErrDialectUnsupported) {
		t.Fatalf("error %v does not wrap ErrDialectUnsupported", err)
	}
}

// TestSQLiteMigrate exercises the real driver and the real migrations against a
// temporary file. It uses a file rather than :memory: on purpose: an in-memory
// SQLite database is per-connection, and database/sql pools connections, so a
// migration and a subsequent query can land on different -- and differently
// migrated -- databases.
func TestSQLiteMigrate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "heirloom.db")

	db, err := Open(Config{Dialect: DialectSQLite, DSN: path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := Migrate(db, DialectSQLite); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Migrate must be idempotent; the server runs it on every boot.
	if err := Migrate(db, DialectSQLite); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	var fkEnabled int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled); err != nil {
		t.Fatalf("read foreign_keys pragma: %v", err)
	}
	if fkEnabled != 1 {
		t.Error("foreign_keys is off; the DSN pragmas are not being applied")
	}

	var journal string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil {
		t.Fatalf("read journal_mode pragma: %v", err)
	}
	if journal != "wal" {
		t.Errorf("journal_mode = %q, want wal", journal)
	}

	// The dedupe identity is load-bearing: the upload handler relies on this
	// conflict rather than checking for existence first, which would race.
	insert := `INSERT INTO assets
		(id, collection_id, content_hash, original_filename, media_type,
		 size_bytes, kind, status, sensitivity, storage_key, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	hash := "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

	if _, err := db.Exec(`INSERT INTO collections (id, name, slug, created_at, updated_at)
		VALUES ('c1', 'Test', 'test', '2026-09-07T00:00:00Z', '2026-09-07T00:00:00Z')`); err != nil {
		t.Fatalf("insert collection: %v", err)
	}

	if _, err := db.Exec(insert, "a1", "c1", hash, "a.jpg", "image/jpeg",
		100, "photo", "pending", "normal", "k1", "2026-09-07T00:00:00Z"); err != nil {
		t.Fatalf("insert first asset: %v", err)
	}

	if _, err := db.Exec(insert, "a2", "c1", hash, "b.jpg", "image/jpeg",
		100, "photo", "pending", "normal", "k2", "2026-09-07T00:00:01Z"); err == nil {
		t.Error("duplicate (collection_id, content_hash) was accepted")
	}

	// The same bytes in a different collection must be a distinct asset: two
	// families may hold the same photograph and their catalogs stay independent.
	if _, err := db.Exec(`INSERT INTO collections (id, name, slug, created_at, updated_at)
		VALUES ('c2', 'Other', 'other', '2026-09-07T00:00:00Z', '2026-09-07T00:00:00Z')`); err != nil {
		t.Fatalf("insert second collection: %v", err)
	}
	if _, err := db.Exec(insert, "a3", "c2", hash, "a.jpg", "image/jpeg",
		100, "photo", "pending", "normal", "k3", "2026-09-07T00:00:00Z"); err != nil {
		t.Errorf("same hash in a different collection was rejected: %v", err)
	}

	// A foreign key to a missing collection must fail, which only happens if the
	// DSN pragma took effect on this pooled connection.
	if _, err := db.Exec(insert, "a4", "nope", hash, "a.jpg", "image/jpeg",
		100, "photo", "pending", "normal", "k4", "2026-09-07T00:00:00Z"); err == nil {
		t.Error("asset with a dangling collection_id was accepted")
	}

	// Enum CHECKs. Asserted on both dialects so the two dialect tests make the
	// same claims rather than merely both passing.
	if _, err := db.Exec(insert, "a6", "c1",
		"0000000000000000000000000000000000000000000000000000000000000002",
		"d.jpg", "image/jpeg", 100, "photo", "pending", "nope", "k6",
		"2026-09-07T00:00:00Z"); err == nil {
		t.Error("invalid sensitivity was accepted")
	}

	// error must be empty unless status is failed, so a stale message cannot
	// survive a successful reprocess.
	if _, err := db.Exec(`INSERT INTO assets
		(id, collection_id, content_hash, original_filename, media_type, size_bytes,
		 kind, status, sensitivity, error, storage_key, uploaded_at)
		VALUES ('a5', 'c1', '0000000000000000000000000000000000000000000000000000000000000001',
		        'c.jpg', 'image/jpeg', 1,
		        'photo', 'ready', 'normal', 'boom', 'k5', '2026-09-07T00:00:00Z')`); err == nil {
		t.Error("non-empty error with status=ready was accepted")
	}
}
