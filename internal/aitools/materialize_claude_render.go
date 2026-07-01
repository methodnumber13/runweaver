package aitools

func claudeAgentMarkdown(agent AgentProfile, repo RepoProfile) string {
	focusFiles := agent.FocusFiles
	if len(focusFiles) == 0 {
		focusFiles = repo.KeyFiles
	}
	verification := renderedAgentVerification(agent, repo)
	return `---
name: ` + agent.Name + `
description: "` + yamlEscape(agent.Description) + `"
tools: Read, Glob, Grep, Bash, Edit, MultiEdit, Write
---

` + generatedMarker + `

You are ` + agent.Name + ` for this repository.

Repository kind: ` + repo.Kind + `
Runtime: ` + repo.Runtime + `
Domain: ` + repo.Domain + `

Start from ` + "`CLAUDE.md`" + `, ` + "`.claude/runweaver/profile.json`" + `, ` + "`.runweaver/tmp/index/repo-context.md`" + `, and ` + "`.runweaver/tmp/index/repo-index.compact.json`" + ` when they exist.
Use ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" index --repo . --changed-only --prune`" + ` before relying on stale file anchors. Apply ` + "`context-discipline`" + ` and persist workflow state with ` + "`$RUNWEAVER_BIN workflow update`" + ` using exact flags such as ` + "`--file-read`" + `, ` + "`--file-changed`" + `, ` + "`--last-result`" + `, ` + "`--rejected-path`" + `, ` + "`--verification-result`" + `, ` + "`--residual-risk`" + `, ` + "`--blocker`" + `, ` + "`--next-action`" + `, and ` + "`--next-verification`" + `.

## Key Files And Directories

` + markdownList(focusFiles) + `

## Role Workflow

` + numberedList(agent.Workflow) + `

## Risks

` + markdownList(repo.Risks) + `

## Verification

` + markdownList(verification) + `

## Output Contract

Return ` + "`files_read`, `files_changed`, `contracts_checked`, `verification`, `risks`, and `checkpoint_update`" + ` with concrete checkpoint fields, including ` + "`lastResult`" + `, ` + "`rejectedPaths`" + `, and ` + "`nextVerification`" + ` when relevant.
`
}

func claudeSkillMarkdown(skill SkillProfile, repo RepoProfile) string {
	return runtimeSkillMarkdown(skill, repo, RuntimeClaude)
}
