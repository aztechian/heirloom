# Heirloom API

This directory holds the **outward-facing API definitions**. `openapi.yaml` is the source of truth for the HTTP contract; the Go types and server interfaces in `internal/api/types/` are generated from it and are not committed.

The direction of authority matters and is easy to erode: change the spec, regenerate, then make the handlers compile. Never edit generated output, and never add a field to a response struct that the spec does not describe.

## Layout

```text
api/
  openapi.yaml        OpenAPI 3.0.3 document -- the contract
  oapi-codegen.yaml   generator configuration
  README.md           this file
```

Generated output lands in `internal/api/types/generated.go`, driven by the `//go:generate` directive in `internal/api/types/doc.go`.

The spec is deliberately a **single file** for now. Splitting paths and schemas across `$ref`'d files is more pleasant to maintain and is the eventual goal, but external `$ref` resolution interacts with `embedded-spec` bundling in ways that need to be verified on a working toolchain before we depend on it. Split it once `make generate` is green, not before.

## Regenerating

```bash
make tidy        # resolve dependencies and write go.sum (required on first checkout)
make generate    # regenerate internal/api/types/generated.go
make validate    # run the generator against the spec, discarding output
```

`make generate` is dependency-tracked on the spec, the generator config, and `go.sum`, so it re-runs when any of those change and no-ops otherwise. Every other target that needs Go code depends on it, which means a stale spec cannot silently survive a build.

`make validate` is a full generation run with the output discarded — it catches unresolvable `$ref`s and schemas the Go generator cannot express, and costs the same as generating. It is not a cheap syntax check.

## What the generator does and does not give us

Worth being precise about, because the whole justification for generating is that drift becomes a compile error, and there are two places where that is not true.

**Handlers are compiler-checked, except for uploads.** Strict mode gives each operation a typed request struct and typed responses, so a spec change that a handler does not satisfy fails to build. A `multipart/form-data` body has no typed struct, so `uploadAsset` reads the body itself; `UploadAssetRequest` in the spec documents that body but does not constrain the code. That handler needs tests where the others need only to compile.

**No validation is generated.** `pattern`, `minLength`, `maxLength`, `minimum`, `maximum`, and enum membership are contract, not code. `?limit=-1` and `?status=bogus` arrive at a handler well-typed and out of range. Schema `default` values are likewise not applied. So:

- Every constraint in the spec is a **handler obligation**, returning `422` on violation.
- Defaults (`limit` 50, `sensitivity` `normal`) must be applied in handler code.
- A shared validation helper per module is the right place for this, not scattered `if` statements — otherwise the gap between the documented contract and the enforced one widens quietly.

**Binding errors must be problem documents too.** The generator's default error handler writes `text/plain`, which would mean clients face two error formats: problem JSON from handlers and plain text from the router. Install an `ErrorHandlerFunc` that emits `application/problem+json` before any handler is wired up. Every operation declares a `default` response so a typed client can decode statuses added later.

## Conventions

**Versioned paths.** All routes live under `/api/v1/` deliberately: this repository is public and other people will run their own instances, so a breaking change costs strangers real pain. The version prefix is nearly free now and cannot be retrofitted gracefully.

**Assets are nested under collections.** `/api/v1/collections/{collectionSlug}/assets/{assetId}`, with no instance-wide `/assets/{id}` route. The nesting is a security measure rather than an aesthetic one: it puts the authorization scope in the path, so a handler cannot accidentally serve an asset from the wrong collection by looking up an asset in isolation. Given that collections are the tenancy boundary for unrelated families, that mistake is exactly the one worth designing out.

The corollary has to be implemented, not just intended: an asset that exists in a *different* collection returns **`404`, not `403`**. A `403` confirms the asset exists, which leaks one family's contents to another. Every asset lookup filters on `collectionSlug` in the query — never fetch by asset id alone and then compare.

**Errors are RFC 9457 `application/problem+json`.** A public API consumed by scripts needs machine-readable failures; branch on HTTP status and the `type` field, never on the prose in `detail`.

**Cursor pagination.** List endpoints take `limit` and `cursor` and return `nextCursor`. Cursors are opaque by contract, and bound to the query that produced them — reusing a cursor with different filters or a different `limit` returns `400`. `nextCursor` is always present in the response and is `null` on the last page; that null is the only end-of-list signal a client should trust. A page may be shorter than `limit` while more pages remain, so a short page proves nothing.

