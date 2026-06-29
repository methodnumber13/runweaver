package aitools

const runWeaverStartupProtocol = `## RunWeaver Startup Protocol

For any non-trivial coding, bugfix, refactor, test, review, onboarding, or metadata task:

1. Call ` + "`runweaver start --repo . --task \"<user task>\"`" + ` first. Treat its JSON ` + "`executionContract`" + ` as the source of truth for workflow, participants, next phase, next action, and next verification.
2. If ` + "`runweaver start`" + ` returns ` + "`action: resumed`" + `, resume automatically from ` + "`.runweaver/tmp/swarm-runs/latest.json`" + ` and the run-local ` + "`current.md`" + `. Do not create a competing run.
3. If ` + "`runweaver start`" + ` returns ` + "`action: created`" + `, continue from the selected workflow and participants it returned. The plan is only the durable checkpoint; do not stop after plan creation unless the user explicitly asked for planning-only work.
4. Keep ` + "`checkpoint.json`" + `, ` + "`todo.md`" + `, and ` + "`current.md`" + ` current with ` + "`runweaver workflow update`" + ` after each phase. Include ` + "`lastResult`" + `, ` + "`filesChanged`" + `, ` + "`rejectedPaths`" + `, ` + "`nextAction`" + `, and ` + "`nextVerification`" + ` whenever they explain why the next move is safe.
5. Use the participants returned by ` + "`runweaver start`" + `. Spawn/delegate to named agents when the runtime supports it; otherwise emulate the selected participant roles explicitly and record them in the checkpoint.
6. Continue until implementation and verification are complete, unless the user explicitly asks for planning-only work.
7. Before final response, run ` + "`runweaver workflow verify --repo . --resume latest`" + ` when feasible and record blockers plus the next verification step when it is not.

Use ` + "`runweaver status --repo .`" + ` and ` + "`runweaver workflow run --resume latest --status`" + ` only as diagnostics when ` + "`runweaver start`" + ` is unavailable or blocked by permissions. Do not ask the user to run start, resume, status, update, or verify commands manually unless RunWeaver itself is unavailable.
`
