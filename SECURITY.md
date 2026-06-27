# Security Policy

## Supported Versions

RunWeaver is pre-1.0. Security fixes are published on the default branch and in
the latest release when releases are available.

## Reporting a Vulnerability

Please report vulnerabilities privately through GitHub Security Advisories for
the repository. Do not open a public issue with exploit details, secrets, or
private infrastructure information.

Include:

- affected RunWeaver version or commit;
- operating system and Go version;
- reproduction steps;
- impact and expected versus actual behavior;
- whether the issue affects generated repository metadata, runtime execution,
  credentials, or local files.

RunWeaver should never print credential values. Reports about leaked tokens,
unsafe generated permissions, path traversal, or unintended writes outside the
target repository are high priority.

## Dependency and Toolchain Checks

Release builds should use the Go toolchain declared in `go.mod` and pass:

```sh
go test ./...
go vet ./...
go test -race ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```
