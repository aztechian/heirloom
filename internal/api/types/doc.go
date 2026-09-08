//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config ../../../api/oapi-codegen.yaml -o generated.go ../../../api/openapi.yaml

// Package types contains the generated API types and server interfaces derived
// from api/openapi.yaml.
//
// Nothing in this package is hand-written and generated.go is not committed;
// it is produced by `make generate` (or `go generate ./internal/api/types`).
// Change the OpenAPI document, not the output.
package types