**Enums are closed and prefixed.** `always-prefix-enum-values` is on, so generated constants are unambiguous across schemas. Adding an enum value is a breaking change for strict clients; treat it as one. Note the flip side of closed enums: because the generator does not check membership, handlers must, or an unknown value becomes a typed-but-invalid input.

**Instance limits are discoverable.** `GET /api/v1/instance` reports `maxAssetBytes`, accepted media types, and whether authentication is enforced. A bulk ingest tool should read it first rather than learning the size limit from a `413` at the end of a 200 MB transfer.

## Collections

A collection is the isolation boundary. Every asset, person, and assertion belongs to exactly one. A single-family self-hosted instance is an instance with one collection — not a special mode, and not a separate code path. That is what lets one deployment serve several unrelated families without a second implementation.

`slug` is URL-safe, unique per instance, and derived from `name` when omitted. It is **immutable once set**, because it will appear in links handed to relatives, and those links live in inboxes for years. Collections have no separate opaque id: the slug *is* the identifier, which is what lets a collection's directory on storage be found directly, without a database mapping from an id to a location.

## Uploading assets

`POST /api/v1/collections/{collectionSlug}/assets` takes `multipart/form-data` with one file per request. Bytes stream through the application to the configured storage backend rather than being buffered — source scans run 50–200 MB, and buffering them is how a self-hosted instance on modest hardware falls over.

### Part ordering is contractual

Streaming imposes a constraint most multipart APIs do not have: `sha256`, `originalPath`, and `sensitivity` must arrive **before** the `file` part. The server reads parts in order and hands `file` straight to storage, so a metadata part behind the file could only be applied by buffering the file first, which defeats the reason for streaming. Out-of-order parts are rejected with `422`.

An unrecognized part name is also `422` rather than ignored, so a client typo fails loudly instead of silently dropping the metadata the client thinks it sent.

### Why proxied rather than presigned

Bytes pass through the app so that the same contract works over object storage and a plain filesystem, and so that revocation is immediate rather than bounded by a signature's lifetime. Presigned direct-to-S3 upload is a planned per-deployment option, not a different API: the storage layer is an interface with S3 and filesystem implementations, and adding a presigning capability means an optional method on that interface plus a negotiation step, not a new resource model. Designing the interface with that in mind now is what keeps it cheap later.

### Deduplication is the reason bulk ingest is safe

An asset's identity is the pair (`collectionSlug`, `contentHash`), so deduplication is **scoped to a collection**. Identical bytes uploaded into two collections produce two assets — two families may legitimately hold the same photograph, and neither should be able to learn that from the other's dedupe behavior.

Uploading content already present in the collection returns `200 OK` with the existing asset instead of `201 Created` with a new one. Nothing is created and nothing is modified.

This is what makes ingesting several thousand scans practical. A bulk script that dies partway through can simply be re-run: already-ingested files are no-ops, and there is no partial state to reconcile. The cost is that duplicate content is still transferred over the wire before being recognized. A cheap precheck endpoint that trades a hash for a yes/no before transferring would remove that cost, and is worth adding once ingest volume makes it hurt — it is an addition, not a redesign.

**The `200` path discards supplied metadata.** `sensitivity` and `originalPath` sent with a duplicate upload are dropped, because an upload is not an update. This is a sharp edge worth stating outright: a script that re-uploads material marked `restricted` will not raise the sensitivity of the existing asset. Check the status code and follow with `PATCH` when it is `200`.

Passing `sha256` in the upload is optional but recommended for scripted ingest. The server verifies it and rejects a mismatch with `422`. Without it, a truncated transfer is stored as a silently corrupt asset that looks fine until someone opens it years later, which is the worst possible failure mode for an archive.

`originalPath` is recorded verbatim and nothing consumes it yet. Capture it anyway: scan folder structure is frequently the only surviving record of how an album or box was organized, and it cannot be recovered once the physical grouping is broken up. It is constrained to a relative path with no `..` or empty segments — it is provenance, never used to build a filesystem path, but there is no reason to keep a traversal primitive lying around for whatever reads it in two years.

### Ingest is asynchronous

A successful upload means the bytes are durably stored. It does not mean derivatives exist, OCR has run, or faces have been detected. Clients poll the asset's `status`, or list by `status`, to observe progress. `failed` retains the bytes and is retryable — processing failure must never lose source material.

Media type is detected from content, not from the client-declared `Content-Type`, and the detected value wins. A client that lies about its content type gets `415`.

### Correcting an asset

`PATCH /api/v1/collections/{collectionSlug}/assets/{assetId}` changes `sensitivity` and `kind`, and nothing else.

