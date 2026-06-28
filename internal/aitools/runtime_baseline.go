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
	}
	if hasRuntime(runtimeIDs, RuntimeOpenCode) {
		dirs = append(dirs,
			".opencode/agents",
			".opencode/skills/context-discipline",
			".opencode/skills/metadata-refresh",
			".opencode/skills/repo-onboarding",
			".opencode/swarm",
		)
	}
	if hasRuntime(runtimeIDs, RuntimeCodex) {
		dirs = append(dirs,
			".agents/skills/context-discipline",
			".agents/skills/metadata-refresh",
			".agents/skills/repo-onboarding",
			".codex/agents",
			".codex/runweaver",
		)
	}
	if hasRuntime(runtimeIDs, RuntimeClaude) {
		dirs = append(dirs,
			".claude/agents",
			".claude/skills/context-discipline",
			".claude/skills/metadata-refresh",
			".claude/skills/repo-onboarding",
			".claude/runweaver",
		)
	}
	return dirs
}

func runtimeBaselineFiles(runtimeIDs []string) map[string]string {
	files := map[string]string{
		".runweaver/.gitignore":                             "tmp/\n",
		".runweaver/START_HERE.md":                          startHereMD,
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
		files[".opencode/agents/swarm.md"] = swarmAgent
		files[".opencode/agents/repo-classifier.md"] = repoClassifierAgent
		files[".opencode/agents/repo-surface-indexer.md"] = repoSurfaceIndexerAgent
		files[".opencode/agents/agent-skill-drift-reviewer.md"] = driftReviewerAgent
		files[".opencode/agents/profile-regenerator.md"] = profileRegeneratorAgent
		files[".opencode/agents/stale-anchor-fixer.md"] = staleAnchorFixerAgent
		files[".opencode/agents/repo-surface-engineer.md"] = repoSurfaceEngineerAgent
		files[".opencode/agents/repo-contract-reviewer.md"] = repoContractReviewerAgent
		files[".opencode/agents/repo-test-quality-reviewer.md"] = repoTestQualityReviewerAgent
		files[".opencode/skills/context-discipline/SKILL.md"] = contextDisciplineSkill
		files[".opencode/skills/metadata-refresh/SKILL.md"] = metadataRefreshSkill
		files[".opencode/skills/repo-onboarding/SKILL.md"] = repoOnboardingSkill
	}
	if hasRuntime(runtimeIDs, RuntimeCodex) {
		files["AGENTS.md"] = agentsMD
		files[".codex/agents/swarm.toml"] = codexSwarmAgentTOML
		files[".agents/skills/context-discipline/SKILL.md"] = codexContextDisciplineSkill
		files[".agents/skills/metadata-refresh/SKILL.md"] = codexMetadataRefreshSkill
		files[".agents/skills/repo-onboarding/SKILL.md"] = codexRepoOnboardingSkill
	}
	if hasRuntime(runtimeIDs, RuntimeClaude) {
		files["CLAUDE.md"] = claudeMD
		files[".claude/agents/swarm.md"] = claudeSwarmAgent
		files[".claude/skills/context-discipline/SKILL.md"] = claudeContextDisciplineSkill
		files[".claude/skills/metadata-refresh/SKILL.md"] = claudeMetadataRefreshSkill
		files[".claude/skills/repo-onboarding/SKILL.md"] = claudeRepoOnboardingSkill
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
name = "swarm"
description = "Primary RunWeaver workflow coordinator for Codex"
developer_instructions = """
You are the RunWeaver swarm coordinator for this repository.

Read AGENTS.md, .codex/runweaver/profile.json, .runweaver/tmp/index/repo-context.md, and the latest workflow checkpoint when present.

` + runWeaverStartupProtocol + `

For non-trivial work, create or resume a durable workflow under .runweaver/tmp/swarm-runs. Keep checkpoint.json and todo.md current through runweaver workflow update. Continue through implementation and verification unless the user explicitly asks for planning only.

Use repo-specific agents from .codex/agents and repo skills from .agents/skills as local role instructions. Record participants, files read, files changed, findings, decisions, lastResult, rejectedPaths, verification results, blockers, nextAction, and nextVerification in the checkpoint.
"""
`

const codexContextDisciplineSkill = `---
name: context-discipline
description: "Filesystem-backed context discipline for durable RunWeaver workflows in Codex"
compatibility: codex
---

Use this skill for every non-trivial coding, bugfix, refactor, test, onboarding, or metadata workflow.

Maintain .runweaver/tmp/swarm-runs/<run-id>/checkpoint.json, todo.md, events.ndjson, phase notes, and verification logs through runweaver workflow update. Include lastResult, rejectedPaths, nextAction, and nextVerification whenever they explain the next move.

Before final response, run runweaver workflow verify --repo . --resume latest when feasible and record exact blockers plus the next verification step otherwise.
`

const codexMetadataRefreshSkill = `---
name: metadata-refresh
description: "Refresh RunWeaver/Codex metadata after code structure changes"
compatibility: codex
---

Run runweaver refresh --repo ., inspect .runweaver/tmp/index/repo-context.md, .runweaver/tmp/index/repo-index.compact.json, .runweaver/tmp/drift-report.json, and apply generated metadata changes only after review.
`

const codexRepoOnboardingSkill = `---
name: repo-onboarding
description: "Onboard into a repository by reading configs, source layout, tests, and RunWeaver metadata"
compatibility: codex
---

Read AGENTS.md, run runweaver index --repo . --changed-only --prune, and inspect detected packages, tools, entrypoints, configs, tests, routes/pages, symbols, and verification commands.
`

const claudeMD = `# RunWeaver Repository AI Rules

RunWeaver metadata is generated and should be refreshed after code structure changes.

` + runWeaverStartupProtocol + `

Use runweaver index --repo . --changed-only --prune to refresh the local package/file/symbol index under .runweaver/tmp.

Use runweaver refresh --repo . after route, page, controller, service, test, or build config moves.

For non-trivial work, create or resume a workflow under .runweaver/tmp/swarm-runs and keep checkpoint.json current with runweaver workflow update.
`

const claudeSwarmAgent = `---
name: swarm
description: Primary RunWeaver workflow coordinator for Claude Code
tools: Read, Glob, Grep, Bash, Edit, MultiEdit, Write
---

You are the RunWeaver swarm coordinator for this repository.

Read CLAUDE.md, .claude/runweaver/profile.json, .runweaver/tmp/index/repo-context.md, and the latest workflow checkpoint when present.

` + runWeaverStartupProtocol + `

For non-trivial work, create or resume a durable workflow under .runweaver/tmp/swarm-runs. Keep checkpoint.json and todo.md current through runweaver workflow update. Continue through implementation and verification unless the user explicitly asks for planning only.

Use repo-specific agents from .claude/agents and repo skills from .claude/skills as local role instructions. Record participants, files read, files changed, findings, decisions, lastResult, rejectedPaths, verification results, blockers, nextAction, and nextVerification in the checkpoint.
`

const claudeContextDisciplineSkill = `---
name: context-discipline
description: "Filesystem-backed context discipline for durable RunWeaver workflows in Claude Code"
compatibility: claude
---

Use this skill for every non-trivial coding, bugfix, refactor, test, onboarding, or metadata workflow.

Maintain .runweaver/tmp/swarm-runs/<run-id>/checkpoint.json, todo.md, events.ndjson, phase notes, and verification logs through runweaver workflow update. Include lastResult, rejectedPaths, nextAction, and nextVerification whenever they explain the next move.

Before final response, run runweaver workflow verify --repo . --resume latest when feasible and record exact blockers plus the next verification step otherwise.
`

const claudeMetadataRefreshSkill = `---
name: metadata-refresh
description: "Refresh RunWeaver/Claude metadata after code structure changes"
compatibility: claude
---

Run runweaver refresh --repo ., inspect .runweaver/tmp/index/repo-context.md, .runweaver/tmp/index/repo-index.compact.json, .runweaver/tmp/drift-report.json, and apply generated metadata changes only after review.
`

const claudeRepoOnboardingSkill = `---
name: repo-onboarding
description: "Onboard into a repository by reading configs, source layout, tests, and RunWeaver metadata"
compatibility: claude
---

Read CLAUDE.md, run runweaver index --repo . --changed-only --prune, and inspect detected packages, tools, entrypoints, configs, tests, routes/pages, symbols, and verification commands.
`
