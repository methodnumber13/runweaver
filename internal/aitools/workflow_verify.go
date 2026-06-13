package aitools

import (
	"fmt"
	"os"
	"path/filepath"
)

// VerifyWorkflowRun checks whether a workflow run has enough evidence to finish.
func VerifyWorkflowRun(repoPath, resume string) (WorkflowVerificationResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return WorkflowVerificationResult{}, err
	}
	result := WorkflowVerificationResult{
		Status:   "ok",
		Ready:    true,
		RepoRoot: root,
		RunDir:   "",
		Checks:   []WorkflowVerificationCheck{},
	}
	runDir, err := resolveWorkflowRunDir(root, resume)
	if err != nil {
		addWorkflowVerificationCheck(&result, WorkflowVerificationCheck{
			Name:        "run-dir",
			Status:      "error",
			Summary:     "Workflow run directory could not be resolved",
			Evidence:    []string{err.Error()},
			NextActions: []string{"Create a run with runweaver workflow run --workflow <file> --task <text>."},
		})
		return finalizeWorkflowVerification(result), nil
	}
	result.RunDir = rel(root, runDir)
	if !Exists(runDir) {
		addWorkflowVerificationCheck(&result, WorkflowVerificationCheck{
			Name:     "run-dir",
			Status:   "error",
			Summary:  "Workflow run directory is missing",
			Evidence: []string{result.RunDir},
		})
		return finalizeWorkflowVerification(result), nil
	}
	addWorkflowVerificationCheck(&result, WorkflowVerificationCheck{
		Name:     "run-dir",
		Status:   "ok",
		Summary:  "Workflow run directory exists",
		Evidence: []string{result.RunDir},
	})

	var plan WorkflowPlanFile
	planPath := filepath.Join(runDir, "plan.json")
	planOK := true
	if err := ReadJSON(planPath, &plan); err != nil {
		planOK = false
		addWorkflowVerificationCheck(&result, WorkflowVerificationCheck{
			Name:        "plan",
			Status:      "error",
			Summary:     "plan.json is missing or invalid",
			Evidence:    []string{err.Error()},
			NextActions: []string{"Recreate the workflow run; plan.json is treated as immutable run intent."},
		})
	} else if err := validateWorkflowSpec(plan.Workflow); err != nil {
		planOK = false
		addWorkflowVerificationCheck(&result, WorkflowVerificationCheck{
			Name:     "plan",
			Status:   "error",
			Summary:  "Workflow plan contains an invalid workflow spec",
			Evidence: []string{err.Error()},
		})
	} else {
		addWorkflowVerificationCheck(&result, WorkflowVerificationCheck{
			Name:     "plan",
			Status:   "ok",
			Summary:  "plan.json is readable and workflow spec is valid",
			Evidence: []string{rel(root, planPath)},
		})
	}

	var checkpoint WorkflowCheckpoint
	checkpointPath := filepath.Join(runDir, "checkpoint.json")
	checkpointOK := true
	if err := ReadJSON(checkpointPath, &checkpoint); err != nil {
		checkpointOK = false
		addWorkflowVerificationCheck(&result, WorkflowVerificationCheck{
			Name:     "checkpoint",
			Status:   "error",
			Summary:  "checkpoint.json is missing or invalid",
			Evidence: []string{err.Error()},
		})
	} else {
		addWorkflowVerificationCheck(&result, WorkflowVerificationCheck{
			Name:     "checkpoint",
			Status:   "ok",
			Summary:  "checkpoint.json is readable",
			Evidence: []string{rel(root, checkpointPath)},
		})
	}

	todoPath := filepath.Join(runDir, "todo.md")
	if data, err := os.ReadFile(todoPath); err != nil {
		addWorkflowVerificationCheck(&result, WorkflowVerificationCheck{
			Name:     "todo",
			Status:   "error",
			Summary:  "todo.md is missing or unreadable",
			Evidence: []string{err.Error()},
		})
	} else if planOK && checkpointOK {
		expected := todoMarkdownWithCheckpoint(plan.Workflow, checkpoint)
		if string(data) != expected {
			addWorkflowVerificationCheck(&result, WorkflowVerificationCheck{
				Name:        "todo",
				Status:      "warning",
				Summary:     "todo.md does not match checkpoint phase state",
				Evidence:    []string{rel(root, todoPath)},
				NextActions: []string{"Run runweaver workflow update on the active phase so todo.md is regenerated from checkpoint.json."},
			})
		} else {
			addWorkflowVerificationCheck(&result, WorkflowVerificationCheck{
				Name:     "todo",
				Status:   "ok",
				Summary:  "todo.md matches checkpoint phase state",
				Evidence: []string{rel(root, todoPath)},
			})
		}
	} else {
		addWorkflowVerificationCheck(&result, WorkflowVerificationCheck{
			Name:     "todo",
			Status:   "warning",
			Summary:  "todo.md could not be compared because plan or checkpoint failed",
			Evidence: []string{rel(root, todoPath)},
		})
	}

	eventsPath := filepath.Join(runDir, "events.ndjson")
	eventCount, eventErr := countWorkflowEvents(eventsPath)
	if eventErr != nil {
		addWorkflowVerificationCheck(&result, WorkflowVerificationCheck{
			Name:     "events",
			Status:   "error",
			Summary:  "events.ndjson is missing or invalid",
			Evidence: []string{eventErr.Error()},
		})
	} else {
		addWorkflowVerificationCheck(&result, WorkflowVerificationCheck{
			Name:     "events",
			Status:   "ok",
			Summary:  fmt.Sprintf("events.ndjson has %d event(s)", eventCount),
			Evidence: []string{rel(root, eventsPath)},
		})
	}

	if planOK {
		verifyPhaseArtifacts(root, runDir, plan.Workflow, &result)
	}
	if planOK && checkpointOK {
		verifyPhaseState(plan.Workflow, checkpoint, &result)
		verifyParticipantBudget(plan.Workflow, checkpoint, &result)
		verifyTerminalEvidence(plan.Workflow, checkpoint, &result)
	}
	return finalizeWorkflowVerification(result), nil
}
