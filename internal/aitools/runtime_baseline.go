package aitools

import "strings"

func runtimeProviderIDs(providers []RuntimeProvider) []string {
	ids := make([]string, 0, len(providers))
	for _, provider := range providers {
		ids = append(ids, provider.ID)
	}
	return ids
}

func runtimeSelectionString(ids []string) string {
	if len(ids) == 0 {
		return RuntimeOpenCode
	}
	return strings.Join(ids, ",")
}

func hasRuntime(ids []string, id string) bool {
	for _, current := range ids {
		if current == id {
			return true
		}
	}
	return false
}

func shouldRunOpenCodeModelPreflight(opts InitOptions, runtimeIDs []string) bool {
	if opts.RequireModel {
		return true
	}
	classification, err := normalizeClassifyOptions(opts.Classification, ClassificationDeterministic)
	if err != nil {
		return true
	}
	return classification.Mode != ClassificationDeterministic && classification.Runtime == RuntimeOpenCode
}

func skippedModelPreflight(providerID string) ModelConfigCheck {
	if providerID == "" {
		providerID = "openai-compatible"
	}
	return ModelConfigCheck{
		Status:          "skipped",
		Ready:           false,
		ProviderID:      providerID,
		Env:             map[string]bool{},
		Recommendations: []string{"OpenCode model preflight was skipped because the selected runtime/classification path does not require OpenCode."},
	}
}

func runtimeBaselineDirs(runtimeIDs []string) []string {
	dirs := []string{
		".runweaver/workflows",
		".runweaver/tmp",
		".runweaver/tmp/bin",
	}
	if hasRuntime(runtimeIDs, RuntimeOpenCode) {
		dirs = append(dirs,
			".opencode/agents",
			".opencode/commands",
			".opencode/skills/context-discipline",
			".opencode/skills/metadata-refresh",
			".opencode/skills/post-implementation-review",
			".opencode/skills/repo-onboarding",
			".opencode/skills/workflow-system-master",
			".opencode/swarm",
		)
	}
	if hasRuntime(runtimeIDs, RuntimeCodex) {
		dirs = append(dirs,
			".agents/skills/context-discipline",
			".agents/skills/metadata-refresh",
			".agents/skills/post-implementation-review",
			".agents/skills/repo-onboarding",
			".agents/skills/runweaver-start",
			".agents/skills/workflow-system-master",
			".codex/agents",
			".codex/runweaver",
		)
	}
	if hasRuntime(runtimeIDs, RuntimeClaude) {
		dirs = append(dirs,
			".claude/agents",
			".claude/skills/context-discipline",
			".claude/skills/metadata-refresh",
			".claude/skills/post-implementation-review",
			".claude/skills/repo-onboarding",
			".claude/skills/runweaver-start",
			".claude/skills/workflow-system-master",
			".claude/workflows",
			".claude/runweaver",
		)
	}
	return dirs
}

