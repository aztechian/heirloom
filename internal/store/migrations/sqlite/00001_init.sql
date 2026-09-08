-- Collections and assets: the schema for the API surface that exists today.
--
-- Deliberately does NOT include the assertion model (claims, people, face
-- detections). That model is the interesting design work in this project and it
-- has no API yet; guessing at its tables now would produce a migration to
-- delete rather than a foundation to build on.
--
-- +goose Up
-- +goose StatementBegin
CREATE TABLE collections (
    -- UUIDs are stored as lowercase hex text rather than BLOB. Text costs 20
    -- bytes more per row -- irrelevant at 10,000 assets -- and buys a database
    -- that is legible in a sqlite3 shell, which matters a great deal when
    -- diagnosing an archive by hand years from now.
    id          TEXT PRIMARY KEY NOT NULL,
    name        TEXT NOT NULL,

    -- Immutable once set, because it appears in links shared with relatives. No
    -- database mechanism enforces immutability; the handler must refuse to
    -- update it.
    slug        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',

    -- Timestamps are RFC 3339 UTC strings ('2026-09-07T18:22:04Z'), not SQLite
    -- integers or Postgres timestamptz. Fixed-width UTC text sorts
    -- lexicographically in the same order it sorts chronologically, which is
    -- what makes it usable directly in a pagination cursor, and it survives a
    -- dump to text without an encoding convention. The application supplies
    -- these values; there is no DEFAULT, so a missing timestamp is a loud NOT
    -- NULL failure rather than a silent row stamped with the migration's clock.
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

    -- ON DELETE CASCADE is correct here only because deleting a collection is
    -- meant to remove its catalog rows. It does not touch stored bytes: object
    -- deletion is a separate, deliberate operation, since the whole point of
    -- content-addressed storage is that the catalog can be rebuilt from the
    -- objects but not the reverse.
    collection_id     TEXT NOT NULL
        REFERENCES collections(id) ON DELETE CASCADE,

    -- Lowercase hex SHA-256 of the source bytes.
    content_hash      TEXT NOT NULL,

    original_filename TEXT NOT NULL,

    -- Empty string rather than NULL when not supplied. NULL and '' would be two
    -- representations of "no path", and every query would have to handle both.
    original_path     TEXT NOT NULL DEFAULT '',

    -- Detected from content, never from the client's Content-Type header.
    media_type        TEXT NOT NULL,
    size_bytes        INTEGER NOT NULL,

    -- Enums are TEXT + CHECK rather than integers. The tradeoff is that adding
    -- an enum value requires a migration -- which is the point: the API enum and
    -- the database constraint cannot drift silently, and a table dump is
    -- readable without a lookup table.
    kind              TEXT NOT NULL
        CHECK (kind IN ('photo', 'document', 'unknown')),
    status            TEXT NOT NULL
        CHECK (status IN ('pending', 'processing', 'ready', 'failed')),
    sensitivity       TEXT NOT NULL
        CHECK (sensitivity IN ('normal', 'restricted')),

    -- Populated only when status = 'failed'. The paired CHECK below enforces
    -- that correspondence, so a stale error message cannot survive a successful
    -- reprocess.
    error             TEXT NOT NULL DEFAULT '',

    -- Opaque locator in the configured storage backend (object key or relative
    -- filesystem path). Not exposed through the API: clients address assets by
    -- ID, and leaking the layout would freeze it.
    storage_key       TEXT NOT NULL,

    uploaded_at       TEXT NOT NULL,
    processed_at      TEXT,

    -- The dedupe identity. Uniqueness is per collection, not global: two
    -- families may legitimately hold the same photograph, and their catalogs
    -- must stay independent even though the bytes are identical. This constraint
    -- is what makes upload idempotent -- the handler relies on the conflict, and
    -- application-level "does it exist" checks would race.
    CONSTRAINT assets_content_identity UNIQUE (collection_id, content_hash),

    CONSTRAINT assets_hash_format  CHECK (length(content_hash) = 64),
    CONSTRAINT assets_size_positive CHECK (size_bytes > 0),
    CONSTRAINT assets_filename_len CHECK (length(original_filename) BETWEEN 1 AND 255),
    CONSTRAINT assets_path_len     CHECK (length(original_path) <= 4096),
    CONSTRAINT assets_error_only_when_failed
        CHECK ((status = 'failed') OR (error = ''))
);
-- +goose StatementEnd

-- Pagination is ordered by (uploaded_at, id): uploaded_at alone is not unique,
-- and a non-unique sort key makes cursor pagination skip or repeat rows at page
-- boundaries. The id tiebreaker is what makes the ordering total.
-- +goose StatementBegin
CREATE INDEX assets_collection_page_idx
    ON assets (collection_id, uploaded_at, id);
-- +goose StatementEnd

-- Supports the status and kind filters on listAssets. Two narrow indexes rather
-- than one composite, because the filters are independent and either may be
-- absent.
-- +goose StatementBegin
CREATE INDEX assets_collection_status_idx ON assets (collection_id, status);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX assets_collection_kind_idx ON assets (collection_id, kind);
-- +goose StatementEnd

-- The ingest pipeline claims work by scanning for pending assets across all
-- collections, so this one is deliberately not collection-scoped.
--
-- Note what is absent: no index on content_hash by itself. Cross-collection
-- "who else has this photo" is not a supported query, and indexing for it would
-- invite building it.
--
-- Also absent: a stored asset_count on collections. Collection.assetCount is
-- computed with COUNT(*) at read time. A denormalized counter would need
-- trigger or transaction discipline to stay correct, and COUNT(*) over an
-- indexed collection_id at 10,000 rows is not measurable.
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
