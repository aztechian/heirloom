// Package store owns the relational persistence layer and the dialect
// boundary. The archive supports two database dialects on purpose, for
// deployment reasons rather than capability ones: SQLite needs no
// infrastructure and is the default, while Postgres lets the application pod
// hold no durable state on Kubernetes by delegating durability to a database
// operator. Nothing in the workload requires Postgres.
//
// Only SQLite is implemented today. Postgres exists here as a real but
// unimplemented dialect so that migrations, tests, and CI exercise the
// two-dialect shape from the beginning. Retrofitting a second dialect after a
// year of accumulated SQLite-isms is the expensive path; keeping the seam open
// and tested is cheap.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	// Pure-Go SQLite driver, registered as "sqlite". Chosen over cgo-backed
	// drivers because it needs no build tags or C toolchain, not because cgo is
	// forbidden -- see the cgo decision in the project README.
	_ "modernc.org/sqlite"
)

// Dialect identifies a supported database backend.
type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
)

// ErrDialectUnsupported is returned for a dialect that is recognized but not
// yet implemented. It is deliberately distinct from a parse error: a caller
// configuring Postgres has not made a typo, they have asked for something that
// is on the roadmap.
var ErrDialectUnsupported = errors.New("store: dialect recognized but not implemented")

// maxSQLiteConns bounds the connection pool. SQLite permits many readers but
// exactly one writer; WAL plus busy_timeout handles the contention, and the real
// write volume here is a handful of people typing occasionally.
const maxSQLiteConns = 4

// ParseDialect converts a configuration string to a Dialect.
func ParseDialect(s string) (Dialect, error) {
	switch d := Dialect(strings.ToLower(strings.TrimSpace(s))); d {
	case DialectSQLite, DialectPostgres:
		return d, nil
	default:
		return "", fmt.Errorf("store: unknown dialect %q (want %q or %q)",
			s, DialectSQLite, DialectPostgres)
	}
}

// Config selects and locates a database.
type Config struct {
	Dialect Dialect
	// DSN is dialect-specific. For SQLite it is a file path; Open supplies the
	// required pragmas, so callers should pass a bare path rather than a URI.
	DSN string
}

// Open connects to the configured database and applies dialect-specific
// connection settings. It does not run migrations; call Migrate for that.
func Open(cfg Config) (*sql.DB, error) {
	switch cfg.Dialect {
	case DialectSQLite:
		return openSQLite(cfg.DSN)
	case DialectPostgres:
		// Wiring a driver here is the easy part. The reason this is not done
		// yet is the query layer: see internal/store/README.md for what has to
		// be true before this returns a *sql.DB.
		return nil, fmt.Errorf("%w: %s", ErrDialectUnsupported, DialectPostgres)
	default:
		return nil, errors.New("store: dialect not set")
	}
}

func openSQLite(path string) (*sql.DB, error) {
	if path == "" {
		return nil, errors.New("store: sqlite dsn is empty")
	}

	// These pragmas must be set per connection, not once at startup, because
	// database/sql pools connections and opens new ones lazily -- a PRAGMA
	// executed after Open applies only to whichever pooled connection happened
	// to serve it. Passing them in the DSN is what makes them apply to every
	// connection the pool creates. This is a well-worn source of bugs where
	// foreign keys silently stop being enforced under load.
	//
	//   busy_timeout  wait rather than immediately returning SQLITE_BUSY
	//   journal_mode  WAL, so readers do not block the writer
	//   foreign_keys  off by default in SQLite, which is rarely what anyone wants
	//   synchronous   NORMAL is safe under WAL and much faster than FULL
	//
	// The path is URL-escaped rather than interpolated. A file path may
	// legitimately contain a space, '?', '#', or '%' -- this project's own
	// working directory contains a space -- and any of those would otherwise
	// truncate or corrupt the query string, silently dropping every pragma.
	pragmas := "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)"

	dsn := (&url.URL{
		Scheme:   "file",
		Opaque:   (&url.URL{Path: path}).EscapedPath(),
		RawQuery: pragmas,
	}).String()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open sqlite: %w", err)
	}

	// SQLite permits many readers but exactly one writer. WAL plus busy_timeout
	// handles the contention, and the real write volume here is a handful of
	// people typing occasionally, so a small pool is ample. Keeping it small
	// also bounds memory when a clustering run is holding embeddings in RAM.
	db.SetMaxOpenConns(maxSQLiteConns)
	db.SetMaxIdleConns(maxSQLiteConns)

	// sql.Open is lazy and validates almost nothing, so a malformed DSN or an
	// unwritable directory would otherwise surface as a baffling error from the
	// first query rather than from the call that was actually wrong.
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: connect to sqlite at %s: %w", path, err)
	}

	return db, nil
}

// TextSearch is the one place the dialect abstraction is allowed to leak, and
// it is isolated here so that the leak is visible rather than smeared through
// the query layer.
//
// SQLite FTS5 and Postgres tsvector are not the same feature in different
// syntax: they differ in index construction, ranking, tokenization, and query
// grammar. Any attempt to express both through a shared query builder produces
// something that is worse than either. So each dialect implements this
// interface honestly, and callers get one narrow, well-tested surface instead
// of dialect conditionals scattered across handlers.
//
// Searching OCR'd document text is core scope, not a nicety, which is why this
// interface exists before there is anything to search.
type TextSearch interface {
	// Index makes the given text searchable for an asset. Called by the ingest
	// pipeline after OCR; must be idempotent, because reprocessing re-runs it.
	Index(assetID string, text string) error

	// Search returns matching asset IDs, best match first. Query syntax is
	// whatever the dialect natively accepts; callers must not construct
	// dialect-specific operators and should pass user input through.
	Search(collectionID string, query string, limit int) ([]string, error)
}
