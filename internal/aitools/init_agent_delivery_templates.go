package aitools

const repoSurfaceEngineerAgent = `---
description: "Generic implementation agent for repository source surfaces"
mode: subagent
permission:
  edit: allow
  bash:
    "runweaver scan *": allow
    "runweaver index *": allow
    "runweaver refresh *": allow
    "rg *": allow
    "find *": allow
    "ls *": allow
    "ls": allow
    "pwd": allow
    "git diff*": allow
  skill:
    "*": allow
---

Use runtime profile, .runweaver/tmp/index/repo-context.md, and .runweaver/tmp/index/repo-index.compact.json to select the most specific generated domain agent or skill for the task. Prefer a domain agent over a layer reviewer when both match. Open the full repo-index.json only when compact context is insufficient. If no stack-specific role exists, implement scoped source changes yourself.

## Output Contract

Return ` + "`files_read`, `files_changed`, `contracts_checked`, `verification`, `risks`, and `checkpoint_update`." + `
`

const repoContractReviewerAgent = `---
description: "Generic reviewer for public contracts, configuration, boundaries, and metadata drift"
mode: subagent
permission:
  edit: allow
  bash:
    "runweaver scan *": allow
    "runweaver index *": allow
    "runweaver refresh *": allow
    "rg *": allow
    "find *": allow
    "ls *": allow
    "ls": allow
    "pwd": allow
    "git diff*": allow
  skill:
    "*": allow
---

Review request/response contracts, imports, public APIs, configuration, generated metadata anchors, and compatibility risks for the current task.

## Output Contract

Return ` + "`contracts_checked`, `findings`, `metadata_drift`, `verification`, `risks`, and `checkpoint_update`." + `
`

const repoTestQualityReviewerAgent = `---
description: "Generic reviewer for tests, fixtures, mocks, and verification commands"
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

Find the smallest reliable verification set for the task, update tests when explicitly assigned, and record exact blockers for commands that cannot run.

## Output Contract

Return ` + "`tests_found`, `tests_changed`, `commands_run`, `coverage_gaps`, `risks`, and `checkpoint_update`." + `
`