It exists because both are wrong at upload time in practice. Material is recognized as private only once someone looks at it, and the dedupe path cannot set sensitivity at all — without this endpoint, anything ingested as `normal` would be permanently shareable, which contradicts the whole point of having the flag. `kind` is correctable because content detection will read a photographed document as a photo.

`contentHash`, `sizeBytes`, `mediaType`, `collectionSlug`, and the timestamps are immutable and rejected here. Source bytes are never rewritten, which is what keeps `contentHash` a stable identity; extracted metadata goes to XMP sidecars instead.

## Security posture, stated plainly

**There is no authentication in this version.** `operatorSession` and `csrfToken` are declared in the spec to fix the contract's shape. Neither is implemented — and note that the generator produces no enforcement from a `securityScheme` declaration in any case, so these will need real middleware, not a config flag.

That means an instance currently exposes an **unauthenticated file upload endpoint**. Anyone who can reach it can write arbitrary bytes into storage until the disk fills, or park content on someone else's server. Bind to localhost, and do not put an instance on the public internet until operator authentication and an upload size limit are both in place. This is the honest state of the project, not a footnote.

`GET /api/v1/instance` reports `authenticationEnabled`, so a health check or first-run screen can say this out loud rather than depending on the operator having read this file.

`csrfToken` is declared per-operation on writes rather than globally, so read operations do not appear to require a token they have no use for.

Two constraints already visible in the schema anticipate the eventual model. `sensitivity: restricted` marks material that must never appear in anything reachable by a share token, regardless of that token's grant — a filter that has to exist in the data model from the start, because retrofitting it means auditing every read path. And capability tokens, when they arrive, will be scoped to a slice of one collection rather than the archive, on the assumption that every such link eventually leaks.

## Adding an endpoint

1. Edit `openapi.yaml`. Give the operation an `operationId`; it becomes the generated method name. Declare a `default` response, and `security` per operation.
2. `make generate`.
3. Implement the generated interface method under `internal/api/<module>/`, following a precedent pf splitting `routes.go` for registration and `handlers.go` for logic.
4. Enforce, in the handler, every constraint the spec states — the generator enforces none of them.
5. Register the module's routes on the mux the generated `HandlerFromMux` is given. Paths in the spec already include the `/api/v1` prefix, so do **not** mount them under an `/api/v1` subrouter as well; that yields `/api/v1/api/v1/collections`.
6. `make lint test`.

Step 5's warning is about mux *nesting*, not about scoping middleware to API routes — those are different problems with different fixes. Do not reach for a subrouter to get API-only middleware; use `StdHTTPServerOptions.Middlewares` instead (see below).

### API-only middleware, without a subrouter

It's tempting to mount the generated handler under an `/api/v1/` subrouter specifically so API-only middleware (auth, CSRF, rate limiting) can wrap it without also running on static/UI routes. Don't: the generated handler already registers the full `/api/v1/...` paths on whatever mux it's given, so nesting it under another mux mounted at that same prefix duplicates it, exactly as step 5 says.

The generator gives you the scoping mechanism directly: `HandlerWithOptions(si, StdHTTPServerOptions{BaseRouter: mux, Middlewares: []MiddlewareFunc{...}})` wraps *each generated operation handler* individually with the given middlewares before registering it on `BaseRouter`, then registers the routes on that same top-level `mux` used for static files and `/healthz`. Global, cross-cutting middleware (request ID, logging, recovery) stays in `applyGlobalMiddleware`; anything that must never touch static assets goes in `Middlewares` instead. No path nesting, no double prefix.

Step 2 is what tells you what step 3 has to satisfy — for every operation except `uploadAsset`, a spec change you forget to implement fails the build rather than drifting quietly. For `uploadAsset`, write the test.

## Known-unverified

The generator has not been run against this spec. The document itself is validated — YAML parses, every internal `$ref` resolves, operation IDs are unique, and every `pattern` compiles as a Go regular expression — but `make generate` has not executed, so the following are unconfirmed:

- Whether `strict-server` produces a usable streaming shape for `uploadAsset`'s multipart body. It is documented to surface a `*multipart.Reader`. If the generated signature fights the streaming path, uncomment `exclude-operation-ids` in `oapi-codegen.yaml` and hand-write that single handler; the rest stays compiler-checked.
- Whether `minProperties` on `UpdateAssetRequest` survives generation in any useful form. Assume it does not and check for an empty patch in the handler.
- `go.sum` does not exist yet. Run `make tidy` before anything else.

Confirm these before building on top of this.
