package aitools

import (
	"strings"
)

func todoMarkdown(workflow WorkflowSpec, task string) string {
	return todoMarkdownWithState(workflow, task, nil, "", "")
}

func todoMarkdownWithCheckpoint(workflow WorkflowSpec, checkpoint WorkflowCheckpoint) string {
	activePhase := checkpoint.CurrentPhase
	if activePhase == "" {
		activePhase = checkpoint.NextPhase
	}
	return todoMarkdownWithState(workflow, checkpoint.Task, checkpoint.CompletedPhases, activePhase, checkpoint.Status)
}

func todoMarkdownWithState(workflow WorkflowSpec, task string, completedPhases []string, activePhase, status string) string {
	completed := map[string]bool{}
	for _, phase := range completedPhases {
		completed[phase] = true
	}
	var b strings.Builder
	b.WriteString("# Workflow Todo\n\n")
	b.WriteString("Task: ")
	b.WriteString(task)
	b.WriteString("\n\n")
	for _, phase := range workflow.Phases {
		marker := " "
		if completed[phase.ID] {
			marker = "x"
		} else if status != "complete" && phase.ID == activePhase {
			marker = ">"
		}
		b.WriteString("- [")
		b.WriteString(marker)
		b.WriteString("] ")
		b.WriteString(phase.ID)
		if phase.Name != "" {
			b.WriteString(" - ")
			b.WriteString(phase.Name)
		}
		if len(phase.Agents) > 0 {
			b.WriteString(" (")
			b.WriteString(strings.Join(phase.Agents, ", "))
			b.WriteString(")")
		}
		b.WriteString("\n")
	}
	return b.String()
}
