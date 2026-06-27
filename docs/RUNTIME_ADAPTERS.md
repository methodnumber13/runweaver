# Runtime Adapters

## Goal

RunWeaver should become runtime-neutral:

> one repository-intelligence core, multiple coding-agent renderers and executors.

OpenCode, Codex, and Claude Code are executable runtimes. The core should not depend on one runtime's paths or command semantics.

Current implementation status:

| Runtime | Discovery | Init/render | Execute |
| --- | --- | --- | --- |
| OpenCode | implemented | implemented | implemented |
| Codex | implemented | implemented | implemented |
| Claude Code | implemented | implemented | implemented |

## Core Versus Runtime

Keep these parts runtime-independent:

- repository root resolution;
- package/file/symbol/edge indexing;
- semantic classification;
- generated swarm profile;
- workflow specs;
- durable run state: `plan.json`, `checkpoint.json`, `todo.md`, `events.ndjson`;
- drift detection;
- verification gates;
- process diagnostics where possible.

Move these behind runtime adapters:

- generated instruction files;
- generated agent files;
- generated skill files;
- runtime config files;
- model/auth doctor logic;
- execution command;
- subagent delegation instructions;
- permission/sandbox defaults;
- runtime-specific logs and attach behavior.

## Adapter Shape

The Go implementation is split into two runtime layers:

- `internal/aitools/runtimecatalog` contains runtime-neutral catalog types, ID normalization, and stable ordering.
- `internal/aitools/runtimecatalog/opencode`, `internal/aitools/runtimecatalog/codex`, and `internal/aitools/runtimecatalog/claude` contain provider-specific metadata: binary names, generated paths, capabilities, profile path, and delegation guidance.
- `internal/aitools` contains the orchestration-facing adapter facade. It binds catalog metadata to rendering, discovery, classifier, and workflow execution code that depends on core RunWeaver types such as `Profile`, `WorkflowExecuteOptions`, and `ClassifyOptions`.

This keeps the folder layout explicit without forcing runtime packages to import the whole orchestration layer.

The facade adapter contract remains:

```go
type RuntimeAdapter interface {
    ID() string
    Provider() RuntimeProvider
    ProfilePath() string
    GeneratedPaths() []string
    Capabilities() map[string]RuntimeFlag
    PathChecks(root string) (configs, auth, metadata, managed []RuntimeFileCheck)
    MaterializeProfile(root string, profile Profile, force bool) error
    ExecutionSpec(root string, opts WorkflowExecuteOptions, plan WorkflowRunSummary, prompt string) (workflowRuntimeExecutionSpec, error)
    ClassifierSpec(root, prompt string, opts ClassifyOptions, finalOutputPath string) (classifierCommandSpec, error)
    DelegationGuidance() string
}
```

The CLI exposes:

```sh
runweaver init --repo . --runtime opencode
runweaver init --repo . --runtime codex
runweaver init --repo . --runtime claude
runweaver init --repo . --runtime all

runweaver doctor runtime --repo . --runtime codex
runweaver workflow run --repo . --runtime claude --execute
```

Default runtime is `opencode` to preserve existing behavior. Codex and Claude Code also support provider discovery, generated metadata, AI classification, and workflow execution.

## OpenCode Adapter

Current adapter behavior:

- writes `opencode.json`;
- writes `.opencode/agents/*.md`;
- writes `.opencode/skills/*/SKILL.md`;
- writes `.runweaver/workflows/*.json`;
- writes `.opencode/swarm/profile.json`;
- executes `opencode run --agent swarm --dir <repo> --format json`;
- writes `opencode-exec-prompt.md`, `opencode-stdout.jsonl`, and `opencode-stderr.log`;
- relies on OpenCode primary agents, subagents, task delegation, permissions, and `todowrite`.

Static provider metadata is implemented by `runtimecatalog/opencode`. Runtime behavior is implemented by `runtime_opencode.go`; behavior is covered by runtime-aware command specs and tests.

## Codex Adapter

Codex support targets these surfaces:

- `AGENTS.md` for durable project instructions;
- `.agents/skills/<name>/SKILL.md` for repo-local skills;
- optional `.agents/skills/<name>/agents/openai.yaml` for Codex app metadata and invocation policy;
- `.codex/agents/*.toml` for project-scoped custom subagents;
- optional `.codex/config.toml` additions only when the user asks for project-local Codex config;
- `codex exec --json` for non-interactive execution;
- `codex -a never exec --json --ephemeral -C <repo> --sandbox workspace-write` when execution needs edits;
- `codex -a never exec --json --output-last-message <file>` for AI classification through Codex.

Important Codex constraints:

- Codex reads `AGENTS.md` at session start, so generated guidance must be concise and restart-aware.
- Codex skills use progressive disclosure, so skill descriptions must be specific enough to trigger without flooding the initial context.
- Codex only spawns subagents when explicitly asked, so the root guidance must say when to spawn agents.
- Codex subagents live in `.codex/agents/*.toml` and require `name`, `description`, and `developer_instructions`.
- Subagents inherit sandbox policy, so RunWeaver should avoid surprising permission escalation and should document required sandbox mode.

