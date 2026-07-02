package aitools

import (
	"fmt"
	"strings"
)

func runtimeSkillMarkdown(skill SkillProfile, repo RepoProfile, runtimeID string) string {
	return `---
name: ` + skill.Name + `
description: "` + yamlEscape(skill.Description) + `"
compatibility: ` + runtimeID + `
---

` + generatedMarker + `

## RunWeaver Start Gate

Before applying this skill to a non-trivial user task, ensure the current raw user task has already been passed to ` + "`RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --task \"<full raw user task>\"`" + `. If the current session has not called ` + "`runweaver start`" + ` yet, run the generated ` + "`runweaver-start`" + ` entrypoint first, follow its ` + "`executionContract`" + `, and return to this skill only when it is selected by the workflow.

Use this skill for ` + skillUsageLabel(skill, repo) + ` work in this repository.

## Focus Files

` + markdownList(skill.FocusFiles) + `

## Workflow

` + numberedList(skill.Workflow) + `

## Risks

` + markdownList(append(repo.Risks, skill.Risks...)) + `

## Verification

` + markdownList(verificationForSkill(skill, repo)) + `

## Output Contract

Return ` + "`focus_files`, `workflow_steps`, `verification`, `risks`, and `checkpoint_update`" + ` with ` + "`lastResult`" + `, ` + "`rejectedPaths`" + `, and ` + "`nextVerification`" + ` when relevant.
`
}

func tomlQuote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return `"` + value + `"`
}

func tomlMultiline(value string) string {
	value = strings.ReplaceAll(value, `"""`, `\"\"\"`)
	return `"""` + "\n" + value + `"""`
}

func markdownList(items []string) string {
	if len(items) == 0 {
		return "- None detected yet; refresh the surface index before assuming absence.\n"
	}
	var b strings.Builder
	for _, item := range items {
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteString("\n")
	}
	return b.String()
}

func numberedList(items []string) string {
	if len(items) == 0 {
		return "1. Refresh the surface index and derive the workflow from current repository evidence.\n"
	}
	var b strings.Builder
	for i, item := range items {
		fmt.Fprintf(&b, "%d. %s\n", i+1, item)
	}
	return b.String()
}

func yamlEscape(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}
