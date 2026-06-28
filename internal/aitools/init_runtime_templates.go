package aitools

const agentsMD = `# Repository AI Rules

RunWeaver metadata is generated and should be refreshed after code structure changes.

` + runWeaverStartupProtocol + `

Use ` + "`runweaver index --repo . --changed-only --prune`" + ` to refresh the local package/file/symbol index under .runweaver/tmp.

Use ` + "`runweaver refresh --repo .`" + ` after route, page, controller, service, test, or build config moves.

Runtime-specific metadata may be generated for OpenCode, Codex, Claude Code, or all selected providers. The durable workflow state currently lives under .runweaver/tmp/swarm-runs for compatibility.
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
