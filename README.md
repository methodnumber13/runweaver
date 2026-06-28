# RunWeaver

Portable coding-agent orchestration utility for repositories.

This tool bootstraps repository-local workflow templates, package/file/symbol indexes, drift reports, generated profiles, runtime-specific agents/skills, and an optional read-only MCP stdio server for supported coding-agent runtimes. It does not auto-edit global MCP/runtime configs.

See [docs/COMPETITIVE_ANALYSIS.md](docs/COMPETITIVE_ANALYSIS.md) for the current comparison with adjacent AI coding agents, multi-agent frameworks, and context-engineering skill libraries.
See [docs/RUNTIME_ADAPTERS.md](docs/RUNTIME_ADAPTERS.md) for the adapter plan and current runtime support.
See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the internal package boundaries.

## Install

Prerequisites:

- Go must be installed locally. Releases and source builds should use the
  patched toolchain declared in `go.mod`.
- `runweaver` must be on a PATH visible to the selected coding runtime. The default install path is `~/.local/bin/runweaver`.
- Configure the selected runtime before AI classification or execution. OpenCode uses project/global OpenCode config or auth storage; Codex and Claude Code use their own CLI config/auth surfaces discovered by `runweaver doctor runtime`.

Install from the public Go module:

```sh
go install github.com/methodnumber13/runweaver/cmd/runweaver@latest
```

Install from a source checkout:

```sh
./scripts/install.sh
```

By default the script builds `runweaver` with Go and installs it to `~/.local/bin/runweaver`.

Override the destination:

```sh
RUNWEAVER_BIN_DIR=/usr/local/bin ./scripts/install.sh
```

If you keep the default destination, make sure your shell and selected runtime can see it:

```sh
export PATH="$HOME/.local/bin:$PATH"
which runweaver
```

## Project Status

RunWeaver is an early public alpha. The core CLI, repository indexer,
runtime-specific metadata renderers, workflow checkpointing, and OpenCode,
Codex, and Claude Code execution adapters are implemented and covered by tests.

The project is released under the MIT License. See [LICENSE](LICENSE).
Security reports should follow [SECURITY.md](SECURITY.md); contribution
guidelines are in [CONTRIBUTING.md](CONTRIBUTING.md).

## Quickstart

For a newly cloned repository:

```sh
runweaver doctor runtime --repo . --runtime all
runweaver init --repo . --force --classification ai --classifier-runtime opencode
runweaver doctor opencode --repo .
```

Then open the repository in the selected runtime and write the task to the generated `swarm` entrypoint. The swarm should create or resume a workflow run under `.runweaver/tmp/swarm-runs`, select repo-specific participants from the runtime profile, update `checkpoint.json`/`todo.md`, and continue phase by phase.

For multi-runtime metadata:

```sh
runweaver doctor runtime --repo . --runtime all
runweaver init --repo . --runtime all --force --classification deterministic
```

Current runtime status:

| Runtime | Discovery | Init/render | Execute |
| --- | --- | --- | --- |
| OpenCode | project/global/managed config and auth | `.opencode/agents`, `.opencode/skills`, `opencode.json` | `workflow run --execute` via `opencode run` |
| Codex | project/global/managed config, including `.codex/config.toml`, and auth | `AGENTS.md`, `.agents/skills`, `.codex/agents`, `.codex/runweaver/profile.json`; `.codex/config.toml` is discovered but not written by default | `workflow run --runtime codex --execute` via `codex exec --json` |
| Claude Code | project/global/managed settings and auth | `CLAUDE.md`, `.claude/agents`, `.claude/skills`, `.claude/runweaver/profile.json` | `workflow run --runtime claude --execute` via `claude --print --output-format stream-json` |

For explicit CLI execution instead of relying on Desktop/CLI default-agent behavior:

```sh
runweaver workflow run \
  --workflow .runweaver/workflows/feature-delivery-swarm.json \
  --task "implement task" \
  --execute
```

