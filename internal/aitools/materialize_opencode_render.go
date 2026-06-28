package aitools

func agentMarkdown(agent AgentProfile, repo RepoProfile) string {
	focusFiles := agent.FocusFiles
	if len(focusFiles) == 0 {
		focusFiles = repo.KeyFiles
	}
	verification := agent.Verification
	if len(verification) == 0 {
		for _, skill := range repo.CustomSkills {
			verification = append(verification, skill.Verification...)
		}
		verification = Unique(verification)
	}
	return `---
description: "` + yamlEscape(agent.Description) + `"
mode: subagent
permission:
  edit: allow
  bash:
    "*": allow
    "runweaver scan *": allow
    "runweaver index *": allow
    "runweaver refresh *": allow
    "rg *": allow
    "find *": allow
    "ls *": allow
    "ls": allow
    "pwd": allow
    "git diff*": allow
  skill:
    "*": allow
---

` + generatedMarker + `

You are ` + agent.Name + ` for this repository.

Repository kind: ` + repo.Kind + `
Runtime: ` + repo.Runtime + `
Domain: ` + repo.Domain + `

## Operating Rules

1. Start from ` + "`AGENTS.md`" + `, ` + "`.opencode/swarm/profile.json`" + `, ` + "`.runweaver/tmp/index/repo-context.md`" + `, and ` + "`.runweaver/tmp/index/repo-index.compact.json`" + ` when they exist.
2. Prefer current source discovery with ` + "`runweaver index --repo . --changed-only --prune`" + ` and ` + "`rg --files`" + ` before relying on stale file anchors. Use ` + "`runweaver scan --repo . --out .runweaver/tmp/surface-index.json`" + ` only when a legacy surface-index artifact is required.
3. Apply the ` + "`context-discipline`" + ` skill when participating in a workflow run. Record files read, files changed, decisions, artifacts, last result, rejected or paused paths, next verification, verification results, blockers, and participant rationale through ` + "`runweaver workflow update`" + `.
4. Keep edits scoped to the task and to the surfaces listed below unless the code path proves a broader change is required.
5. If routes, pages, tests, configs, or major directories move, request or run metadata refresh.

## Key Files And Directories

` + markdownList(focusFiles) + `

## Role Workflow

` + numberedList(agent.Workflow) + `

## Risks

` + markdownList(repo.Risks) + `

## Verification

` + markdownList(verification) + `

## Output Contract

Return ` + "`files_read`, `files_changed`, `contracts_checked`, `verification`, `risks`, and `checkpoint_update`." + `

` + "`checkpoint_update`" + ` must include concrete ` + "`participants`" + `, ` + "`participantRationale`" + `, ` + "`findings`" + `, ` + "`decisions`" + `, ` + "`filesRead`" + `, ` + "`filesChanged`" + `, ` + "`artifacts`" + `, ` + "`lastResult`" + `, ` + "`rejectedPaths`" + `, ` + "`verification`" + `, ` + "`verificationResults`" + `, ` + "`nextVerification`" + `, ` + "`blockers`" + `, and ` + "`nextAction`" + ` fields when they apply. Do not return a vague prose-only checkpoint update.
`
}

func skillMarkdown(skill SkillProfile, repo RepoProfile) string {
	return `---
name: ` + skill.Name + `
description: "` + yamlEscape(skill.Description) + `"
compatibility: opencode
---

` + generatedMarker + `

Use this skill for ` + repo.Kind + ` work in this repository.

## Focus Files

` + markdownList(skill.FocusFiles) + `

## Workflow

` + numberedList(skill.Workflow) + `

## Risks

` + markdownList(append(repo.Risks, skill.Risks...)) + `

## Verification

` + markdownList(skill.Verification) + `

## Output Contract

Return ` + "`focus_files`, `workflow_steps`, `verification`, `risks`, and `checkpoint_update`." + `

` + "`checkpoint_update`" + ` must name concrete ` + "`filesRead`" + `, ` + "`filesChanged`" + `, ` + "`findings`" + `, ` + "`decisions`" + `, ` + "`artifacts`" + `, ` + "`lastResult`" + `, ` + "`rejectedPaths`" + `, ` + "`verificationResults`" + `, ` + "`nextVerification`" + `, and ` + "`blockers`" + ` when they apply.
`
}
