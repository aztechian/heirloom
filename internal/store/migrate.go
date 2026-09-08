package store

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

// Migrations are per-dialect SQL files rather than generated from Go structs.
//
// Automatic migration from model definitions is convenient for one dialect and
// actively harmful for two: it hides where the schemas diverge, produces
// different results on each backend, and gives no place to put a data
// migration. Explicit SQL means a schema change is reviewable and the
// divergence between dialects is visible in a diff.
//
// The files live under this package so that go:embed can reach them -- embed
// cannot include paths outside the embedding package's directory tree.
//
//go:embed migrations/sqlite/*.sql migrations/postgres/*.sql
//nolint:gochecknoglobals // embed requires a package-level var
var migrationsFS embed.FS

// gooseDialect maps our dialect names to goose's, which differ (goose expects
// "sqlite3"). Kept as an explicit mapping so a rename upstream fails loudly
// here rather than silently selecting the wrong SQL parser.
func gooseDialect(d Dialect) (string, error) {
	switch d {
	case DialectSQLite:
		return "sqlite3", nil
	case DialectPostgres:
		return "postgres", nil
	default:
		return "", fmt.Errorf("store: no goose dialect for %q", d)
	}
}

// Migrate applies all pending migrations for the given dialect.
//
// Note that this runs for Postgres even though Open refuses to connect to it.
// That is intentional and is the mechanism that keeps the second dialect
// honest: the Postgres migrations are executed against a real Postgres in CI,
// so schema drift is caught at commit time instead of being discovered when
// Postgres support is finally wired up.
//
// NOT SAFE FOR CONCURRENT USE. goose keeps the base FS and the dialect in
// package-level state, so two Migrate calls for different dialects racing each
// other would apply one dialect's SQL under the other's version table. In
// practice this is called once at startup, and the two-dialect tests are
// sequential. If that ever stops being true, this needs a mutex.
func Migrate(db *sql.DB, dialect Dialect) error {
	gooseName, err := gooseDialect(dialect)
	if err != nil {
		return err
	}

	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect(gooseName); err != nil {
		return fmt.Errorf("store: set goose dialect %q: %w", gooseName, err)
	}

	dir := "migrations/" + string(dialect)
	if err := goose.Up(db, dir); err != nil {
		return fmt.Errorf("store: migrate %s: %w", dialect, err)
	}

	return nil
}

// MigrationsFS exposes the embedded migrations so tests can assert that both
// dialects define the same migration versions. Two dialects drifting out of
// step -- a migration added to one and forgotten in the other -- is the single
// most likely failure mode of this arrangement, and it is cheap to test for.
func MigrationsFS() embed.FS { return migrationsFS }