func runtimeBaselineFiles(runtimeIDs []string) map[string]string {
	files := map[string]string{
		".gitignore":               ".omo/\n",
		".runweaver/.gitignore":    "tmp/\n",
		".runweaver/START_HERE.md": startHereMD,
		".runweaver/workflows/repo-intelligence-swarm.json": repoIntelligenceWorkflow,
		".runweaver/workflows/metadata-refresh-swarm.json":  metadataRefreshWorkflow,
		".runweaver/workflows/repo-onboarding-swarm.json":   repoOnboardingWorkflow,
		".runweaver/workflows/feature-delivery-swarm.json":  featureDeliveryWorkflow,
		".runweaver/workflows/bugfix-swarm.json":            bugfixWorkflow,
		".runweaver/workflows/refactor-swarm.json":          refactorWorkflow,
		".runweaver/workflows/test-hardening-swarm.json":    testHardeningWorkflow,
	}
	if hasRuntime(runtimeIDs, RuntimeOpenCode) {
		files["AGENTS.md"] = agentsMD
		files["opencode.json"] = opencodeJSON
		files[".opencode/.gitignore"] = "tmp/\nswarm/profile.generated.json\nnode_modules/\npackage.json\npackage-lock.json\n"
		files[".opencode/agents/"+OpenCodePrimaryAgentName+".md"] = swarmAgent
		files[".opencode/agents/repo-classifier.md"] = repoClassifierAgent
		files[".opencode/agents/repo-surface-indexer.md"] = repoSurfaceIndexerAgent
		files[".opencode/agents/agent-skill-drift-reviewer.md"] = driftReviewerAgent
		files[".opencode/agents/profile-regenerator.md"] = profileRegeneratorAgent
		files[".opencode/agents/stale-anchor-fixer.md"] = staleAnchorFixerAgent
		files[".opencode/agents/repo-surface-engineer.md"] = repoSurfaceEngineerAgent
		files[".opencode/agents/repo-contract-reviewer.md"] = repoContractReviewerAgent
		files[".opencode/agents/repo-test-quality-reviewer.md"] = repoTestQualityReviewerAgent
		files[".opencode/agents/post-implementation-reviewer.md"] = postImplementationReviewerAgent
		files[".opencode/commands/runweaver-start.md"] = opencodeRunWeaverStartCommand
		files[".opencode/skills/context-discipline/SKILL.md"] = contextDisciplineSkill
		files[".opencode/skills/metadata-refresh/SKILL.md"] = metadataRefreshSkill
		files[".opencode/skills/post-implementation-review/SKILL.md"] = postImplementationReviewSkill(RuntimeOpenCode)
		files[".opencode/skills/repo-onboarding/SKILL.md"] = repoOnboardingSkill
		files[".opencode/skills/workflow-system-master/SKILL.md"] = workflowSystemMasterSkill(RuntimeOpenCode)
	}
	if hasRuntime(runtimeIDs, RuntimeCodex) {
		files["AGENTS.md"] = agentsMD
		files[".codex/agents/"+RunWeaverPrimaryAgentName+".toml"] = codexSwarmAgentTOML
		files[".codex/agents/post-implementation-reviewer.toml"] = codexPostImplementationReviewerAgentTOML
		files[".agents/skills/context-discipline/SKILL.md"] = codexContextDisciplineSkill
		files[".agents/skills/metadata-refresh/SKILL.md"] = codexMetadataRefreshSkill
		files[".agents/skills/post-implementation-review/SKILL.md"] = postImplementationReviewSkill(RuntimeCodex)
		files[".agents/skills/repo-onboarding/SKILL.md"] = codexRepoOnboardingSkill
		files[".agents/skills/runweaver-start/SKILL.md"] = codexRunWeaverStartSkill
		files[".agents/skills/workflow-system-master/SKILL.md"] = workflowSystemMasterSkill(RuntimeCodex)
	}
	if hasRuntime(runtimeIDs, RuntimeClaude) {
		files["CLAUDE.md"] = claudeMD
		files[".claude/agents/"+RunWeaverPrimaryAgentName+".md"] = claudeSwarmAgent
		files[".claude/agents/post-implementation-reviewer.md"] = claudePostImplementationReviewerAgent
		files[".claude/skills/context-discipline/SKILL.md"] = claudeContextDisciplineSkill
		files[".claude/skills/metadata-refresh/SKILL.md"] = claudeMetadataRefreshSkill
		files[".claude/skills/post-implementation-review/SKILL.md"] = postImplementationReviewSkill(RuntimeClaude)
		files[".claude/skills/repo-onboarding/SKILL.md"] = claudeRepoOnboardingSkill
		files[".claude/skills/runweaver-start/SKILL.md"] = claudeRunWeaverStartSkill
		files[".claude/skills/workflow-system-master/SKILL.md"] = workflowSystemMasterSkill(RuntimeClaude)
	}
	return files
}

func runtimeProfilePaths(runtimeIDs []string) []string {
	paths := []string{}
	for _, runtimeID := range runtimeIDs {
		adapter, ok := RuntimeAdapterByID(runtimeID)
		if !ok {
			continue
		}
		paths = append(paths, adapter.ProfilePath())
	}
	if len(paths) == 0 {
		paths = append(paths, openCodeRuntimeAdapter{}.ProfilePath())
	}
	return Unique(paths)
}

