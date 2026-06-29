package aitools

const opencodeRunWeaverStartCommand = `---
description: Start or resume RunWeaver orchestration for the current task
---

Run this command for non-trivial coding, bugfix, refactor, test, review, onboarding, or metadata work.

` + "`runweaver start --repo . --runtime opencode --task \"$ARGUMENTS\"`" + `

Follow the returned ` + "`executionContract`" + `, selected participants, task tier, task-scoped context, and next verification step. Do not stop after plan creation unless the user explicitly asked for planning-only work.
`

const codexRunWeaverStartSkill = `---
name: runweaver-start
description: "Start or resume RunWeaver orchestration before non-trivial repository work"
compatibility: codex
---

Use this skill when a user asks for coding, bugfix, refactor, test, review, onboarding, or metadata work.

First run:

` + "`runweaver start --repo . --runtime codex --task \"<user task>\"`" + `

Then follow the returned ` + "`executionContract`" + `, selected participants, task tier, task-scoped context, and next verification step. Keep ` + "`checkpoint.json`" + ` current with ` + "`runweaver workflow update`" + ` after each phase.
`

const claudeRunWeaverStartSkill = `---
name: runweaver-start
description: "Start or resume RunWeaver orchestration before non-trivial repository work"
compatibility: claude
---

Use this skill when a user asks for coding, bugfix, refactor, test, review, onboarding, or metadata work.

First run:

` + "`runweaver start --repo . --runtime claude --task \"<user task>\"`" + `

Then follow the returned ` + "`executionContract`" + `, selected participants, task tier, task-scoped context, and next verification step. Keep ` + "`checkpoint.json`" + ` current with ` + "`runweaver workflow update`" + ` after each phase.
`
