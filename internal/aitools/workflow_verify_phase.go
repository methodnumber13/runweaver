package aitools

import (
	"path/filepath"
)

func verifyPhaseArtifacts(root, runDir string, workflow WorkflowSpec, result *WorkflowVerificationResult) {
	missing := []string{}
	for _, phase := range workflow.Phases {
		phaseDir := filepath.Join(runDir, "phases", sanitizeRunID(phase.ID))
		for _, name := range []string{"handoff.md", "notes.md", "verification.jsonl"} {
			path := filepath.Join(phaseDir, name)
			if !Exists(path) {
				missing = append(missing, rel(root, path))
			}
		}
	}
	if len(missing) > 0 {
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:        "phase-artifacts",
			Status:      "warning",
			Summary:     "Some run-local phase artifact files are missing",
			Evidence:    missing,
			NextActions: []string{"Create missing phase artifacts under .runweaver/tmp/swarm-runs/<run>/phases/<phase>/ before relying on resume context."},
		})
		return
	}
	addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
		Name:     "phase-artifacts",
		Status:   "ok",
		Summary:  "Run-local phase artifact files exist",
		Evidence: []string{rel(root, filepath.Join(runDir, "phases"))},
	})
}

func verifyPhaseState(workflow WorkflowSpec, checkpoint WorkflowCheckpoint, result *WorkflowVerificationResult) {
	phaseIDs := workflowPhaseIDs(workflow)
	phaseSet := map[string]bool{}
	for _, phaseID := range phaseIDs {
		phaseSet[phaseID] = true
	}
	unknown := []string{}
	for _, phase := range checkpoint.CompletedPhases {
		if !phaseSet[phase] {
			unknown = append(unknown, phase)
		}
	}
	if checkpoint.CurrentPhase != "" && !phaseSet[checkpoint.CurrentPhase] {
		unknown = append(unknown, checkpoint.CurrentPhase)
	}
	if checkpoint.NextPhase != "" && !phaseSet[checkpoint.NextPhase] {
		unknown = append(unknown, checkpoint.NextPhase)
	}
	if len(unknown) > 0 {
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:     "phase-state",
			Status:   "error",
			Summary:  "Checkpoint references phases that are not in plan.json",
			Evidence: Unique(unknown),
		})
		return
	}
	expectedNext := expectedNextPhase(phaseIDs, checkpoint.CompletedPhases)
	if checkpoint.Status == "complete" && expectedNext != "" {
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:     "phase-state",
			Status:   "error",
			Summary:  "Checkpoint is complete but not all plan phases are completed",
			Evidence: []string{"next expected phase: " + expectedNext},
		})
		return
	}
	if checkpoint.Status != "complete" && checkpoint.NextPhase != "" && expectedNext != "" && checkpoint.NextPhase != expectedNext {
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:     "phase-state",
			Status:   "warning",
			Summary:  "Checkpoint nextPhase does not match the first incomplete plan phase",
			Evidence: []string{"checkpoint nextPhase: " + checkpoint.NextPhase, "expected nextPhase: " + expectedNext},
		})
		return
	}
	if checkpoint.Status != "complete" {
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:     "phase-state",
			Status:   "warning",
			Summary:  "Workflow is not complete yet",
			Evidence: []string{"status: " + checkpoint.Status},
		})
		return
	}
	addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
		Name:     "phase-state",
		Status:   "ok",
		Summary:  "Checkpoint phase state is consistent with plan.json",
		Evidence: checkpoint.CompletedPhases,
	})
}
