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

1. Inspect current state:

   ` + "`runweaver status --repo .`" + `

2. Refresh compact repository context when code moved or before non-trivial work:

   ` + "`runweaver index --repo . --changed-only --prune`" + `

3. Create or resume workflow state under ` + "`.runweaver/tmp/swarm-runs`" + `.

## Files To Read

- ` + "`AGENTS.md`" + ` - repo rules and RunWeaver startup protocol for Codex/OpenCode-compatible agents.
- ` + "`CLAUDE.md`" + ` - Claude Code startup protocol when generated.
- ` + "`.runweaver/tmp/current.md`" + ` - latest human-readable workflow state when a run exists.
- ` + "`.runweaver/tmp/index/repo-context.md`" + ` - compact repository context.
- ` + "`.runweaver/workflows`" + ` - portable workflow templates.

## Agent Rule

Agents should resume matching active workflows automatically and should not ask the user to run status, resume, update, or verify commands manually unless RunWeaver is unavailable or blocked by permissions.
`

const opencodeJSON = `{
  "$schema": "https://opencode.ai/config.json",
  "default_agent": "swarm",
  "instructions": ["AGENTS.md"],
  "permission": {
    "edit": "allow",
    "bash": {
      "*": "allow",
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
