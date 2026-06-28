package aitools

import (
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
	"os"
	"path/filepath"
	"strings"
)

const workflowCurrentFile = "current.md"

func writeWorkflowCurrent(root, runDir string, checkpoint WorkflowCheckpoint) error {
	content := []byte(workflowCurrentMarkdown(checkpoint, rel(root, runDir)))
	currentPath := filepath.Join(runDir, workflowCurrentFile)
	if err := os.WriteFile(currentPath, content, 0o644); err != nil {
		return fmt.Errorf("write workflow current %s: %w", rel(root, currentPath), err)
	}
	latestPath := statepath.TmpPath(root, workflowCurrentFile)
	if err := os.WriteFile(latestPath, content, 0o644); err != nil {
		return fmt.Errorf("write workflow current %s: %w", rel(root, latestPath), err)
	}
	return nil
}

func workflowCurrentMarkdown(checkpoint WorkflowCheckpoint, runDir string) string {
	var b strings.Builder
	b.WriteString("# Current RunWeaver Workflow\n\n")
	writeMarkdownField(&b, "Run", checkpoint.RunID)
	writeMarkdownField(&b, "Workflow", checkpoint.Workflow)
	writeMarkdownField(&b, "Task", checkpoint.Task)
	writeMarkdownField(&b, "Status", checkpoint.Status)
	if checkpoint.CurrentPhase != "" {
		writeMarkdownField(&b, "Current phase", checkpoint.CurrentPhase)
	}
	if checkpoint.NextPhase != "" {
		writeMarkdownField(&b, "Next phase", checkpoint.NextPhase)
	}
	writeMarkdownField(&b, "Run directory", runDir)
	writeMarkdownField(&b, "Updated", checkpoint.UpdatedAt)

	if len(checkpoint.Participants) > 0 {
		b.WriteString("\n## Participants\n\n")
		writeMarkdownList(&b, checkpoint.Participants)
	}
	if checkpoint.LastResult != "" {
		b.WriteString("\n## Last Result\n\n")
		b.WriteString(checkpoint.LastResult)
		b.WriteString("\n")
	}
	if len(checkpoint.FilesChanged) > 0 {
		b.WriteString("\n## Changed Files\n\n")
		writeMarkdownList(&b, checkpoint.FilesChanged)
	}
	if len(checkpoint.RejectedPaths) > 0 {
		b.WriteString("\n## Rejected Or Paused Paths\n\n")
		writeMarkdownList(&b, checkpoint.RejectedPaths)
	}
	if checkpoint.NextAction != "" {
		b.WriteString("\n## Next Action\n\n")
		b.WriteString(checkpoint.NextAction)
		b.WriteString("\n")
	}
	if checkpoint.NextVerification != "" {
		b.WriteString("\n## Next Verification\n\n")
		b.WriteString(checkpoint.NextVerification)
		b.WriteString("\n")
	}
	if len(checkpoint.Blockers) > 0 {
		b.WriteString("\n## Blockers\n\n")
		writeMarkdownList(&b, checkpoint.Blockers)
	}
	if len(checkpoint.VerificationResults) > 0 {
		b.WriteString("\n## Verification Results\n\n")
		writeMarkdownList(&b, checkpoint.VerificationResults)
	}
	if checkpoint.IndexFreshnessStatus != "" {
		b.WriteString("\n## Index Freshness At Last Update\n\n")
		writeMarkdownField(&b, "Status", checkpoint.IndexFreshnessStatus)
		writeMarkdownField(&b, "Stale index", fmt.Sprintf("%t", checkpoint.StaleIndex))
		if len(checkpoint.StaleIndexFiles) > 0 {
			b.WriteString("\nStale files:\n\n")
			writeMarkdownList(&b, checkpoint.StaleIndexFiles)
		}
	}

	b.WriteString("\n## Agent Commands\n\n")
	b.WriteString("- Inspect JSON status: `runweaver workflow run --resume latest --status`\n")
	b.WriteString("- Update checkpoint: `runweaver workflow update --repo . --resume latest --phase <phase> ...`\n")
	b.WriteString("- Verify before finishing: `runweaver workflow verify --repo . --resume latest`\n")
	b.WriteString("\nResume automatically from this file plus `checkpoint.json`; do not ask the user to run resume commands manually.\n")
	return b.String()
}

func writeMarkdownField(b *strings.Builder, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	b.WriteString("- ")
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteString("\n")
}

func writeMarkdownList(b *strings.Builder, values []string) {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(value)
		b.WriteString("\n")
	}
}
