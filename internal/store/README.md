# internal/store

Relational persistence and the dialect boundary.

## What is actually implemented

| Dialect    | Connect | Migrations    | Query layer      |
| ---------- | ------- | ------------- | ---------------- |
| `sqlite`   | yes     | yes           | interface only   |
| `postgres` | **no**  | yes (CI only) | no               |

There is no query layer yet on either dialect — `TextSearch` is declared and
unimplemented, and nothing else is declared at all. The package today is `Open`,
`Migrate`, and the schema.

`store.Open` returns `ErrDialectUnsupported` for `postgres`. That is not an
oversight to be fixed by adding a driver import — see
[Before Postgres can be turned on](#before-postgres-can-be-turned-on).

## Why two dialects at all

The reason is deployment topology, not capability. Nothing about the workload
needs Postgres: at the characterized scale (~2,000 assets now, 10,000 as the
support ceiling, single-digit concurrent users, writes measured in a handful per
minute) SQLite is not close to a limit.

SQLite's cost is that the application pod holds durable state. On Kubernetes that
means a PersistentVolume, `replicas: 1`, and `strategy: Recreate`. Postgres moves
that durability to a database operator, which is not less storage — it is storage
somebody else already knows how to back up.

So SQLite is the default and the only implemented dialect, and Postgres exists as
an escape hatch for people who already run Postgres and would rather not add a
stateful pod.

## Why the second dialect is here before it works

Because retrofitting a dialect is expensive and keeping a seam open is cheap. A
year of SQLite-only development produces `INSERT OR IGNORE`, `strftime`,
`last_insert_rowid()`, and `?` placeholders scattered through the query layer, and
every one of them has to be found by hand later. Having the constraint present
from commit one means the SQLite-isms never accumulate.

The mechanism that makes this real rather than aspirational is that the Postgres
migrations run against a real Postgres in CI. An untested second dialect is just
a comment.

## Why `database/sql` and hand-written SQL, not GORM

This is a **deliberate divergence from the reference project**, which uses GORM.

- With two dialects, the differences should be visible in a diff, not resolved
  invisibly by a driver abstraction. An ORM makes dialect divergence silent,
  which is exactly the property that hurts here.
- The eventual assertion store — claims with author, basis, polarity, and
  precedence resolution — is a recursive/windowed query problem. Those queries
  are where an ORM stops helping and starts being an obstacle.
- Schema changes want to be reviewable SQL. Automigration from struct tags has no
  place to put a data migration and produces different results per dialect.

The cost is real: more boilerplate per query, manual `Scan` targets, and no free
association loading. Accepted.

## Migrations

[goose](https://github.com/pressly/goose) over an embedded FS, one directory per
dialect:

```
internal/store/migrations/
├── postgres/00001_init.sql
└── sqlite/00001_init.sql
```

They live under this package because `go:embed` cannot reach files above the
embedding package's directory.

`Migrate` is idempotent, and the intent is for the server to call it on startup —
there is no `cmd/server` yet, so today only the tests call it. Migrations are
forward-only in practice; `-- +goose Down` blocks exist for local development,
not for production rollback.

`Migrate` is **not safe for concurrent use**: goose holds the base FS and the
dialect in package-level state, so two calls for different dialects racing each
other would apply one dialect's SQL under the other's version table.

### Adding a migration

1. Create `NNNNN_description.sql` in **both** dialect directories. The version
   number and name must match. `TestMigrationParity` fails if they do not.
2. Wrap statements in `-- +goose StatementBegin` / `-- +goose StatementEnd`.
   Strictly this is only *required* for statements containing internal semicolons
   — trigger bodies, `CREATE FUNCTION`, `DO` blocks — and plain single-statement
   DDL does not need it on either dialect. Wrapping everything anyway is a
   harmless convention that removes the need to know which case you are in, and
   it keeps interleaved explanatory comments from being ambiguous to the parser.
3. Never edit a released migration. goose records applied versions and will not
   re-run one, so an edit produces two divergent schemas in the wild.

### Schema conventions

- **UUIDs are lowercase hex `TEXT`**, not `BLOB` or Postgres `uuid`. Costs ~20
  bytes a row; buys a database legible in a `sqlite3` shell, which is worth a
  great deal when diagnosing an archive by hand years from now.
- **Timestamps are RFC 3339 UTC `TEXT`** (`2026-09-07T18:22:04Z`) in both
  dialects, not `timestamptz`. Fixed-width UTC text sorts lexicographically in
  chronological order, so it can be used directly in a pagination cursor and
  compares identically on both backends. The cost — no interval arithmetic in
  SQL, no database-level validation — is documented at the top of the Postgres
  migration, along with the note that this is a shared-query-layer tradeoff and
  not a claim that native types are wrong.
- **Enums are `TEXT` + `CHECK`**, not integers or Postgres `ENUM`. Adding a value
  is therefore visibly a migration, which is the point: the API enum and the
  database constraint cannot drift silently.
- **No stored `assetCount`.** `Collection.assetCount` is `COUNT(*)` at read time.
  A counter needs trigger or transaction discipline to stay correct, and at
  10,000 rows the count is not measurable.
- **`UNIQUE (collection_id, content_hash)`** is the asset identity and the dedupe
  mechanism. The upload handler relies on the conflict rather than checking for
  existence first, which would race. Uniqueness is per collection on purpose: two
  families may legitimately hold the same photograph.

The initial migration covers **collections and assets only** — the API surface
that exists. The assertion model (people, face detections, claims) is the
interesting design work in this project and has no API yet; guessing at its
tables now would produce a migration to delete rather than a foundation.

## SQLite pragmas

Set in the DSN, not executed after `Open`. `database/sql` pools connections and
opens them lazily, so a `PRAGMA` statement applies only to whichever pooled
connection happened to serve it. This is a well-worn source of bugs where foreign
keys silently stop being enforced under load. `TestSQLiteMigrate` asserts
`foreign_keys` and `journal_mode` on a live pool for exactly that reason.

`busy_timeout=5000`, `journal_mode=WAL`, `foreign_keys=1`,
`synchronous=NORMAL`. Pool capped at 4 connections: SQLite allows one writer, and
a small pool also bounds memory when a clustering run is holding embeddings.

Tests use a file in `t.TempDir()`, never `:memory:` — an in-memory database is
per-connection, so a migration and a later query can land on differently migrated
databases.

## The one permitted leak: `TextSearch`

SQLite FTS5 and Postgres `tsvector` are not one feature in two syntaxes. They
differ in index construction, ranking, tokenization, and query grammar, and any
shared query builder over both is worse than either. So `TextSearch` is an
interface each dialect implements honestly, and it is the only place in this
package where the dialect is allowed to show through. Callers get one narrow
surface instead of dialect conditionals in handlers.

It exists before there is anything to search because searching OCR'd document
text is core scope, not a nicety.

## Before Postgres can be turned on

Wiring `pgx` into `Open` is a few lines. These are the things that actually have
to be true first, and none of them are:

1. **Placeholders.** SQLite uses `?`, Postgres uses `$1`. Every query in the
   query layer needs rewriting or a rebinder. Decide which before writing many
   queries, not after.
2. **Conflict detection.** The upload path branches on a unique-constraint
   violation to return `200` instead of `201`. Detecting that means
   `sqlite3.Error` on one side and `pgconn.PgError` code `23505` on the other,
   behind a `store`-level `IsUniqueViolation(err)` helper that does not exist yet.
3. **`TextSearch`** has no Postgres implementation.
4. **Concurrency assumptions.** The SQLite path can assume a single writer.
   Anything relying on that — read-then-write without a transaction — becomes a
   race under Postgres and must be found deliberately.
5. **A Postgres CI leg** running the real query layer, not just migrations.

Until all five are done, `ErrDialectUnsupported` is the honest answer.

## Not verified

No Go code in this package has been compiled or run. There is no Go toolchain in
the environment it was written in and no network egress to fetch one. Treat
everything here as reviewed-but-unrun: the `modernc.org/sqlite` DSN pragma syntax,
the goose API surface, and the `CHECK` constraint spellings are all from
documentation rather than from a passing test.

**`go.sum` does not yet contain the new dependencies**, because they could not be
resolved offline. The first thing to do with a real toolchain is:

```sh
go get -u github.com/pressly/goose/v3 modernc.org/sqlite github.com/jackc/pgx/v5
go mod tidy
```

Until that is committed, every CI job that builds this package fails on a missing
`go.sum` entry, and the `hygiene` job's tidy check fails as well. That is the
correct behaviour — CI is telling you the dependency set was written by hand — but
it means the first commit after this one is a tidy commit, not a feature.
