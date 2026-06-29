# Contributing

Thanks for improving RunWeaver.

## Development Setup

Install Go with the toolchain declared in `go.mod`, then run:

```sh
go test ./...
go vet ./...
go test -race ./...
go build -o /tmp/runweaver ./cmd/runweaver
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
go build -o /tmp/runweaver ./cmd/runweaver
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Run the installer smoke test from the repository root:

```sh
tmpdir=$(mktemp -d)
RUNWEAVER_BIN_DIR="$tmpdir" ./scripts/install.sh
"$tmpdir/runweaver" version
"$tmpdir/runweaver" help
"$tmpdir/runweaver" smoke codex --keep
rm -rf "$tmpdir"
```

When Codex is configured locally, also run a live smoke on a disposable repo:

```sh
runweaver smoke codex --live --timeout 4m --keep
```

For tagged releases:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The release workflow uses GoReleaser to publish GitHub Release archives. Run
`goreleaser check` locally when GoReleaser is installed. See
[docs/PACKAGING.md](docs/PACKAGING.md) before enabling additional package
manager channels such as Homebrew or npm.
