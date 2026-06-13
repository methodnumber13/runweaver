# Contributing

Thanks for improving RunWeaver.

## Development Setup

Install Go with the toolchain declared in `go.mod`, then run:

```sh
go test ./...
go vet ./...
go test -race ./...
go build ./cmd/runweaver
```

For a local binary:

```sh
./scripts/install.sh
```

## Code Style

- Keep the CLI in `cmd/runweaver`.
- Keep runtime-neutral orchestration in `internal/aitools`.
- Put reusable low-level helpers in the existing subpackages such as
  `foundation`, `modelconfig`, `runtimecatalog`, `runtimeenv`, and `statepath`.
- Keep generated runtime state under `.runweaver/tmp`.
- Do not print secrets. Report key names or credential sources only.
- Add or update tests for behavior changes.
- Public package comments must use idiomatic GoDoc. `quality_test.go` enforces
  package and exported declaration comment prefixes.

## Release Checks

Before opening a release PR, run:

```sh
gofmt -w $(rg --files -g '*.go')
go test ./...
go vet ./...
go test -cover ./...
go test -race ./...
go build ./cmd/runweaver
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Run the installer smoke test from the repository root:

```sh
tmpdir=$(mktemp -d)
RUNWEAVER_BIN_DIR="$tmpdir" ./scripts/install.sh
"$tmpdir/runweaver" help
rm -rf "$tmpdir"
```
