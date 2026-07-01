package aitools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyWorkflowRunPassesForCompleteConsistentRun(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "maxParticipants": 3,
  "phases": [
    {"id": "plan", "name": "Plan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "plan"},
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	if _, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "map repo"); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:               "latest",
		Phase:                "plan",
		Participants:         []string{"repo-surface-indexer", "repo-test-quality-reviewer"},
		ParticipantRationale: []string{"surface owner plus test reviewer"},
		FilesRead:            []string{"go.mod"},
		Findings:             []string{"repo uses Go"},
		Decisions:            []string{"use go test ./..."},
		Artifacts:            []string{".runweaver/tmp/swarm-runs/latest/phases/plan/handoff.md"},
		CompletePhase:        true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:              "latest",
		Phase:               "verify",
		Verification:        []string{"go test ./..."},
		VerificationResults: []string{"go test ./... passed"},
		Artifacts:           []string{".runweaver/tmp/swarm-runs/latest/phases/verify/verification.jsonl"},
		CompletePhase:       true,
	}); err != nil {
		t.Fatal(err)
	}
	result, err := VerifyWorkflowRun(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ok" {
		t.Fatalf("verification = %#v, want ready ok", result)
	}
}

func TestVerifyWorkflowRunNormalizesCompletedResumeHints(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "maxParticipants": 3,
  "phases": [
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	run, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "map repo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:               "latest",
		Phase:                "verify",
		Participants:         []string{"repo-test-quality-reviewer"},
		ParticipantRationale: []string{"test reviewer owns verification"},
		Verification:         []string{"go test ./..."},
		VerificationResults:  []string{"go test ./... passed"},
		CompletePhase:        true,
	}); err != nil {
		t.Fatal(err)
	}

	var checkpoint WorkflowCheckpoint
	checkpointPath := filepath.Join(root, run.CheckpointPath)
	if err := ReadJSON(checkpointPath, &checkpoint); err != nil {
		t.Fatal(err)
	}
	checkpoint.NextAction = "rerun verify"
	checkpoint.NextVerification = "run runweaver workflow verify --repo . --resume latest"
	if err := WriteJSON(checkpointPath, checkpoint); err != nil {
		t.Fatal(err)
	}
	runDir := filepath.Join(root, run.RunDir)
	if err := syncWorkflowTodo(root, runDir, checkpoint); err != nil {
		t.Fatal(err)
	}

	result, err := VerifyWorkflowRun(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ok" {
		t.Fatalf("verification = %#v, want ready ok", result)
	}
	checkpoint = WorkflowCheckpoint{}
	if err := ReadJSON(checkpointPath, &checkpoint); err != nil {
		t.Fatal(err)
	}
	if checkpoint.NextAction != completeWorkflowNextAction || checkpoint.NextVerification != "" {
		t.Fatalf("checkpoint = %#v, want normalized terminal hints", checkpoint)
	}
}

func TestVerifyWorkflowRunWarnsWhenVerificationWarningsLackResidualRisk(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "maxParticipants": 3,
  "phases": [
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	run, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "map repo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:               "latest",
		Phase:                "verify",
		Participants:         []string{"repo-test-quality-reviewer"},
		ParticipantRationale: []string{"test reviewer owns verification"},
		Verification:         []string{"tool test"},
		VerificationResults:  []string{"tool test passed"},
		CompletePhase:        true,
	}); err != nil {
		t.Fatal(err)
	}

	var checkpoint WorkflowCheckpoint
	checkpointPath := filepath.Join(root, run.CheckpointPath)
	if err := ReadJSON(checkpointPath, &checkpoint); err != nil {
		t.Fatal(err)
	}
	checkpoint.VerificationResults = []string{"tool test passed\nWarning: non-fatal async boundary warning"}
	checkpoint.ResidualRisks = nil
	if err := WriteJSON(checkpointPath, checkpoint); err != nil {
		t.Fatal(err)
	}
	if err := syncWorkflowTodo(root, filepath.Join(root, run.RunDir), checkpoint); err != nil {
		t.Fatal(err)
	}

	result, err := VerifyWorkflowRun(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status != "warning" {
		t.Fatalf("verification = %#v, want warning while residual risk is missing", result)
	}
	if !workflowCheckSelected(result.Checks, "verification-warnings") {
		t.Fatalf("checks = %#v, want verification-warnings check", result.Checks)
	}
}

func TestVerifyWorkflowRunNormalizesCompletedResumeHintsEvenWithWarnings(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "maxParticipants": 3,
  "phases": [
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	run, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "map repo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:              "latest",
		Phase:               "verify",
		Participants:        []string{"repo-test-quality-reviewer"},
		Verification:        []string{"go test ./..."},
		VerificationResults: []string{"go test ./... passed"},
		CompletePhase:       true,
	}); err != nil {
		t.Fatal(err)
	}

	checkpointPath := filepath.Join(root, run.CheckpointPath)
	var checkpoint WorkflowCheckpoint
	if err := ReadJSON(checkpointPath, &checkpoint); err != nil {
		t.Fatal(err)
	}
	checkpoint.NextAction = "rerun verify"
	checkpoint.NextVerification = "run runweaver workflow verify --repo . --resume latest"
	if err := WriteJSON(checkpointPath, checkpoint); err != nil {
		t.Fatal(err)
	}
	if err := syncWorkflowTodo(root, filepath.Join(root, run.RunDir), checkpoint); err != nil {
		t.Fatal(err)
	}

	result, err := VerifyWorkflowRun(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status != "warning" {
		t.Fatalf("verification = %#v, want warning from missing participant rationale", result)
	}
	checkpoint = WorkflowCheckpoint{}
	if err := ReadJSON(checkpointPath, &checkpoint); err != nil {
		t.Fatal(err)
	}
	if checkpoint.NextAction != completeWorkflowNextAction || checkpoint.NextVerification != "" {
		t.Fatalf("checkpoint = %#v, want normalized terminal hints despite verifier warning", checkpoint)
	}
}

func TestVerifyWorkflowRunWarnsForIncompleteRun(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {"id": "scan", "name": "Scan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "scan"}
  ]
}`)
	if _, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "map repo"); err != nil {
		t.Fatal(err)
	}
	result, err := VerifyWorkflowRun(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status != "warning" {
		t.Fatalf("verification = %#v, want warning not ready", result)
	}
}

func TestVerifyWorkflowRunErrorsOnMissingPlan(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {"id": "scan", "name": "Scan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "scan"}
  ]
}`)
	run, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "map repo")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, run.RunDir, "plan.json")); err != nil {
		t.Fatal(err)
	}
	result, err := VerifyWorkflowRun(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status != "error" {
		t.Fatalf("verification = %#v, want error not ready", result)
	}
}

func workflowCheckSelected(checks []WorkflowVerificationCheck, name string) bool {
	for _, check := range checks {
		if check.Name == name {
			return true
		}
	}
	return false
}
