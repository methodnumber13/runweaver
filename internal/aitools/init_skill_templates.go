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
- Editable through CLI only: ` + "`checkpoint.json`" + ` and ` + "`todo.md`" + `. Update them with ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" workflow update`" + ` so phase state and events stay consistent.
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

Record concise entries with ` + "`$RUNWEAVER_BIN workflow update`" + ` using the exact CLI flags shown here. Do not invent plural changed-file flags.

- ` + "`--participants`" + ` and repeated ` + "`--participant-rationale`" + `
- repeated ` + "`--finding`" + ` and ` + "`--decision`" + `
- repeated ` + "`--file-read <path>`" + ` and ` + "`--file-changed <path>`" + `
- repeated ` + "`--artifact <path>`" + `
- ` + "`--last-result <text>`" + ` with the latest command, agent, or phase outcome
- repeated ` + "`--rejected-path <reason>`" + ` for rejected or paused approaches, paths, or commands, including why
- repeated ` + "`--verification <command>`" + `, ` + "`--verification-result <result>`" + `, and ` + "`--residual-risk <reason>`" + `
- ` + "`--next-verification <step>`" + ` with the next concrete verification step
- repeated ` + "`--blocker <text>`" + ` when work cannot safely continue
- ` + "`--next-action <text>`" + ` when the workflow is not complete

Prefer one domain owner plus up to two reviewers/skills. Do not exceed workflow ` + "`maxParticipants`" + ` unless the workflow explicitly requires it.

## Verification

Before a final response for an execution workflow, run:

` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" workflow verify --repo . --resume latest`" + `

The verifier must return ` + "`status: ok`" + ` and ` + "`ready: true`" + ` before a final response. Treat ` + "`warning`" + `, ` + "`error`" + `, and ` + "`ready: false`" + ` as hard blockers: fix the cause, complete/update phases, or record ` + "`--blocker`" + ` plus ` + "`--next-verification`" + `. Do not claim verification passed when this command returns warning/error.

For changed source, test, or config files, record a focused lint/static/type/build check separately from broad checks. For changed tests, record assertion-quality review evidence showing assertions prove required values/behavior and were not weakened to existence-only checks.

## Output Contract

Return ` + "`files_read`, `files_changed`, `checkpoint_fields_updated`, `artifacts`, `last_result`, `rejected_paths`, `verification_results`, `next_verification`, `blockers`, and `workflow_verify`." + `
`

const metadataRefreshSkill = `---
name: metadata-refresh
description: "Refresh repository runtime metadata after code changes"
compatibility: opencode
---

1. Use this skill only for explicit metadata regeneration tasks. Run ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" refresh --repo . --classification deterministic`" + `.
2. Inspect .runweaver/tmp/index/repo-context.md, .runweaver/tmp/index/repo-index.compact.json, .runweaver/tmp/index/manifest.json, .runweaver/tmp/surface-index.json, .runweaver/tmp/drift-report.json, and .runweaver/tmp/profile.generated.json.
3. Apply profile/agent/skill changes only after review. Do not run metadata refresh as a routine finalization step after ordinary code/test/build work.

## Output Contract

Return ` + "`surface_index_path`, `drift_report_path`, `profile_changes`, and `checkpoint_update`." + `
`

const repoOnboardingSkill = `---
name: repo-onboarding
description: "Onboard into a repository by reading configs, source layout, tests, and runtime metadata"
compatibility: opencode
---

1. Read ` + "`AGENTS.md`" + `.
2. Run ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" index --repo . --changed-only --prune`" + `.
3. Inspect detected package roles, tools, entrypoints, configs, tests, routes/controllers/pages, symbols, and verification commands.

## Output Contract

Return ` + "`task_map`, `important_files`, `risks`, `verification`, and `checkpoint_update`." + `
`

func workflowSystemMasterSkill(runtimeID string) string {
	return `---
name: workflow-system-master
description: "Operate RunWeaver workflow state, checkpoints, participant routing, and resume discipline"
compatibility: ` + runtimeID + `
---

Use this skill when a task touches RunWeaver workflows, workflow selection, participant selection, context retrieval, checkpoint state, or resume behavior.

## Procedure

1. Treat ` + "`.runweaver/workflows`" + ` as workflow templates and ` + "`.runweaver/tmp/swarm-runs`" + ` as mutable run state.
2. Start or resume with ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --task \"<full raw user task>\"`" + ` before lower-level workflow commands. Do not summarize or shorten the user task before this call.
3. Use ` + "`$RUNWEAVER_BIN workflow update`" + ` to record participants, rationale, files read, files changed, findings, decisions, last result, rejected paths, blockers, next action, and next verification.
4. Use ` + "`$RUNWEAVER_BIN workflow verify --repo . --resume latest`" + ` before final response when feasible.
5. Treat verifier ` + "`warning`" + `, ` + "`error`" + `, or ` + "`ready: false`" + ` as a hard blocker. Fix it, complete/update phases, or record ` + "`--blocker`" + ` and ` + "`--next-verification`" + ` before final response.
6. If a workflow, participant, or context selection looks wrong, record the rejected path and rerun the specific selector rather than guessing.

## Output Contract

Return ` + "`workflow`, `participants`, `checkpoint_fields_updated`, `rejected_paths`, `verification_results`, and `next_verification`." + `
`
}

func postImplementationReviewSkill(runtimeID string) string {
	return `---
name: post-implementation-review
description: "Review completed implementation for behavioral correctness before final response"
compatibility: ` + runtimeID + `
---

Use this skill after source changes and before final response for feature, bugfix, refactor, or test-hardening workflows.

## Review Checklist

1. If workflow state is missing, call ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --task \"<full raw user task>\"`" + ` before reviewing. Do not summarize or shorten the user task before this call.
2. Compare the original task, selected workflow, selected participants, changed files, and every acceptanceCriteria item from checkpoint/current.md.
3. Check for stale async/race outcomes and verify that late results cannot overwrite newer state.
4. Check allowed/forbidden state semantics, especially when an API says an action is allowed but with warnings or constraints.
5. Check loading, disabled, error, empty, and success states when UI behavior changed.
6. For UI additions, prefer a cohesive small component/helper for a new summary, panel, card, dialog section, or repeated block instead of inlining more structure into an already large component.
7. For tasks that exclude a surface, verify files from that excluded surface were not changed unless task evidence required it.
8. Prefer existing focused tests. Flag broad scaffolding, unrelated fixtures, or new harnesses that are not needed for the task.
9. If tests changed, review assertion quality against the task acceptance criteria. Flag tests that only prove existence or snapshot shape when the requirement needs values, state transitions, errors, or calls.
10. Require a focused lint/static/type/build check for changed source, test, or config files. Broad lint failures can be residual risk only after a focused changed-file check is recorded.
11. Record unmet acceptance criteria as blockers or residual risks; record findings, rejected paths, verification results, blockers, and the next verification step in the RunWeaver checkpoint.

## Output Contract

Return ` + "`changed_files_reviewed`, `findings`, `rejected_paths`, `verification_results`, `next_verification`, and `checkpoint_update`." + `
`
}
