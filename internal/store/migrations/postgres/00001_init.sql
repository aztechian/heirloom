-- Postgres counterpart to migrations/sqlite/00001_init.sql.
--
-- Postgres is not wired up in the application yet (store.Open refuses it), but
-- these migrations run against a real Postgres in CI. That is the whole point:
-- the alternative is discovering a year of accumulated SQLite-isms on the day
-- Postgres is finally needed.
--
-- On deliberately not using native Postgres types
-- ------------------------------------------------
-- This schema uses TEXT for UUIDs and TIMESTAMPTZ-shaped values instead of the
-- native `uuid` and `timestamptz` types, which is not what one would write for a
-- Postgres-only application.
--
-- The reason is that the query layer above is shared. Native types change what
-- comes back from a Scan -- uuid.UUID vs string, time.Time vs string -- and that
-- means either two sets of scan targets or a conversion layer whose only job is
-- to paper over the difference. With timestamps stored as fixed-width RFC 3339
-- UTC text, both dialects sort identically, compare identically, and produce the
-- same cursor bytes, so pagination is provably the same on both.
--
-- What this costs: no interval arithmetic in SQL, no timezone conversion in the
-- database, 16 bytes per UUID instead of 16 bits of type safety, and no
-- database-level guarantee that a timestamp is well-formed. At this scale none
-- of that binds, and none of it is irreversible -- a later migration can widen
-- the columns to native types once (and if) Postgres is the primary target. It
-- would be a mistake to read this as a claim that native types are wrong.
--
-- +goose Up
-- +goose StatementBegin
CREATE TABLE collections (
    id          TEXT PRIMARY KEY NOT NULL,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',

    -- RFC 3339 UTC strings. See the header note.
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,

    CONSTRAINT collections_slug_unique UNIQUE (slug),
    CONSTRAINT collections_name_len    CHECK (length(name) BETWEEN 1 AND 200),
    CONSTRAINT collections_slug_len    CHECK (length(slug) BETWEEN 1 AND 64),
    CONSTRAINT collections_desc_len    CHECK (length(description) <= 2000)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE assets (
    id                TEXT PRIMARY KEY NOT NULL,
    collection_id     TEXT NOT NULL
        REFERENCES collections(id) ON DELETE CASCADE,
    content_hash      TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    original_path     TEXT NOT NULL DEFAULT '',
    media_type        TEXT NOT NULL,

    -- BIGINT to match SQLite's 64-bit INTEGER. A 4-byte INTEGER would cap at
    -- 2 GiB, which is under the documented maximum asset size.
    size_bytes        BIGINT NOT NULL,

    kind              TEXT NOT NULL
        CHECK (kind IN ('photo', 'document', 'unknown')),
    status            TEXT NOT NULL
        CHECK (status IN ('pending', 'processing', 'ready', 'failed')),
    sensitivity       TEXT NOT NULL
        CHECK (sensitivity IN ('normal', 'restricted')),
    error             TEXT NOT NULL DEFAULT '',
    storage_key       TEXT NOT NULL,
    uploaded_at       TEXT NOT NULL,
    processed_at      TEXT,

    CONSTRAINT assets_content_identity UNIQUE (collection_id, content_hash),

    CONSTRAINT assets_hash_format   CHECK (length(content_hash) = 64),
    CONSTRAINT assets_size_positive CHECK (size_bytes > 0),
    CONSTRAINT assets_filename_len  CHECK (length(original_filename) BETWEEN 1 AND 255),
    CONSTRAINT assets_path_len      CHECK (length(original_path) <= 4096),
    CONSTRAINT assets_error_only_when_failed
        CHECK ((status = 'failed') OR (error = ''))
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX assets_collection_page_idx
    ON assets (collection_id, uploaded_at, id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX assets_collection_status_idx ON assets (collection_id, status);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX assets_collection_kind_idx ON assets (collection_id, kind);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX assets_status_idx ON assets (status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE assets;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE collections;
-- +goose StatementEnd
