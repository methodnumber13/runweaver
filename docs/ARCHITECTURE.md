# Architecture

RunWeaver is intentionally split into a small public CLI and an internal repository-intelligence/orchestration engine.

## Packages

```text
cmd/runweaver
  CLI parsing, colored output, progress rendering, and command wiring.

internal/aitools
  Orchestration facade. Keeps the CLI-facing API stable and owns the modules that are still tightly coupled through shared profile, workflow, index, and classifier types.

internal/aitools/foundation
  Low-level filesystem, JSON, repository root, list, and path helpers.

internal/aitools/jsonc
  JSONC comment stripping used by config readers.

internal/aitools/modelconfig
  OpenCode-compatible model/provider/auth discovery.

internal/aitools/processdiag
  Runtime process diagnostics and VS Code debugger noise detection.

internal/aitools/runtimecatalog
  Runtime provider catalog and stable runtime ordering.

internal/aitools/runtimecatalog/{opencode,codex,claude}
  Provider-specific metadata: binary names, generated paths, capabilities, profile paths, and delegation guidance.

internal/aitools/runtimeenv
  Runtime-specific execution environment helpers.

internal/aitools/statepath
  Canonical `.runweaver` state, workflow, index, and cache paths.
```

## Current Facade Boundary

The remaining files in `internal/aitools` are grouped around the core orchestration loop:

- `scan`, `index`, and `index_analysis` collect repository evidence.
- `classification`, `classifier`, and `profile` turn evidence into agents, skills, and runtime profiles.
- `materialize`, `runtime_*`, and `runtime_baseline` write provider metadata.
- `workflow`, `executor`, and `drift` maintain durable run state and verification.
- `init` and `refresh` compose the above steps into user-facing workflows.

These modules still share core types from `types.go` and workflow/profile structs. Moving them into deeper packages should happen after those shared structs are promoted into a lower-level core package; otherwise the code would either create import cycles or replace one large package with many packages coupled through unstable wrappers.

## Refactoring Rule

New code should start outside the facade when it has a stable boundary:

- path constants and filesystem helpers go into `foundation` or `statepath`;
- config parsing and auth discovery go into `modelconfig`;
- runtime provider metadata goes into `runtimecatalog/<provider>`;
- runtime execution environment helpers go into `runtimeenv`;
- process inspection goes into `processdiag`.

Only orchestration code that needs several core structs at once should remain directly in `internal/aitools`.

