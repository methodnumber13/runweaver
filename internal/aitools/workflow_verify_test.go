package aitools

import (
	"os"
	"path/filepath"
	"strings"
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

func TestVerifyWorkflowRunWarnsWhenCommandEvidenceLacksStructuredResults(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "maxParticipants": 3,
  "phases": [
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	run, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "check evidence")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:               "latest",
		Phase:                "verify",
		Participants:         []string{"repo-test-quality-reviewer"},
		ParticipantRationale: []string{"verification owner"},
		Verification:         []string{"focused checks passed; static check passed"},
		VerificationResults:  []string{"temporary result"},
		CompletePhase:        true,
	}); err != nil {
		t.Fatal(err)
	}
	checkpointPath := filepath.Join(root, run.CheckpointPath)
	var checkpoint WorkflowCheckpoint
	if err := ReadJSON(checkpointPath, &checkpoint); err != nil {
		t.Fatal(err)
	}
	checkpoint.VerificationResults = nil
	if err := WriteJSON(checkpointPath, checkpoint); err != nil {
		t.Fatal(err)
	}

	result, err := VerifyWorkflowRun(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status != "warning" {
		t.Fatalf("verification = %#v, want warning without structured verification results", result)
	}
	checkpoint = WorkflowCheckpoint{}
	if err := ReadJSON(checkpointPath, &checkpoint); err != nil {
		t.Fatal(err)
	}
	if len(checkpoint.VerificationResults) != 0 {
		t.Fatalf("checkpoint = %#v, want command evidence not promoted into verification results", checkpoint)
	}
}

