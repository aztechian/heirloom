# Project metadata
PROJECT_NAME := heirloom
PROJECT_TAGLINE := Family Archive
PROJECT_DESCRIPTION := Catalog scanned family photographs and documents
PROJECT_VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# API definitions and generated output
API_SPEC := api/openapi.yaml
API_CODEGEN_CONFIG := api/oapi-codegen.yaml
API_TYPES_FILE := internal/api/types/generated.go

SERVER_SOURCES := $(shell find internal cmd -type f -name "*.go" 2>/dev/null) go.mod $(wildcard go.sum)

# Frontend build output, embedded into the binary. Populated once the frontend
# exists; see the note on the `all` target below.
STATIC_DIR := static
STATIC_STAMP := $(STATIC_DIR)/.build-stamp
FRONTEND_SOURCES := $(shell find frontend -type f ! -path "*/node_modules/*" 2>/dev/null)

.PHONY: all tidy generate validate lint test junit clean help

all: generate $(STATIC_STAMP) $(PROJECT_NAME)

help:
	@echo "$(PROJECT_NAME) $(PROJECT_VERSION)"
	@echo ""
	@echo "  make tidy       Resolve module dependencies and write go.sum"
	@echo "  make generate   Generate Go types and server interfaces from $(API_SPEC)"
	@echo "  make validate   Run the generator against the spec, discarding output"
	@echo "  make lint       Run golangci-lint (requires generated code)"
	@echo "  make test       Run Go tests"
	@echo "  make junit      Run Go tests, writing report.xml for CI"
	@echo "  make clean      Remove generated and built artifacts"

# go.sum is committed. This rule exists so that editing go.mod by hand -- which
# happens when adding a dependency without network access to resolve it -- does
# not leave every downstream target failing with a module error.
#
# Note the limitation: a fresh `git clone` or `actions/checkout` gives every file
# the same mtime, so make does not consider go.sum stale and will not tidy. Do
# not rely on this rule to repair a go.sum that was never generated; run
# `make tidy` explicitly.
tidy: go.sum

go.sum: go.mod
	@echo "Resolving module dependencies..."
	@go mod tidy

# Generate API types and server interfaces from the OpenAPI spec.
generate: $(API_TYPES_FILE)

$(API_TYPES_FILE): $(API_SPEC) $(API_CODEGEN_CONFIG) internal/api/types/doc.go go.sum
	@echo "Generating API types from $(API_SPEC)..."
	@go generate ./internal/api/types

# Exercise the spec through the real generator and throw the output away. This
# catches everything a build would catch -- unresolvable $refs, schemas the Go
# generator cannot express -- without touching the working tree. It is not a
# lightweight syntax check; it is a full generation run, and it costs the same.
validate: go.sum
	@echo "Validating $(API_SPEC)..."
	@go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen \
		-config $(API_CODEGEN_CONFIG) -o /dev/null $(API_SPEC)
	@echo "OK"

frontend/node_modules: frontend/package.json
	@cd frontend && npm install

# Directory mtimes are not a reliable build signal -- writing a file into an
# existing directory may not bump it -- so gate on a stamp file instead.
$(STATIC_STAMP): $(FRONTEND_SOURCES) frontend/node_modules
	@cd frontend && VITE_APP_NAME=$(PROJECT_NAME) \
		VITE_APP_TAGLINE="$(PROJECT_TAGLINE)" \
		VITE_APP_DESCRIPTION="$(PROJECT_DESCRIPTION)" \
		VITE_APP_VERSION="$(PROJECT_VERSION)" \
		npm run build
	@mkdir -p $(STATIC_DIR) && touch $@

# Build the server with the frontend embedded. CGO stays disabled so the output
# is a fully static, cross-compilable binary -- see README open decisions, this
# constraint is what rules out several OCR and face-recognition libraries.
$(PROJECT_NAME): $(SERVER_SOURCES) $(API_TYPES_FILE) $(STATIC_STAMP)
	@echo "Compiling $(PROJECT_NAME)..."
	@CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(PROJECT_VERSION)" \
		-o $(PROJECT_NAME) ./cmd/server

lint: $(API_TYPES_FILE)
	@golangci-lint run

test: $(API_TYPES_FILE)
	@go test -race -coverprofile=coverage.out ./...

# CI entry point. Phony because a stale report.xml on disk must never be
# mistaken for a passing run.
junit: $(API_TYPES_FILE)
	(go test -v -race -coverprofile=coverage.out ./... 2>&1 || true) \
		| go tool go-junit-report --set-exit-code > report.xml

clean:
	rm -rf $(PROJECT_NAME) $(STATIC_DIR) $(API_TYPES_FILE) coverage.out report.xml
