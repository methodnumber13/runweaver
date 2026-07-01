package aitools

const opencodeRunWeaverStartCommand = `---
description: Start or resume RunWeaver orchestration for the current task
---

Run this command for non-trivial coding, bugfix, refactor, test, review, onboarding, or metadata work.

` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --runtime opencode --task \"$ARGUMENTS\"`" + `

Pass the full raw user task as ` + "`$ARGUMENTS`" + `, including constraints and verification requirements. Do not summarize or shorten it before invoking RunWeaver.

Follow the returned ` + "`executionContract`" + `, selected participants, task tier, task-scoped context, and next verification step. Do not stop after plan creation unless the user explicitly asked for planning-only work.
`

const codexRunWeaverStartSkill = `---
name: runweaver-start
description: "Start or resume RunWeaver orchestration before non-trivial repository work"
compatibility: codex
---

Use this skill when a user asks for coding, bugfix, refactor, test, review, onboarding, or metadata work.

First-action contract: after automatic runtime instruction loading, this skill should cause the first active tool or shell action for a non-trivial task to be ` + "`runweaver start`" + `. Do not inspect other generated repo skills or source files first unless the start command is unavailable.

First run:

` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --runtime codex --task \"<full raw user task>\"`" + `

Use the full raw user task exactly as received, including constraints, file/path restrictions, verification requirements, and examples. Do not summarize or shorten it before invoking RunWeaver.

Use the repo-local ` + "`RUNWEAVER_BIN`" + ` shim instead of a bare global ` + "`runweaver`" + `; the global binary can be stale and may not contain ` + "`runweaver start`" + `.

Then follow the returned ` + "`executionContract`" + `, especially acceptanceCriteria, selected participants, task tier, task-scoped context, and next verification step. Keep ` + "`checkpoint.json`" + ` current with ` + "`$RUNWEAVER_BIN workflow update`" + ` after each phase using exact flags such as ` + "`--file-read`" + `, ` + "`--file-changed`" + `, ` + "`--last-result`" + `, ` + "`--rejected-path`" + `, ` + "`--verification-result`" + `, and ` + "`--blocker`" + `.
`

const claudeRunWeaverStartSkill = `---
name: runweaver-start
description: "Start or resume RunWeaver orchestration before non-trivial repository work"
compatibility: claude
---

Use this skill when a user asks for coding, bugfix, refactor, test, review, onboarding, or metadata work.

First-action contract: after automatic runtime instruction loading, this skill should cause the first active tool or shell action for a non-trivial task to be ` + "`runweaver start`" + `. Do not inspect other generated repo skills or source files first unless the start command is unavailable.

First run:

` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --runtime claude --task \"<full raw user task>\"`" + `

Use the full raw user task exactly as received, including constraints, file/path restrictions, verification requirements, and examples. Do not summarize or shorten it before invoking RunWeaver.

Use the repo-local ` + "`RUNWEAVER_BIN`" + ` shim instead of a bare global ` + "`runweaver`" + `; the global binary can be stale and may not contain ` + "`runweaver start`" + `.

Then follow the returned ` + "`executionContract`" + `, especially acceptanceCriteria, selected participants, task tier, task-scoped context, and next verification step. Keep ` + "`checkpoint.json`" + ` current with ` + "`$RUNWEAVER_BIN workflow update`" + ` after each phase using exact flags such as ` + "`--file-read`" + `, ` + "`--file-changed`" + `, ` + "`--last-result`" + `, ` + "`--rejected-path`" + `, ` + "`--verification-result`" + `, and ` + "`--blocker`" + `.
`
