//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config ../../../api/oapi-codegen.yaml -o generated.go ../../../api/openapi.yaml

// Package types contains the generated API types and server interfaces derived
// from api/openapi.yaml.
//
// Nothing in this package is hand-written and generated.go is not committed;
// it is produced by `make generate` (or `go generate ./internal/api/types`).
// Change the OpenAPI document, not the output.
package types

// Blank import so go.mod keeps github.com/oapi-codegen/runtime as a real,
// direct dependency even when generated.go has not been produced yet (fresh
// clone, dependabot's own internal `go mod tidy` when it opens a PR, or a
// contributor running `go mod tidy` before `make generate`). Once generated,
// generated.go imports this package itself for request/response binding
// helpers -- but until then there is nothing else in the checked-out source
// that references it, so `go mod tidy` sees it as unused and strips it. That
// strip is exactly what showed up as an unwanted diff in the hygiene job's
// go.mod-is-tidy check on dependabot PRs.
import _ "github.com/oapi-codegen/runtime"
