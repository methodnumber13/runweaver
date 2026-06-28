package aitools

const runWeaverStartupProtocol = `## RunWeaver Startup Protocol

For any non-trivial coding, bugfix, refactor, test, review, onboarding, or metadata task:

1. Run ` + "`runweaver status --repo .`" + ` first and inspect the active workflow, ` + "`current.md`" + `, and recommendations.
2. If the latest workflow matches the user request and is not complete, resume automatically from ` + "`.runweaver/tmp/swarm-runs/latest.json`" + ` and the run-local ` + "`current.md`" + `.
3. If there is no matching active workflow, refresh context with ` + "`runweaver index --repo . --changed-only --prune`" + ` and create the closest workflow from ` + "`.runweaver/workflows`" + `.
4. Keep ` + "`checkpoint.json`" + `, ` + "`todo.md`" + `, and ` + "`current.md`" + ` current with ` + "`runweaver workflow update`" + ` after each phase.
5. Use repo-specific agents and skills as participants. Spawn/delegate to named agents when the runtime supports it; otherwise emulate the selected participant roles explicitly and record them in the checkpoint.
6. Continue until implementation and verification are complete, unless the user explicitly asks for planning-only work.
7. Before final response, run ` + "`runweaver workflow verify --repo . --resume latest`" + ` when feasible and record blockers when it is not.

Do not ask the user to run resume, status, update, or verify commands manually unless RunWeaver is unavailable or blocked by permissions.
`