func TestVerifyWorkflowRunUsesPhaseVerificationLogEvidence(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "maxParticipants": 3,
  "phases": [
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	run, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "check phase evidence")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:               "latest",
		Phase:                "verify",
		Participants:         []string{"repo-test-quality-reviewer"},
		ParticipantRationale: []string{"verification owner"},
		Verification:         []string{"focused-check-command"},
		VerificationResults:  []string{"temporary result"},
		CompletePhase:        true,
	}); err != nil {
		t.Fatal(err)
	}
	checkpointPath := filepath.Join(root, run.CheckpointPath)
	var checkpoint WorkflowCheckpoint
	if err := ReadJSON(checkpointPath, &checkpoint); err != nil {
		t.Fatal(err)
	}
	checkpoint.VerificationResults = nil
	if err := WriteJSON(checkpointPath, checkpoint); err != nil {
		t.Fatal(err)
	}
	verificationLog := filepath.Join(root, run.RunDir, "phases", "verify", "verification.jsonl")
	if err := os.WriteFile(verificationLog, []byte(`{"command":"focused-check-command","result":"passed"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := VerifyWorkflowRun(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ok" {
		t.Fatalf("verification = %#v, want ready ok from phase verification log evidence", result)
	}
	checkpoint = WorkflowCheckpoint{}
	if err := ReadJSON(checkpointPath, &checkpoint); err != nil {
		t.Fatal(err)
	}
	if len(checkpoint.VerificationResults) != 1 || !strings.Contains(checkpoint.VerificationResults[0], "phase verify verification") {
		t.Fatalf("checkpoint = %#v, want persisted phase verification evidence", checkpoint)
	}
}

func TestVerificationEntryLooksLikeResultAvoidsSubstringMatches(t *testing.T) {
	for _, entry := range []string{"check auth bypass command", "validate password command"} {
		if verificationEntryLooksLikeResult(entry) {
			t.Fatalf("entry %q looked like a result, want command-only", entry)
		}
	}
	for _, entry := range []string{"focused check passed", "command failed with exit code 1"} {
		if !verificationEntryLooksLikeResult(entry) {
			t.Fatalf("entry %q did not look like a result", entry)
		}
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

func TestVerifyWorkflowRunAutoCompletesWhenVerificationEvidenceIsRecorded(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/feature-delivery-swarm.json", `{
  "id": "feature-delivery-swarm",
  "name": "Feature Delivery",
  "maxParticipants": 4,
  "phases": [
    {"id": "plan", "name": "Plan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "plan"},
    {"id": "implement", "name": "Implement", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["repo-surface-engineer"], "prompt": "implement"},
    {"id": "review", "name": "Review", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["post-implementation-reviewer"], "prompt": "review"}
  ]
}`)
	run, err := PlanWorkflow(root, ".runweaver/workflows/feature-delivery-swarm.json", "implement launch readiness")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:               "latest",
		Phase:                "plan",
		Participants:         []string{"repo-surface-engineer", "post-implementation-reviewer"},
		ParticipantRationale: []string{"repo-surface-engineer owns implementation", "post-implementation-reviewer checks acceptance"},
		CompletePhase:        true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:              "latest",
		Phase:               "implement",
		FilesChanged:        []string{"src/launch_summary.go", "src/launch_summary_test.go"},
		Verification:        []string{"focused static check for changed files", "focused tests"},
		VerificationResults: []string{"focused static check for changed files passed", "focused tests passed; assertion quality reviewed against acceptance criteria"},
		LastResult:          "implementation and verification finished",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := VerifyWorkflowRun(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ok" {
		t.Fatalf("verification = %#v, want auto-completed ok run", result)
	}
	var checkpoint WorkflowCheckpoint
	if err := ReadJSON(filepath.Join(root, run.CheckpointPath), &checkpoint); err != nil {
		t.Fatal(err)
	}
	if checkpoint.Status != "complete" || checkpoint.CurrentPhase != "" || checkpoint.NextPhase != "" {
		t.Fatalf("checkpoint = %#v, want terminal complete state", checkpoint)
	}
	if !sameStrings(checkpoint.CompletedPhases, []string{"plan", "implement", "review"}) {
		t.Fatalf("completedPhases = %#v, want all phases auto-completed", checkpoint.CompletedPhases)
	}
}

func TestVerifyWorkflowRunRequiresFocusedStaticEvidenceForChangedFiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "maxParticipants": 3,
  "phases": [
    {"id": "implement", "name": "Implement", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["repo-surface-engineer"], "prompt": "implement"},
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	if _, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "change source"); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:               "latest",
		Phase:                "verify",
		Participants:         []string{"repo-surface-engineer", "repo-test-quality-reviewer"},
		ParticipantRationale: []string{"implementation owner", "verification reviewer"},
		FilesChanged:         []string{"src/feature/service.go"},
		Verification:         []string{"focused tests"},
		VerificationResults:  []string{"focused tests passed"},
		CompletePhase:        true,
	}); err != nil {
		t.Fatal(err)
	}

	result, err := VerifyWorkflowRun(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status != "warning" {
		t.Fatalf("verification = %#v, want warning until focused static evidence is recorded", result)
	}
	if !workflowCheckSelected(result.Checks, "focused-static-verification") {
		t.Fatalf("checks = %#v, want focused-static-verification warning", result.Checks)
	}
}

func TestVerifyWorkflowRunRequiresAssertionQualityEvidenceForChangedTests(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "maxParticipants": 3,
  "phases": [
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	if _, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "update tests"); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:               "latest",
		Phase:                "verify",
		Participants:         []string{"repo-test-quality-reviewer"},
		ParticipantRationale: []string{"verification reviewer"},
		FilesChanged:         []string{"src/feature/service_test.go"},
		Verification:         []string{"focused static check for changed files", "focused tests"},
		VerificationResults:  []string{"focused static check for changed files passed", "focused tests passed"},
		CompletePhase:        true,
	}); err != nil {
		t.Fatal(err)
	}

	result, err := VerifyWorkflowRun(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status != "warning" {
		t.Fatalf("verification = %#v, want warning until assertion quality review is recorded", result)
	}
	if !workflowCheckSelected(result.Checks, "assertion-quality") {
		t.Fatalf("checks = %#v, want assertion-quality warning", result.Checks)
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
