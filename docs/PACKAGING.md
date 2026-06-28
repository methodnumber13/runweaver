# Packaging And Releases

RunWeaver is distributed as a Go CLI. Release packaging is intentionally split into a stable core path and optional package-manager channels.

## Versioning

RunWeaver uses SemVer tags:

- `v0.x.y` while runtime metadata schemas, workflow contracts, and MCP tools are still stabilizing.
- `v1.0.0` only after the CLI flags, generated layout, workflow checkpoint schema, and supported runtime adapters are considered stable.
- Patch releases should not change generated file layout or workflow schema semantics.
- Minor releases may add commands, fields, runtime adapters, MCP tools, or generated metadata.

Public compatibility surfaces:

- CLI command names, flags, and JSON output.
- `.runweaver/workflows/*.json` workflow shape.
- `.runweaver/tmp/swarm-runs/*/checkpoint.json` resume fields.
- Generated AGENTS/CLAUDE/OpenCode/Codex metadata contracts.
- MCP tool names and input/output schemas.

## GitHub Releases

The canonical binary release path is GitHub Releases produced from version tags:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The release workflow runs tests, vet, and GoReleaser. The regular CI workflow
uses the project toolchain from `go.mod`; the release workflow uses stable Go
so the GoReleaser action has a current release toolchain. GoReleaser builds:

- macOS arm64/amd64
- Linux arm64/amd64
- Windows arm64/amd64

Release binaries receive build metadata through ldflags:

```sh
runweaver version
runweaver version --json
```

## Go Install

Go module installation remains the simplest source-based fallback:

```sh
go install github.com/methodnumber13/runweaver/cmd/runweaver@latest
```

`go install` may report the module version through Go build info. Tagged GitHub Release binaries include richer commit/date metadata.

## Homebrew

Homebrew should be added after a tap exists, preferably:

```text
methodnumber13/homebrew-tap
```

Expected user install shape:

```sh
brew install methodnumber13/homebrew-tap/runweaver
```

Do not make Homebrew tap publishing part of the default release workflow until the tap repository and a dedicated cross-repository token are configured. Otherwise a valid GitHub Release could fail because formula publishing lacks permission.

When enabling the tap:

1. Create the tap repository.
2. Add a CI secret such as `TAP_GITHUB_TOKEN` with permission to commit formula updates to the tap.
3. Add the GoReleaser Homebrew publisher or a separate formula update workflow.
4. Test formula install from a fresh machine before announcing Homebrew as stable.

## npm

npm is useful for JavaScript-heavy users, but RunWeaver should not use a normal-path `postinstall` downloader.

Recommended package shape:

- Top-level package: `@runweaver/cli` or `@methodnumber13/runweaver`.
- Executable: `runweaver`.
- Thin JS wrapper in `bin/runweaver.js`.
- Platform packages as optional dependencies:
  - `@runweaver/darwin-arm64`
  - `@runweaver/darwin-x64`
  - `@runweaver/linux-arm64`
  - `@runweaver/linux-x64`
  - `@runweaver/win32-arm64`
  - `@runweaver/win32-x64`

Each platform package should include the compiled binary and constrain `os`/`cpu`. The top-level wrapper resolves the installed platform package and forwards argv to the binary. If optional dependencies are omitted, it should fail with a clear message and point users to GitHub Releases or `go install`.

Publishing requirements before npm is enabled:

- Reserve the package scope.
- Publish from CI with provenance/trusted publishing when available.
- Keep package contents minimal and audited.
- Keep GitHub Releases and `go install` documented as fallbacks.

## Local Source Install

From a checkout:

```sh
./scripts/install.sh
```

The installer writes to `~/.local/bin/runweaver` by default and injects git-derived version metadata when the checkout has git history.