## Commands

```sh
runweaver init --repo .
runweaver init --repo . --force --classification ai
runweaver init --repo . --runtime all --force --classification deterministic
runweaver index --repo . --changed-only --prune
runweaver index clean --repo .
runweaver scan --repo . --out .runweaver/tmp/surface-index.json
runweaver classify --repo . --classification ai --apply
runweaver doctor model --repo .
runweaver doctor opencode --repo .
runweaver doctor runtime --repo . --runtime all
runweaver doctor processes --summary
runweaver refresh --repo .
runweaver refresh --repo . --apply
runweaver doctor --repo .
runweaver init --repo . --require-model
runweaver status --repo .
runweaver mcp serve --repo .
runweaver workflow run --workflow .runweaver/workflows/feature-delivery-swarm.json --task "implement task"
runweaver workflow run --workflow .runweaver/workflows/feature-delivery-swarm.json --task "implement task" --execute
runweaver workflow run --workflow .runweaver/workflows/feature-delivery-swarm.json --task "implement task" --runtime codex --execute
runweaver workflow run --workflow .runweaver/workflows/feature-delivery-swarm.json --task "implement task" --runtime claude --execute
runweaver workflow run --resume latest --execute
runweaver workflow run --resume latest --status
runweaver workflow verify --repo . --resume latest
```

CLI output is split by purpose:

- machine-readable JSON goes to stdout
- colored operator status/errors go to stderr when the terminal supports ANSI colors
- flag/usage errors are short and include the next command to run

`index` writes repo-local artifacts under `.runweaver/tmp/`:

```text
.runweaver/tmp/index/files.json
.runweaver/tmp/index/packages.json
.runweaver/tmp/index/symbols.json
.runweaver/tmp/index/edges.json
.runweaver/tmp/index/repo-index.json
.runweaver/tmp/index/repo-index.compact.json
.runweaver/tmp/index/repo-context.md
.runweaver/tmp/index/manifest.json
.runweaver/tmp/cache/<sha256>.json
```

The cache is content-hash based, so unchanged files are not reprocessed and moved files can reuse the same cached analysis. Use `--prune` to remove cache entries no longer referenced by the current repository scan and enforce the default cache cap. Use `runweaver index clean --repo .` to delete the disposable `.runweaver/tmp` index/cache completely.

| Command | Effect |
| --- | --- |
| `runweaver index --repo . --changed-only` | Rebuilds current index and reuses cache for unchanged files. |
| `runweaver index --repo . --prune` | Keeps current index but removes stale cache entries and enforces the cache cap. |
| `runweaver index clean --repo .` | Deletes all disposable `.runweaver/tmp` index/cache artifacts. |

Agents should prefer `repo-context.md` and `repo-index.compact.json` for prompt context. The full `repo-index.json` is still written for deeper inspection but should only be attached when compact context is insufficient.

If filesystem, cache, or package-file reads cannot be completed, `index` keeps going where possible and reports `warnings` plus `skipped` count in JSON instead of silently producing an incomplete result.

`init` is a smart bootstrap: it runs the deterministic index first, creates a `repo-intelligence-swarm` plan/checkpoint under `.runweaver/tmp/swarm-runs/`, then creates the initial profile, provider-specific agents/skills, and default workflows from the indexed technology/package evidence. Use `--runtime opencode`, `--runtime codex`, `--runtime claude`, or `--runtime all`. The shared workflow state is runtime-neutral and stays under `.runweaver/tmp`.

Existing project files are preserved by default:

