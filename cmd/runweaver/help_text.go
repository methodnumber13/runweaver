package main

func commandUsage(command string) string {
	switch command {
	case "scan":
		return `Usage:
  runweaver scan --repo <path> [--out file]

Flags:
  --repo path    repository path (default ".")
  --out file     write JSON scan result to file instead of stdout
`
	case "index":
		return `Usage:
  runweaver index --repo <path> [--changed-only] [--prune] [--classification auto|ai|deterministic]
  runweaver index clean --repo <path>

Flags:
  --repo path                   repository path (default ".")
  --changed-only                reuse content-hash cache for unchanged files (default true)
  --prune                       remove stale cache entries after indexing
  --max-cache-mb n              max cache size when pruning; 0 disables size pruning (default 256)
  --classification mode         auto, ai, or deterministic
  --classifier-runtime id       AI classifier runtime: opencode, codex, or claude
  --classifier-model model      optional runtime model override
  --classifier-timeout duration AI classifier timeout (default 180s)
`
	case "index clean":
		return `Usage:
  runweaver index clean --repo <path>

Flags:
  --repo path    repository path (default ".")
`
	case "classify":
		return `Usage:
  runweaver classify --repo <path> [--classification auto|ai|deterministic] [--apply]

Flags:
  --repo path                         repository path (default ".")
  --apply                             materialize generated profile, agents, and skills into selected runtime metadata
  --runtime ids                       runtime provider to materialize: opencode, codex, claude, all, or comma-separated list
  --classification mode               auto, ai, or deterministic (default auto)
  --classifier mode                   alias for --classification
  --classifier-provider provider      OpenCode provider id for AI classification; inferred from configured model when omitted
  --classifier-runtime id             AI classifier runtime: opencode, codex, or claude
  --classifier-model model            optional runtime model override
  --classifier-opencode-bin path      OpenCode executable path (default opencode)
  --classifier-codex-bin path         Codex executable path (default codex)
  --classifier-claude-bin path        Claude Code executable path (default claude)
  --classifier-agent agent            OpenCode classifier agent (default repo-classifier)
  --classifier-sandbox mode           Codex classifier sandbox mode (default read-only)
  --classifier-approval-policy mode   Codex classifier approval policy (default never)
  --classifier-permission-mode mode   Claude classifier permission mode (default dontAsk)
  --classifier-claude-tools list      optional Claude classifier tools passed to --tools
  --classifier-skip-git-repo-check    allow Codex classifier outside a Git repository
  --classifier-timeout duration       AI classifier timeout (default 180s)
  --classifier-skip-model-check       skip model preflight before AI classification
  --classifier-skip-runtime-check     skip runtime binary/config discovery before AI classification
`
	case "refresh":
		return `Usage:
  runweaver refresh --repo <path> [--apply] [--classification auto|ai|deterministic]

Flags:
  --repo path                   repository path (default ".")
  --apply                       update selected runtime profile and generated metadata
  --runtime ids                 runtime provider to materialize: opencode, codex, claude, all, or comma-separated list
  --classification mode         auto, ai, or deterministic (default auto)
  --classifier-runtime id       AI classifier runtime: opencode, codex, or claude
  --classifier-model model      optional runtime model override
  --classifier-timeout duration AI classifier timeout (default 180s)
`
	case "status":
		return `Usage:
  runweaver status --repo <path>

Flags:
  --repo path    repository path (default ".")
`
	case "start":
		return `Usage:
  runweaver start --repo <path> --task <text> [--runtime opencode|codex|claude]

Flags:
  --repo path       repository path (default ".")
  --task text       user task to start or resume
  --runtime id      runtime profile to inspect: opencode, codex, or claude
  --workflow file   explicit workflow JSON path
  --profile file    explicit RunWeaver profile JSON path
  --skip-index      skip automatic index refresh
  --force-new       create a new workflow even when latest matches
`
	case "doctor":
		return `Usage:
  runweaver doctor --repo <path>
  runweaver doctor model --repo <path>
  runweaver doctor opencode --repo <path> [--skip-model-check]
  runweaver doctor runtime --repo <path> [--runtime all]
  runweaver doctor adoption --repo <path> [--runtime all]
  runweaver doctor processes [--summary]
`
	case "doctor model":
		return `Usage:
  runweaver doctor model --repo <path> [--provider id] [--model model] [--base-url url]

Flags:
  --repo path       repository path (default ".")
  --provider id     OpenCode provider id; inferred from configured model when omitted
  --model id        expected model id without provider prefix
  --base-url url    expected OpenAI-compatible base URL
`
	case "doctor opencode":
		return `Usage:
  runweaver doctor opencode --repo <path> [--skip-model-check] [--timeout 60s]

Flags:
  --repo path            repository path (default ".")
  --opencode-bin path    OpenCode executable path (default opencode)
  --agent name           primary agent name (default swarm)
  --provider id          OpenCode provider id for model preflight; inferred from configured model when omitted
  --skip-model-check     skip OpenCode model preflight
  --timeout duration     per OpenCode debug command timeout (default 45s)
`
	case "doctor runtime":
		return `Usage:
  runweaver doctor runtime --repo <path> [--runtime opencode|codex|claude|all]

Flags:
  --repo path       repository path (default ".")
  --runtime ids     runtime provider: opencode, codex, claude, all, or comma-separated list (default all)
`
	case "doctor adoption":
		return `Usage:
  runweaver doctor adoption --repo <path> [--runtime opencode|codex|claude|all]

Flags:
  --repo path       repository path (default ".")
  --runtime ids     runtime provider: opencode, codex, claude, all, or comma-separated list (default all)
`
	case "doctor processes":
		return `Usage:
  runweaver doctor processes [--summary]

Flags:
  --summary    print counts and duplicate process groups without full process details
`
	case "init":
		return `Usage:
  runweaver init --repo <path> [--runtime opencode|codex|claude|all] [--force] [--require-model] [--classification auto|ai|deterministic]

Flags:
  --repo path                   repository path (default ".")
  --runtime ids                 runtime provider: opencode, codex, claude, all, or comma-separated list (default opencode)
  --force                       refresh RunWeaver-generated files; merge project configs with backups
  --require-model               fail if provider/model/key preflight is not ready
  --provider id                 OpenCode provider id for model preflight; inferred from configured model when omitted
  --model id                    expected model id without provider prefix
  --base-url url                expected OpenAI-compatible base URL
  --classification mode         auto, ai, or deterministic (default auto)
  --classifier-runtime id       AI classifier runtime: opencode, codex, or claude
  --classifier-model model      optional runtime model override
  --classifier-timeout duration AI classifier timeout (default 180s)
`
	case "bootstrap":
		return `Usage:
  runweaver bootstrap --repo <path> [--runtime opencode|codex|claude|all] [--force] [--require-model] [--classification auto|ai|deterministic]

Alias for runweaver init with friendlier onboarding naming. Flags are identical to init.
`
	case "mcp serve":
		return `Usage:
  runweaver mcp serve --repo <path> [--allow-workflow-writes]

Flags:
  --repo path                 repository path exposed through MCP tools (default ".")
  --allow-workflow-writes     expose tools that create/update .runweaver workflow state
`
	case "participants select":
		return `Usage:
  runweaver participants select --repo <path> --task <text> [--workflow file]

Flags:
  --repo path       repository path (default ".")
  --task text       task description to route
  --workflow file   workflow JSON path; selected automatically when omitted
  --runtime id      runtime profile to inspect: opencode, codex, or claude
  --profile file    explicit RunWeaver profile JSON path
`
	case "workflow run":
		return `Usage:
  runweaver workflow run --workflow <file> --task <text> [--repo <path>] [--execute]
  runweaver workflow run --resume latest --status
  runweaver workflow run --resume latest --execute

Flags:
  --repo path              repository path (default ".")
  --workflow file          workflow JSON path
  --task text              task description
  --resume latest|path     run directory or latest
  --status                 print workflow status
  --execute                execute through selected runtime after creating/loading the plan
  --dry-run                prepare command without running the selected runtime
  --runtime id             runtime provider: opencode, codex, or claude (default opencode)
  --skip-model-check       skip OpenCode model preflight before execution
  --opencode-bin path      OpenCode executable path (default opencode)
  --codex-bin path         Codex executable path (default codex)
  --claude-bin path        Claude Code executable path (default claude)
  --agent name             OpenCode primary agent for execution (default swarm)
  --provider id            OpenCode provider id for model preflight; inferred from configured model when omitted
  --model model            optional runtime model override
  --attach url             optional opencode serve URL to attach to
  --sandbox mode           Codex sandbox mode (default workspace-write)
  --approval-policy mode   Codex approval policy (default never)
  --permission-mode mode   Claude permission mode (default dontAsk)
  --claude-tools list      Claude tools passed to --tools
  --skip-git-repo-check    allow Codex outside a Git repository
  --skip-runtime-check     skip runtime binary/config discovery before execution
`
	case "workflow select":
		return `Usage:
  runweaver workflow select --repo <path> --task <text> [--workflow file]

Flags:
  --repo path        repository path (default ".")
  --task text        task description to route
  --workflow file    explicit workflow JSON path; bypasses scoring when provided
`
	case "workflow update":
		return `Usage:
  runweaver workflow update --repo <path> --resume latest --phase <id> [--status in_progress]

Flags:
  --repo path              repository path (default ".")
  --resume latest|path     run directory or latest
  --phase id               current workflow phase
  --status status          checkpoint status
  --participants list      comma-separated participant names
  --participant-rationale  participant selection rationale to append; may be repeated
  --finding text           finding to append; may be repeated
  --decision text          decision to append; may be repeated
  --file-read path         file read during the phase; may be repeated
  --file-changed path      file changed during the phase; may be repeated
  --artifact path          artifact path to append; may be repeated
  --last-result text       last result explaining why the workflow is moving or pausing
  --rejected-path text     path, command, or approach rejected/paused with reason; may be repeated
  --next-action text       next action to persist
  --next-verification text next verification step before continuing or finishing
  --verification command   verification command to append; may be repeated
  --verification-result    verification result to append; may be repeated
  --blocker text           blocker to append; may be repeated
  --complete-phase         mark the current phase complete and advance nextPhase
`
	case "workflow verify":
		return `Usage:
  runweaver workflow verify --repo <path> --resume latest

Flags:
  --repo path              repository path (default ".")
  --resume latest|path     run directory or latest
`
	case "version":
		return `Usage:
  runweaver version [--json]

Flags:
  --json    print version metadata as JSON
`
	default:
		return ""
	}
}