const codexSwarmAgentTOML = `# generated by runweaver; safe to regenerate
name = "` + RunWeaverPrimaryAgentName + `"
description = "Primary RunWeaver workflow coordinator for Codex"
developer_instructions = """
You are the ` + RunWeaverPrimaryAgentName + ` coordinator for this repository.

Read AGENTS.md, .codex/runweaver/profile.json, .runweaver/tmp/index/repo-context.md, and the latest workflow checkpoint when present. For non-trivial user tasks, call ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --runtime codex --task \"<full raw user task>\"`" + ` before coding and follow its executionContract. Do not summarize or shorten the user task before this call.

` + runWeaverStartupProtocol + `

For non-trivial work, use the repo-local shim for runweaver start as the single intake/resume entrypoint. Keep checkpoint.json and todo.md current through $RUNWEAVER_BIN workflow update. Continue through implementation and verification unless the user explicitly asks for planning only.

Use repo-specific agents from .codex/agents and repo skills from .agents/skills as local role instructions. In unattended codex exec runs, emulate selected participants in the primary agent by default; do not block on child-agent wait unless the user explicitly asked for live Codex subagents. Record participants, files read, files changed, findings, decisions, lastResult, rejectedPaths, verification results, residual risks, blockers, nextAction, and nextVerification in the checkpoint with exact workflow update flags such as --participants, --participant-rationale, --file-read, --file-changed, --last-result, --rejected-path, --verification-result, --residual-risk, --blocker, --next-action, and --next-verification.
"""
`

const codexContextDisciplineSkill = `---
name: context-discipline
description: "Filesystem-backed context discipline for durable RunWeaver workflows in Codex"
compatibility: codex
---

Use this skill for every non-trivial coding, bugfix, refactor, test, onboarding, or metadata workflow.

Maintain .runweaver/tmp/swarm-runs/<run-id>/checkpoint.json, todo.md, events.ndjson, phase notes, and verification logs through ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" workflow update`" + `. Use exact flags such as ` + "`--participants`" + `, ` + "`--participant-rationale`" + `, ` + "`--acceptance-criterion`" + `, ` + "`--file-read`" + `, ` + "`--file-changed`" + `, ` + "`--last-result`" + `, ` + "`--rejected-path`" + `, ` + "`--verification-result`" + `, ` + "`--residual-risk`" + `, ` + "`--blocker`" + `, ` + "`--next-action`" + `, and ` + "`--next-verification`" + ` whenever they explain the next move.

Before final response, compare every acceptance criterion against code, tests, and verification evidence, then run ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" workflow verify --repo . --resume latest`" + ` when feasible and record exact blockers plus the next verification step otherwise.
`

const codexMetadataRefreshSkill = `---
name: metadata-refresh
description: "Refresh RunWeaver/Codex metadata after code structure changes"
compatibility: codex
---

Use this skill only for explicit metadata regeneration tasks. Run ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" refresh --repo . --classification deterministic`" + `, inspect .runweaver/tmp/index/repo-context.md, .runweaver/tmp/index/repo-index.compact.json, .runweaver/tmp/drift-report.json, and apply generated metadata changes only after review. Do not run metadata refresh as a routine finalization step after ordinary code/test/build work.
`

const codexRepoOnboardingSkill = `---
name: repo-onboarding
description: "Onboard into a repository by reading configs, source layout, tests, and RunWeaver metadata"
compatibility: codex
---

Read AGENTS.md, run ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" index --repo . --changed-only --prune`" + `, and inspect detected packages, tools, entrypoints, configs, tests, routes/pages, symbols, and verification commands.
`

const claudeMD = `# RunWeaver Repository AI Rules

RunWeaver metadata is generated. Ordinary coding tasks should refresh only the deterministic local index; full metadata refresh is explicit metadata-maintenance work.

` + runWeaverStartupProtocol + `

Use ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" index --repo . --changed-only --prune`" + ` to refresh the local package/file/symbol index under .runweaver/tmp.

Use ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" refresh --repo . --classification deterministic`" + ` only when the user explicitly asks to regenerate RunWeaver runtime metadata or when running a metadata-refresh workflow.

For non-trivial work, call ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --runtime claude --task \"<full raw user task>\"`" + ` and keep checkpoint.json current with $RUNWEAVER_BIN workflow update. Do not summarize or shorten the user task before this call.
`

const claudeSwarmAgent = `---
name: ` + RunWeaverPrimaryAgentName + `
description: Primary RunWeaver workflow coordinator for Claude Code
tools: Read, Glob, Grep, Bash, Edit, MultiEdit, Write
---

You are the ` + RunWeaverPrimaryAgentName + ` coordinator for this repository.

Read CLAUDE.md, .claude/runweaver/profile.json, .runweaver/tmp/index/repo-context.md, and the latest workflow checkpoint when present. For non-trivial user tasks, call ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --runtime claude --task \"<full raw user task>\"`" + ` before coding and follow its executionContract. Do not summarize or shorten the user task before this call.

` + runWeaverStartupProtocol + `

For non-trivial work, use the repo-local shim for runweaver start as the single intake/resume entrypoint. Keep checkpoint.json and todo.md current through $RUNWEAVER_BIN workflow update. Continue through implementation and verification unless the user explicitly asks for planning only.

Use repo-specific agents from .claude/agents and repo skills from .claude/skills as local role instructions. Record participants, files read, files changed, findings, decisions, lastResult, rejectedPaths, verification results, residual risks, blockers, nextAction, and nextVerification in the checkpoint with exact workflow update flags such as --participants, --participant-rationale, --file-read, --file-changed, --last-result, --rejected-path, --verification-result, --residual-risk, --blocker, --next-action, and --next-verification.
`

const claudeContextDisciplineSkill = `---
name: context-discipline
description: "Filesystem-backed context discipline for durable RunWeaver workflows in Claude Code"
compatibility: claude
---

Use this skill for every non-trivial coding, bugfix, refactor, test, onboarding, or metadata workflow.

Maintain .runweaver/tmp/swarm-runs/<run-id>/checkpoint.json, todo.md, events.ndjson, phase notes, and verification logs through ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" workflow update`" + `. Use exact flags such as ` + "`--participants`" + `, ` + "`--participant-rationale`" + `, ` + "`--acceptance-criterion`" + `, ` + "`--file-read`" + `, ` + "`--file-changed`" + `, ` + "`--last-result`" + `, ` + "`--rejected-path`" + `, ` + "`--verification-result`" + `, ` + "`--residual-risk`" + `, ` + "`--blocker`" + `, ` + "`--next-action`" + `, and ` + "`--next-verification`" + ` whenever they explain the next move.

Before final response, compare every acceptance criterion against code, tests, and verification evidence, then run ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" workflow verify --repo . --resume latest`" + ` when feasible and record exact blockers plus the next verification step otherwise.
`

const claudeMetadataRefreshSkill = `---
name: metadata-refresh
description: "Refresh RunWeaver/Claude metadata after code structure changes"
compatibility: claude
---

Use this skill only for explicit metadata regeneration tasks. Run ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" refresh --repo . --classification deterministic`" + `, inspect .runweaver/tmp/index/repo-context.md, .runweaver/tmp/index/repo-index.compact.json, .runweaver/tmp/drift-report.json, and apply generated metadata changes only after review. Do not run metadata refresh as a routine finalization step after ordinary code/test/build work.
`

const claudeRepoOnboardingSkill = `---
name: repo-onboarding
description: "Onboard into a repository by reading configs, source layout, tests, and RunWeaver metadata"
compatibility: claude
---

Read CLAUDE.md, run ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" index --repo . --changed-only --prune`" + `, and inspect detected packages, tools, entrypoints, configs, tests, routes/pages, symbols, and verification commands.
`