- OpenCode project config is merged in place. If `opencode.jsonc`, `.opencode/opencode.json`, or `.opencode/opencode.jsonc` already exists, RunWeaver updates that file instead of creating a competing root `opencode.json`.
- Before RunWeaver changes an existing project config or instruction file, it writes a one-time `<file>.runweaver.bak` backup.
- Existing `AGENTS.md` and `CLAUDE.md` keep their content; RunWeaver adds or refreshes only the `<!-- BEGIN RUNWEAVER -->` managed block.
- Existing agents and skills without the RunWeaver generated marker are never overwritten, even with `--force`.
- `--force` refreshes files marked `generated by runweaver; safe to regenerate` and prunes stale generated metadata. It does not replace manual runtime config, manual agents, or manual skills.
- Global runtime configs and credentials are only discovered by doctor/preflight commands. `init` does not modify global OpenCode, Codex, or Claude config files.

## Classification Modes

`--classification deterministic` uses only local index/package/file evidence. It is stable and does not require a model.

`--classification ai` runs the model classifier through the selected classifier runtime and fails if the model classifier cannot return valid, validated JSON. Use it when you want domain-first agents and skills shaped by model analysis.

Classifier runtime defaults to OpenCode for backward compatibility. Select another runtime with `--classifier-runtime codex` or `--classifier-runtime claude`. OpenCode still supports `--classifier-provider openai-compatible`; Codex and Claude use their own CLI auth/config.

`--classification auto` attempts AI classification when possible and falls back to deterministic classification on model/preflight/output failure.

Examples:

```sh
runweaver classify --repo . --classification ai --classifier-runtime opencode --classifier-model openai-compatible/coder-model
runweaver classify --repo . --classification ai --classifier-runtime codex --classifier-model gpt-5.4 --classifier-skip-git-repo-check
runweaver classify --repo . --classification ai --classifier-runtime claude --classifier-model sonnet
runweaver classify --repo . --classification ai --apply --runtime all
runweaver init --repo . --runtime codex --force --classification ai --classifier-timeout 180s
runweaver refresh --repo . --apply --runtime all --classification auto
```

## OpenCode Desktop/CLI flow

`init` writes `opencode.json` with `default_agent: "swarm"` and project agents under `.opencode/agents`. OpenCode docs state `default_agent` applies across the TUI, `opencode run`, Desktop, and GitHub Action; the generated setup relies on that so a user can open the repo in either OpenCode Desktop or CLI and type the task to the repo-local `swarm` agent.

The generated `swarm` agent is responsible for:

1. running `runweaver index --repo . --changed-only --prune`
2. creating or resuming a durable workflow under `.runweaver/tmp/swarm-runs`
3. delegating phases through OpenCode's `task` tool
4. mirroring progress through `todowrite`
5. updating `checkpoint.json` so work can resume after context reset
6. running `runweaver workflow verify --repo . --resume latest` before final execution output

Run this after installation or when debugging a workstation:

```sh
runweaver doctor opencode --repo .
```

It checks local runtime metadata, resolved `default_agent`, `task`/`todowrite` readiness, project skills, `runweaver` availability on PATH, OpenCode debug output, and the OpenCode model preflight unless `--skip-model-check` is passed. If the selected provider credential is available only through `RUNWEAVER_MODEL_API_KEY` in the current shell, the doctor warns because a GUI-launched Desktop process may not inherit that environment.

`doctor model` checks whether OpenCode can use the configured model provider before an agent-driven init/run:

- project `opencode.json`
- global `~/.config/opencode/opencode.json` / JSONC variants
- `OPENCODE_CONFIG`, `OPENCODE_CONFIG_DIR`, `OPENCODE_CONFIG_CONTENT`
- managed config locations where relevant
- OpenCode auth storage and `RUNWEAVER_MODEL_API_KEY`

The command reports whether the effective provider, model, base URL, and credential are present without printing secret values. Config files are evaluated as merged OpenCode layers: global, custom path, project, custom directory, inline config, then managed config.

The default example uses an OpenAI-compatible provider through `@ai-sdk/openai-compatible`; provider base URLs usually end in `/v1` or `/v1-openai`. Provider IDs are not special-cased: if your OpenCode config names an internal OpenAI-compatible provider `company-llm`, pass `--provider company-llm` and keep the provider's own model IDs in `opencode.json`.