Codex static provider metadata is implemented by `runtimecatalog/codex`. Codex generated layout is implemented for `runweaver init --runtime codex` and `--runtime all`:

```text
AGENTS.md
.agents/skills/context-discipline/SKILL.md
.agents/skills/metadata-refresh/SKILL.md
.agents/skills/repo-onboarding/SKILL.md
.agents/skills/<repo-specific-skill>/SKILL.md
.codex/agents/swarm.toml
.codex/agents/<repo-specific-agent>.toml
.codex/runweaver/profile.json
```

Codex execution prompts instruct the root session to:

1. read `AGENTS.md`, `.codex/runweaver/profile.json`, and `.runweaver/tmp/index/repo-context.md`;
2. create or resume `.runweaver/tmp/swarm-runs/latest.json`;
3. spawn named Codex subagents for selected participants;
4. update checkpoints with `runweaver workflow update`;
5. run `runweaver workflow verify`;
6. finish only when workflow verification is clean or a blocker is recorded.

Codex execution artifacts:

```text
.runweaver/tmp/swarm-runs/<run-id>/codex-exec-prompt.md
.runweaver/tmp/swarm-runs/<run-id>/codex-stdout.jsonl
.runweaver/tmp/swarm-runs/<run-id>/codex-stderr.log
.runweaver/tmp/swarm-runs/<run-id>/codex-final-message.md
```

The shared tmp path is `.runweaver/tmp`. Workflow resume is intentionally confined to `.runweaver/tmp/swarm-runs`.

## Claude Code Adapter

Claude Code support targets these surfaces:

- `CLAUDE.md` for project guidance;
- `.claude/agents/*.md` for project-scoped subagents;
- `.claude/skills/<name>/SKILL.md` for project-local skills;
- optional `.claude/settings.json` only for explicit permission/hook configuration;
- `claude --print --output-format stream-json --permission-mode dontAsk` for adapter-run workflows;
- `claude --print --output-format text` for AI classification through Claude;
- native dynamic workflows only as an optional exporter, not as the core state store.

Important Claude constraints:

- Claude subagents are Markdown files with YAML frontmatter and body instructions.
- Project subagents in `.claude/agents/` are the natural equivalent of OpenCode `.opencode/agents`.
- Claude can auto-delegate when descriptions match, but high-stakes workflow execution should still be explicit.
- Claude dynamic workflows are powerful, but they are vendor-native. RunWeaver should keep its own `plan.json`/`checkpoint.json` as the portable source of truth.

Claude Code static provider metadata is implemented by `runtimecatalog/claude`. Claude generated layout is implemented for `runweaver init --runtime claude` and `--runtime all`:

```text
CLAUDE.md
.claude/agents/swarm.md
.claude/agents/<repo-specific-agent>.md
.claude/skills/context-discipline/SKILL.md
.claude/skills/metadata-refresh/SKILL.md
.claude/runweaver/profile.json
```

Claude execution artifacts:

```text
.runweaver/tmp/swarm-runs/<run-id>/claude-exec-prompt.md
.runweaver/tmp/swarm-runs/<run-id>/claude-stdout.jsonl
.runweaver/tmp/swarm-runs/<run-id>/claude-stderr.log
```

## Migration Order

1. Done: introduce `--runtime opencode|codex|claude|all` without changing existing OpenCode behavior.
2. Done: add provider registry and cross-platform provider discovery for project/global/managed config surfaces.
3. Done: add Codex renderer because Codex has clear repo-local `AGENTS.md`, `.agents/skills`, and `.codex/agents` surfaces.
4. Done: add Claude renderer for `CLAUDE.md`, `.claude/agents`, and `.claude/skills`.
5. Done: add runtime-aware workflow execution command specs.
6. Done: add Codex executor through `codex exec --json`.
7. Done: add Claude executor through `claude --print --output-format stream-json`.
8. Done: move shared state from runtime-local tmp assumptions to `.runweaver/tmp` and remove legacy runtime-local resume fallback.
9. Done: add runtime-aware AI classifier command specs for OpenCode, Codex, and Claude.
10. Done: introduce an internal `RuntimeAdapter` contract with OpenCode, Codex, and Claude implementations.
11. Done: make drift scanning and generated workflow prompts runtime-aware instead of OpenCode-only.
12. Done: split runtime provider metadata into `runtimecatalog/{opencode,codex,claude}` subpackages.
13. Next: move renderer/executor behavior into provider packages only after the shared `Profile` and workflow command-spec types are promoted into a lower-level core package.
14. Next: add golden generated-file snapshots once renderer formats stabilize.

## Sources

- OpenCode agents documentation: https://opencode.ai/docs/agents/
- OpenCode config documentation: https://opencode.ai/docs/config/
- Codex manual: https://developers.openai.com/codex/codex-manual.md
- Claude Code subagents documentation: https://code.claude.com/docs/en/sub-agents
- Claude Code skills documentation: https://code.claude.com/docs/en/skills
- Claude Code dynamic workflows documentation: https://code.claude.com/docs/en/workflows
