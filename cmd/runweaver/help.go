package main

import (
	"fmt"
)

func (c cli) usage() {
	fmt.Fprint(c.stdout, c.paint(ansiCyan, "runweaver commands")+"\n")
	fmt.Fprint(c.stdout, `  scan --repo <path> [--out file]       scan repository surfaces
  index --repo <path> [--changed-only] [--prune]
                                        build repo-local package/file/symbol index
  index clean --repo <path>             remove .runweaver/tmp index/cache
  classify --repo <path> [--classification auto]
                                        classify repo into swarm agents/skills
  refresh --repo <path> [--apply]       write surface/drift/profile artifacts
  status --repo <path>                  show active workflow and current RunWeaver state
  start --repo <path> --task <text>     create or resume the right workflow for a user task
  context query --task <text>           return task-scoped files, symbols, routes, and tests
  doctor --repo <path>                  refresh and summarize drift status
  doctor model --repo <path>            check OpenCode provider/model/key setup
  doctor opencode --repo <path> [--timeout 60s]
                                        check Desktop/CLI swarm readiness
  doctor runtime --repo <path> [--runtime all]
                                        discover OpenCode/Codex/Claude configs and metadata
  doctor adoption --repo <path> [--runtime all]
                                        verify RunWeaver auto-start contract for runtimes
  doctor processes [--summary]          show Codex/OpenCode runtime, VS Code debugger, and duplicate MCP processes
  eval adoption --repo <path>           run local adoption/start smoke checks
  smoke codex [--live]                  create a disposable repo and verify Codex adoption
  init --repo <path> [--runtime opencode] [--force]
                                        smart index, plan intelligence workflow, bootstrap metadata
  bootstrap --repo <path> [--runtime opencode]
                                        alias for init with friendlier naming
  mcp serve --repo <path>               expose read-only RunWeaver status/tools over MCP stdio
  participants select --task <text>     choose workflow participants from generated profile
  workflow run --workflow <file> --task <text> [--runtime opencode] [--execute]
                                        create or execute workflow plan/checkpoint under .runweaver/tmp
  workflow select --task <text>         select the closest workflow for a task
  workflow update --resume latest --phase <id>
                                        persist workflow checkpoint participants/findings
  workflow verify --resume latest       verify run artifacts, checkpoint, todo, events, and terminal evidence
  version [--json]                      print build version metadata
`)
}

func (c cli) commandUsage(command string) {
	if command == "" || command == "help" {
		c.usage()
		return
	}
	if text := commandUsage(command); text != "" {
		fmt.Fprint(c.stdout, c.paint(ansiCyan, "runweaver "+command)+"\n")
		fmt.Fprint(c.stdout, text)
		return
	}
	c.usage()
}
