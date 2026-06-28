package aitools

const agentsMD = `# Repository AI Rules

RunWeaver metadata is generated and should be refreshed after code structure changes.

` + runWeaverStartupProtocol + `

Use ` + "`runweaver index --repo . --changed-only --prune`" + ` to refresh the local package/file/symbol index under .runweaver/tmp.

Use ` + "`runweaver refresh --repo .`" + ` after route, page, controller, service, test, or build config moves.

Runtime-specific metadata may be generated for OpenCode, Codex, Claude Code, or all selected providers. The durable workflow state currently lives under .runweaver/tmp/swarm-runs for compatibility.
`

const startHereMD = `# RunWeaver Start Here

This repository has RunWeaver metadata for coding-agent workflows.

## First Commands

1. Start or resume a task:

   ` + "`runweaver start --repo . --task \"<user task>\"`" + `

2. Inspect current state when debugging:

   ` + "`runweaver status --repo .`" + `

3. Refresh compact repository context manually when code moved:

   ` + "`runweaver index --repo . --changed-only --prune`" + `

` + "`runweaver start`" + ` creates or resumes workflow state under ` + "`.runweaver/tmp/swarm-runs`" + ` and returns the selected workflow, participants, next phase, next action, and next verification.

## Files To Read

- ` + "`AGENTS.md`" + ` - repo rules and RunWeaver startup protocol for Codex/OpenCode-compatible agents.
- ` + "`CLAUDE.md`" + ` - Claude Code startup protocol when generated.
- ` + "`.runweaver/tmp/current.md`" + ` - latest human-readable workflow state when a run exists.
- ` + "`.runweaver/tmp/index/repo-context.md`" + ` - compact repository context.
- ` + "`.runweaver/workflows`" + ` - portable workflow templates.

## Agent Rule

Agents should call ` + "`runweaver start`" + ` for non-trivial tasks, resume matching active workflows automatically, and should not ask the user to run start, status, resume, update, or verify commands manually unless RunWeaver is unavailable or blocked by permissions.
`

const opencodeJSON = `{
  "$schema": "https://opencode.ai/config.json",
  "default_agent": "swarm",
  "instructions": ["AGENTS.md"],
  "permission": {
    "edit": "allow",
    "bash": {
      "*": "allow",
      "runweaver start *": "allow",
      "runweaver *": "allow",
      "runweaver workflow run *": "allow",
      "runweaver workflow update *": "allow",
      "runweaver workflow verify *": "allow",
      "runweaver workflow run --resume *": "allow",
      "runweaver index *": "allow",
      "runweaver refresh *": "allow",
      "git status*": "allow",
      "git diff*": "allow",
      "rg *": "allow",
      "find *": "allow",
      "ls -la *": "allow",
      "ls -l *": "allow",
      "ls -a *": "allow",
      "ls *": "allow",
      "ls": "allow",
      "pwd": "allow"
    },
    "task": "allow",
    "todowrite": "allow",
    "skill": {
      "*": "allow"
    }
  },
  "watcher": {
    "ignore": [".git/**", ".runweaver/tmp/**", "node_modules/**", "build/**", "dist/**", "target/**"]
  }
}
`
