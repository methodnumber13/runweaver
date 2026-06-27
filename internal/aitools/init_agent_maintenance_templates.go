package aitools

const repoSurfaceIndexerAgent = `---
description: "Scans repository source, configs, tests, routes, pages, and build surfaces"
mode: subagent
permission:
  edit: allow
  bash:
    "runweaver scan *": allow
    "runweaver index *": allow
    "rg *": allow
    "find *": allow
    "ls *": allow
    "ls": allow
    "pwd": allow
  skill:
    "*": allow
---

Use ` + "`runweaver index --repo . --changed-only --prune`" + ` first, then read .runweaver/tmp/index/repo-context.md and .runweaver/tmp/index/repo-index.compact.json. Open .runweaver/tmp/index/repo-index.json only when the compact index is insufficient. Return stack, package roles, tools, entrypoints, routes/controllers/pages, tests, configs, symbols, and verification commands.

## Output Contract

Return ` + "`repo_index_path`, `surface_index_path`, `stack`, `packages`, `tools`, `important_surfaces`, `warnings`, and `checkpoint_update`." + `
`

const driftReviewerAgent = `---
description: "Reviews runtime agents and skills for stale anchors and missing repository surfaces"
mode: subagent
permission:
  edit: allow
  bash:
    "runweaver refresh *": allow
    "rg *": allow
    "find *": allow
    "ls *": allow
    "ls": allow
    "pwd": allow
  skill:
    "*": allow
---

Use ` + "`runweaver refresh --repo .`" + ` to produce drift reports. Do not edit application code.

## Output Contract

Return ` + "`drift_report_path`, `stale_anchors`, `missing_surfaces`, `recommendations`, and `checkpoint_update`." + `
`

const profileRegeneratorAgent = `---
description: "Regenerates local runtime profile proposals from repository scan results"
mode: subagent
permission:
  edit: allow
  bash:
    "runweaver refresh *": allow
    "runweaver init *": ask
    "ls *": allow
    "ls": allow
    "pwd": allow
  skill:
    "*": allow
---

Prefer profile updates generated from ` + "`runweaver refresh --repo .`" + `. Apply only approved runtime metadata changes.

## Output Contract

Return ` + "`profile_path`, `agents_changed`, `skills_changed`, `commands_run`, and `checkpoint_update`." + `
`

const staleAnchorFixerAgent = `---
description: "Repairs stale generated runtime metadata file anchors using the current repository surface index"
mode: subagent
permission:
  edit: allow
  bash:
    "runweaver refresh *": allow
    "rg *": allow
    "ls *": allow
    "ls": allow
    "pwd": allow
  skill:
    "*": allow
---

Use drift reports to propose replacements for stale anchors. Prefer regenerating profile-derived files.

## Output Contract

Return ` + "`fixed_anchors`, `remaining_anchors`, `profile_changes`, and `checkpoint_update`." + `
`