Minimal provider shape:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "openai-compatible/coder-model",
  "provider": {
    "openai-compatible": {
      "name": "OpenAI-compatible",
      "npm": "@ai-sdk/openai-compatible",
      "models": {
        "coder-model": {
          "name": "coder-model"
        }
      },
      "options": {
        "baseURL": "https://llm-provider.example.com/v1",
        "apiKey": "{env:RUNWEAVER_MODEL_API_KEY}"
      }
    }
  }
}
```

For Desktop, prefer OpenCode auth storage or a provider `apiKey` source that the GUI process can read. A key that exists only in the current terminal environment may work in CLI and fail in Desktop.

## Optional MCP Server

RunWeaver includes a small MCP stdio server. It is read-only by default:

```sh
runweaver mcp serve --repo .
```

The MCP server is intentionally in this repository instead of a separate package because it is a thin adapter over the same tested CLI/core functions. Split it out only if it needs an independent release cadence, hosted transport, or compatibility guarantees separate from the CLI.

The first MCP surface is read/status-oriented:

- `runweaver_status`: repository initialization, index, and latest workflow state.
- `runweaver_get_current`: `.runweaver/tmp/current.md` markdown resume surface.
- `runweaver_list_workflows`: generated workflow templates under `.runweaver/workflows`.
- `runweaver_verify_workflow`: deterministic verification for `latest` or an explicit run.

To let MCP-native clients create and update RunWeaver workflow state, start the server with explicit workflow-write authority:

```sh
runweaver mcp serve --repo . --allow-workflow-writes
```

This exposes only `.runweaver/tmp` workflow-state tools:

- `runweaver_plan_workflow`: creates `plan.json`, `checkpoint.json`, `todo.md`, `events.ndjson`, and `latest.json`.
- `runweaver_update_workflow`: updates checkpoint/todo/current workflow state.

These tools do not edit source files or runtime config files.

RunWeaver does not add MCP entries to user or project runtime configs during `init`. Connect it explicitly when you want the selected LLM client to see RunWeaver as tools instead of only files and commands.

Codex project or user config:

```toml
[mcp_servers.runweaver]
command = "runweaver"
args = ["mcp", "serve", "--repo", "."]
```

Claude Code project MCP config:

```json
{
  "mcpServers": {
    "runweaver": {
      "command": "runweaver",
      "args": ["mcp", "serve", "--repo", "."]
    }
  }
}
```

OpenCode local MCP config shape:

```json
{
  "mcp": {
    "runweaver": {
      "type": "local",
      "command": ["runweaver", "mcp", "serve", "--repo", "."]
    }
  }
}
```

## Process Diagnostics

Use this when VS Code shows many `Node.js Process` entries or when MCP/runtime processes look duplicated:

```sh
runweaver doctor processes --summary
```

Important fields:

- `supervisorCount`: active Codex/OpenCode parent processes.
- `duplicateGroups`: repeated MCP runtime groups such as Context7, GitHub, Memory, Playwright, or Sequential Thinking.
- `vscode`: whether VS Code helper/debugger processes are visible and whether Auto Attach appears noisy.
- `recommendations`: safe remediation steps. Close stale Codex/OpenCode sessions rather than killing child Node.js MCP processes one by one. If VS Code debugger UI is noisy, run `Debug: Toggle Auto Attach` and choose `Off`, or set `debug.javascript.autoAttachFilter` to `disabled`.

`refresh` writes local-only proposals to `.runweaver/tmp`. `refresh --apply` updates the selected runtime profile and generated repo-specific agents/skills; use `--runtime all` to refresh OpenCode, Codex, and Claude metadata together.

`workflow run` is implemented in Go and creates `plan.json`, `checkpoint.json`, `todo.md`, `events.ndjson`, and `latest.json` under `.runweaver/tmp/swarm-runs/`.

`workflow verify` is the deterministic run gate. It validates that:

- `plan.json`, `checkpoint.json`, `todo.md`, and `events.ndjson` exist and parse
- `todo.md` matches checkpoint phase state
- phase artifact files exist under `phases/<phase>/`
- completed/next/current phases are consistent with `plan.json`
- recorded participants stay within `maxParticipants`
- complete workflows include verification commands and verification results, or explicit blockers

The command prints JSON to stdout. `status: "warning"` keeps exit code 0 for in-progress runs; `status: "error"` exits non-zero after printing the checks.

`workflow run --execute` is the runtime adapter layer. It:

1. records runtime discovery/auth/config preflight, and checks OpenCode model readiness through `doctor model` logic only for `--runtime opencode` unless `--skip-model-check` is passed
2. creates or resumes the durable workflow plan
3. writes `<runtime>-exec-prompt.md` into the run directory
4. launches the selected runtime:
   - OpenCode: `opencode run --agent swarm --dir <repo> --format json`
   - Codex: `codex -a never exec --json --ephemeral -C <repo> --sandbox workspace-write`
   - Claude Code: `claude --print --output-format stream-json --permission-mode dontAsk`
5. points the runtime at `plan.json`, `checkpoint.json`, `todo.md`, the runtime profile, `repo-context.md`, `repo-index.compact.json`, and `manifest.json` through the execution prompt or provider-native attachment flags
6. writes stdout/stderr to `<runtime>-stdout.jsonl` and `<runtime>-stderr.log`
7. performs a deterministic post-check that warns if the runtime finished but `checkpoint.json` did not advance

Useful execution flags:

```sh
runweaver workflow run --task "implement task" --execute --dry-run --skip-model-check
runweaver workflow run --resume latest --execute
runweaver workflow run --task "implement task" --execute --model openai-compatible/<model-id>
runweaver workflow run --task "implement task" --execute --provider openai-compatible
runweaver workflow run --task "implement task" --execute --attach http://localhost:4096
runweaver workflow run --task "implement task" --runtime codex --execute --sandbox workspace-write --approval-policy never
runweaver workflow run --task "implement task" --runtime claude --execute --permission-mode dontAsk
```

Workflow resume state:

- `plan.json`: immutable workflow/task plan for the run.
- `checkpoint.json`: durable state used after context resets.
- `todo.md`: human-readable phase status mirror.
- `events.ndjson`: append-only lifecycle log.
- `latest.json`: pointer to the latest run.
- `phases/<phase>/handoff.md`: compact phase handoff.
- `phases/<phase>/notes.md`: longer evidence notes.
- `phases/<phase>/verification.jsonl`: exact verification command/result records.

`checkpoint.json` stores concise recovery fields:

- `participants` and `participantRationale`
- `findings` and `decisions`
- `filesRead` and `filesChanged`
- `artifacts`
- `verification` and `verificationResults`
- `blockers`
- `nextAction`

Generated workflows include `maxParticipants`. The swarm prompt tells the agent to prefer one domain owner plus up to two reviewers/skills, and only exceed that when the workflow explicitly allows it.

`workflow run --resume latest --status` is for inspection. `workflow update` is mostly for manual debugging and external automation; the generated swarm agent should update checkpoints itself while executing phases. Resume paths are intentionally confined to `.runweaver/tmp/swarm-runs`.

Example checkpoint update:

```sh
runweaver workflow update --repo . --resume latest \
  --phase verify \
  --status in_progress \
  --participants "domain-owner-agent,repo-test-quality-reviewer" \
  --participant-rationale "domain owner plus focused test reviewer" \
  --file-read src/auth/auth.guard.ts \
  --file-changed test/unit/auth.guard.spec.ts \
  --finding "public route bypass is handled before token validation" \
  --decision "add focused @Public regression" \
  --verification "npm run test -- auth.guard.spec.ts" \
  --verification-result "passed" \
  --complete-phase
```
