package aitools

import (
	"path/filepath"
)

func relOrEmpty(root, path string) string {
	if path == "" {
		return ""
	}
	return rel(root, path)
}

func workflowExecutionPostCheck(root string, plan WorkflowRunSummary) WorkflowPostCheck {
	check := WorkflowPostCheck{Status: "ok", Summary: "workflow checkpoint advanced or no phases require progress evidence"}
	var checkpoint WorkflowCheckpoint
	if err := ReadJSON(filepath.Join(root, plan.CheckpointPath), &checkpoint); err != nil {
		check.Status = "warning"
		check.Summary = "workflow checkpoint could not be read after execution"
		check.Warnings = []string{err.Error()}
		return check
	}
	check.CheckpointState = checkpoint.Status
	check.NextPhase = checkpoint.NextPhase
	check.CompletedPhases = len(checkpoint.CompletedPhases)
	if len(plan.Phases) > 0 && check.CompletedPhases == 0 && (check.CheckpointState == "" || check.CheckpointState == "planned") {
		check.Status = "warning"
		check.Summary = "workflow checkpoint still looks unstarted after runtime execution"
		check.Warnings = []string{"checkpoint completedPhases is empty and status is still planned"}
	}
	return check
}

func intPtr(value int) *int {
	return &value
}
