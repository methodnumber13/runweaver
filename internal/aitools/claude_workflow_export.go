package aitools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ExportClaudeWorkflows writes optional Claude-native workflow wrappers.
func ExportClaudeWorkflows(root string, force bool) error {
	paths, err := workflowTemplatePaths(root)
	if err != nil {
		return err
	}
	for _, path := range paths {
		var workflow WorkflowSpec
		if err := ReadJSON(path, &workflow); err != nil {
			return fmt.Errorf("load workflow for Claude export %s: %w", rel(root, path), err)
		}
		if workflow.ID == "" {
			continue
		}
		out := filepath.Join(root, ".claude", "workflows", workflow.ID+".md")
		if err := writeGenerated(out, claudeWorkflowMarkdown(workflow), force); err != nil {
			return err
		}
	}
	return nil
}

func claudeWorkflowMarkdown(workflow WorkflowSpec) string {
	var b strings.Builder
	b.WriteString("# ")
	if workflow.Name != "" {
		b.WriteString(workflow.Name)
	} else {
		b.WriteString(workflow.ID)
	}
	b.WriteString("\n\n")
	b.WriteString("RunWeaver portable source of truth: `.runweaver/workflows/")
	b.WriteString(workflow.ID)
	b.WriteString(".json`.\n\n")
	b.WriteString("Start or resume this workflow with:\n\n")
	b.WriteString("`runweaver start --repo . --runtime claude --workflow .runweaver/workflows/")
	b.WriteString(workflow.ID)
	b.WriteString(".json --task \"<user task>\"`\n\n")
	if workflow.Description != "" {
		b.WriteString("## Description\n\n")
		b.WriteString(workflow.Description)
		b.WriteString("\n\n")
	}
	if len(workflow.Phases) > 0 {
		b.WriteString("## Phases\n\n")
		for _, phase := range workflow.Phases {
			b.WriteString("- `")
			b.WriteString(phase.ID)
			b.WriteString("`: ")
			if phase.Name != "" {
				b.WriteString(phase.Name)
			} else {
				b.WriteString(phase.Scope)
			}
			if len(phase.Agents) > 0 {
				b.WriteString(" (agents: ")
				b.WriteString(strings.Join(phase.Agents, ", "))
				b.WriteString(")")
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\nFollow `executionContract`; keep checkpoint state in `.runweaver/tmp/swarm-runs` with `runweaver workflow update`.\n")
	return b.String()
}
