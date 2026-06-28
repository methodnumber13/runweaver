package aitools

func codexAgentTOML(agent AgentProfile, repo RepoProfile) string {
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
	instructions := generatedMarker + "\n\n" +
		"You are " + agent.Name + " for this repository.\n\n" +
		"Repository kind: " + repo.Kind + "\n" +
		"Runtime: " + repo.Runtime + "\n" +
		"Domain: " + repo.Domain + "\n\n" +
		"Start from AGENTS.md, .codex/runweaver/profile.json, .runweaver/tmp/index/repo-context.md, and .runweaver/tmp/index/repo-index.compact.json when they exist.\n" +
		"Use runweaver index --repo . --changed-only --prune before relying on stale file anchors. Apply context-discipline and persist workflow state with runweaver workflow update, including lastResult, rejectedPaths, nextAction, and nextVerification when they explain the next move.\n\n" +
		"Key files and directories:\n" + markdownList(focusFiles) + "\n" +
		"Role workflow:\n" + numberedList(agent.Workflow) + "\n" +
		"Risks:\n" + markdownList(repo.Risks) + "\n" +
		"Verification:\n" + markdownList(verification) + "\n" +
		"Output contract: return files_read, files_changed, contracts_checked, verification, risks, and checkpoint_update with concrete checkpoint fields, including lastResult, rejectedPaths, and nextVerification when relevant.\n"
	return "# " + generatedMarker + "\n" +
		"name = " + tomlQuote(agent.Name) + "\n" +
		"description = " + tomlQuote(agent.Description) + "\n" +
		"developer_instructions = " + tomlMultiline(instructions) + "\n"
}

func codexSkillMarkdown(skill SkillProfile, repo RepoProfile) string {
	return runtimeSkillMarkdown(skill, repo, RuntimeCodex)
}
