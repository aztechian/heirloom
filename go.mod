module github.com/aztechian/heirloom

// Kept in step with .tool-versions on purpose. When these drift, CI (which
// resolves the toolchain from go.mod) passes while a local asdf build fails,
// and the failure looks like a code problem rather than a version problem.
go 1.27.1

// Direct dependencies. The versions below could not be resolved against the
// module proxy in the environment this was written in, so they are last-known
// rather than verified, and go.sum has no entries for them or for the indirect
// graph they pull in — so nothing that imports internal/store builds yet. First
// action with a real toolchain:
//
//	go get -u github.com/pressly/goose/v3 modernc.org/sqlite github.com/jackc/pgx/v5
//	go mod tidy
//
// Until that is committed, CI fails on missing go.sum entries and on the tidy
// check, which is the correct signal rather than a bug to work around.
require (
	github.com/getkin/kin-openapi v0.135.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.0
	// Test-only today: imported by internal/store's Postgres migration test so
	// that dialect drift is caught in CI. store.Open still refuses the postgres
	// dialect -- the driver was never the blocker.
	github.com/jackc/pgx/v5 v5.7.4
	github.com/oapi-codegen/runtime v1.7.0
	github.com/pressly/goose/v3 v3.24.3
	github.com/rs/zerolog v1.35.1
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	modernc.org/sqlite v1.37.0
)

require (
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/dprotaso/go-yit v0.0.0-20220510233725-9ba8df137936 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-openapi/jsonpointer v0.22.4 // indirect
	github.com/go-openapi/swag/jsonname v0.25.4 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jstemmer/go-junit-report/v2 v2.1.0 // indirect
	github.com/mailru/easyjson v0.9.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/oapi-codegen/oapi-codegen/v2 v2.7.1 // indirect
	github.com/oasdiff/yaml v0.0.9 // indirect
	github.com/oasdiff/yaml3 v0.0.9 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sethvargo/go-retry v0.3.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/speakeasy-api/jsonpath v0.6.3 // indirect
	github.com/speakeasy-api/openapi v1.19.2 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/vmware-labs/yaml-jsonpath v0.3.2 // indirect
	github.com/woodsbury/decimal128 v1.4.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/exp v0.0.0-20250506013437-ce4c2cf36ca6 // indirect
	golang.org/x/mod v0.33.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	golang.org/x/tools v0.42.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.65.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.10.0 // indirect
)

tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

tool github.com/jstemmer/go-junit-report/v2
