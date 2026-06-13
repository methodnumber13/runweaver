package aitools

const contextDisciplineSkill = `---
name: context-discipline
description: "Filesystem-backed context discipline for durable OpenCode swarm runs"
compatibility: opencode
---

Use this skill for every non-trivial coding, bugfix, refactor, test, onboarding, or metadata workflow.

## Purpose

Keep model context small while preserving enough filesystem-backed state for another swarm pass to resume without guessing.

## Locked, Editable, Append-Only, Human-Controlled Surfaces

- Locked: ` + "`plan.json`" + `, workflow specs under ` + "`.runweaver/workflows`" + ` during a run, and the original user task. Do not rewrite these to make verification easier.
- Editable through CLI only: ` + "`checkpoint.json`" + ` and ` + "`todo.md`" + `. Update them with ` + "`runweaver workflow update`" + ` so phase state and events stay consistent.
- Append-only: ` + "`events.ndjson`" + ` and phase ` + "`verification.jsonl`" + ` logs. Add new entries; do not erase history.
- Human-controlled: source code intent, secrets, destructive commands, deploys, and broad metadata regeneration. Ask or record a blocker when authority is unclear.

## Run Artifact Layout

Use files under ` + "`.runweaver/tmp/swarm-runs/<run-id>/`" + `:

- ` + "`plan.json`" + `: immutable workflow/task intent.
- ` + "`checkpoint.json`" + `: durable state for resume.
- ` + "`todo.md`" + `: human-readable phase mirror.
- ` + "`events.ndjson`" + `: lifecycle event log.
- ` + "`phases/<phase>/handoff.md`" + `: compact phase handoff and current conclusion.
- ` + "`phases/<phase>/notes.md`" + `: evidence notes that are too long for checkpoint.
- ` + "`phases/<phase>/verification.jsonl`" + `: exact verification command/result records.
- ` + "`agents/<participant>/`" + `: optional participant-specific notes when delegation produces useful handoff material.

## Checkpoint Fields To Maintain

Record concise entries for:

- ` + "`participants`" + ` and ` + "`participantRationale`" + `
- ` + "`findings`" + ` and ` + "`decisions`" + `
- ` + "`filesRead`" + ` and ` + "`filesChanged`" + `
- ` + "`artifacts`" + `
- ` + "`verification`" + ` and ` + "`verificationResults`" + `
- ` + "`blockers`" + ` when work cannot safely continue
- ` + "`nextAction`" + ` when the workflow is not complete

Prefer one domain owner plus up to two reviewers/skills. Do not exceed workflow ` + "`maxParticipants`" + ` unless the workflow explicitly requires it.

## Verification

Before a final response for an execution workflow, run:

` + "`runweaver workflow verify --repo . --resume latest`" + `

Fix warnings when feasible. If a warning cannot be fixed safely, record a blocker and explain the residual risk.

## Output Contract

Return ` + "`files_read`, `files_changed`, `checkpoint_fields_updated`, `artifacts`, `verification_results`, `blockers`, and `workflow_verify`." + `
`

const metadataRefreshSkill = `---
name: metadata-refresh
description: "Refresh repository runtime metadata after code changes"
compatibility: opencode
---

1. Run ` + "`runweaver refresh --repo .`" + `.
2. Inspect .runweaver/tmp/index/repo-context.md, .runweaver/tmp/index/repo-index.compact.json, .runweaver/tmp/index/manifest.json, .runweaver/tmp/surface-index.json, .runweaver/tmp/drift-report.json, and .runweaver/tmp/profile.generated.json.
3. Apply profile/agent/skill changes only after review.

## Output Contract

Return ` + "`surface_index_path`, `drift_report_path`, `profile_changes`, and `checkpoint_update`." + `
`

const repoOnboardingSkill = `---
name: repo-onboarding
description: "Onboard into a repository by reading configs, source layout, tests, and runtime metadata"
compatibility: opencode
---

1. Read ` + "`AGENTS.md`" + `.
2. Run ` + "`runweaver index --repo . --changed-only --prune`" + `.
3. Inspect detected package roles, tools, entrypoints, configs, tests, routes/controllers/pages, symbols, and verification commands.

## Output Contract

Return ` + "`task_map`, `important_files`, `risks`, `verification`, and `checkpoint_update`." + `
`
