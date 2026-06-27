package aitools

import (
	"fmt"
)

func verifyParticipantBudget(workflow WorkflowSpec, checkpoint WorkflowCheckpoint, result *WorkflowVerificationResult) {
	cap := workflowParticipantCap(workflow)
	if len(checkpoint.Participants) == 0 {
		status := "warning"
		summary := "No participants recorded in checkpoint"
		if checkpoint.Status == "planned" {
			status = "ok"
			summary = "No participants recorded yet for planned workflow"
		}
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:     "participant-budget",
			Status:   status,
			Summary:  summary,
			Evidence: []string{fmt.Sprintf("cap: %d", cap)},
		})
		return
	}
	if len(checkpoint.Participants) > cap {
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:        "participant-budget",
			Status:      "warning",
			Summary:     "Recorded participants exceed workflow participant cap",
			Evidence:    []string{fmt.Sprintf("participants: %d", len(checkpoint.Participants)), fmt.Sprintf("cap: %d", cap)},
			NextActions: []string{"Prefer one domain owner plus up to two reviewers/skills unless the workflow explicitly raises maxParticipants."},
		})
		return
	}
	if len(checkpoint.ParticipantRationale) == 0 && checkpoint.Status != "planned" {
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:        "participant-budget",
			Status:      "warning",
			Summary:     "Participants are recorded but rationale is missing",
			Evidence:    checkpoint.Participants,
			NextActions: []string{"Record --participant-rationale entries explaining domain owner and reviewer/skill selection."},
		})
		return
	}
	addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
		Name:     "participant-budget",
		Status:   "ok",
		Summary:  "Participant count is within cap and rationale is present",
		Evidence: []string{fmt.Sprintf("participants: %d", len(checkpoint.Participants)), fmt.Sprintf("cap: %d", cap)},
	})
}

func verifyTerminalEvidence(workflow WorkflowSpec, checkpoint WorkflowCheckpoint, result *WorkflowVerificationResult) {
	if checkpoint.Status != "complete" {
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:     "terminal-evidence",
			Status:   "warning",
			Summary:  "Workflow has not reached a terminal complete state",
			Evidence: []string{"status: " + checkpoint.Status},
		})
		return
	}
	if len(checkpoint.Blockers) > 0 {
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:     "terminal-evidence",
			Status:   "warning",
			Summary:  "Workflow completed with recorded blockers",
			Evidence: checkpoint.Blockers,
		})
		return
	}
	if workflowHasWritePhase(workflow) && len(checkpoint.FilesChanged) == 0 {
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:        "terminal-evidence",
			Status:      "warning",
			Summary:     "Workflow has write phases but no changed files were recorded",
			NextActions: []string{"Record changed files with runweaver workflow update --file-changed <path> or record a blocker if no edit was possible."},
		})
		return
	}
	if len(checkpoint.Verification) == 0 || len(checkpoint.VerificationResults) == 0 {
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:     "terminal-evidence",
			Status:   "warning",
			Summary:  "Workflow is complete but verification commands or results are missing",
			Evidence: []string{fmt.Sprintf("verification commands: %d", len(checkpoint.Verification)), fmt.Sprintf("verification results: %d", len(checkpoint.VerificationResults))},
		})
		return
	}
	addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
		Name:     "terminal-evidence",
		Status:   "ok",
		Summary:  "Terminal workflow evidence is present",
		Evidence: []string{fmt.Sprintf("verification commands: %d", len(checkpoint.Verification)), fmt.Sprintf("verification results: %d", len(checkpoint.VerificationResults))},
	})
}
