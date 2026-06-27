package aitools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkflowStatusMissingLatestReturnsActionableError(t *testing.T) {
	root := t.TempDir()

	_, err := WorkflowStatus(root, "latest")
	if err == nil {
		t.Fatal("WorkflowStatus returned nil error, want missing latest error")
	}
	if !strings.Contains(err.Error(), "no latest workflow run found") || !strings.Contains(err.Error(), "runweaver workflow run") {
		t.Fatalf("error = %q, want actionable missing latest message", err.Error())
	}
}

func TestWorkflowResumeRejectsPathsOutsideSwarmRuns(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeTestFile(t, outside, "checkpoint.json", `{
  "schemaVersion": 1,
  "runId": "outside",
  "workflow": "outside",
  "task": "outside",
  "status": "planned",
  "completedPhases": [],
  "nextPhase": "scan",
  "updatedAt": "2026-06-12T00:00:00Z"
}`)

	for _, resume := range []string{outside, "../outside", ".runweaver/tmp/../outside"} {
		if _, err := WorkflowStatus(root, resume); err == nil || !strings.Contains(err.Error(), "workflow resume path must stay under .runweaver/tmp/swarm-runs") {
			t.Fatalf("WorkflowStatus(%q) err = %v, want confined resume error", resume, err)
		}
		if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{Resume: resume, Phase: "scan"}); err == nil || !strings.Contains(err.Error(), "workflow resume path must stay under .runweaver/tmp/swarm-runs") {
			t.Fatalf("UpdateWorkflow(%q) err = %v, want confined resume error", resume, err)
		}
	}
}

func TestWorkflowStatusRejectsLegacyTmpLatest(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, ".opencode/tmp/swarm-runs/legacy-run")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	checkpoint := WorkflowCheckpoint{
		SchemaVersion:   1,
		RunID:           "legacy-run",
		Workflow:        "legacy-workflow",
		Task:            "legacy task",
		Status:          "planned",
		CompletedPhases: []string{},
		NextPhase:       "scan",
		UpdatedAt:       "2026-06-13T00:00:00Z",
	}
	if err := WriteJSON(filepath.Join(runDir, "checkpoint.json"), checkpoint); err != nil {
		t.Fatal(err)
	}
	if err := WriteJSON(filepath.Join(root, ".opencode/tmp/swarm-runs/latest.json"), WorkflowLatest{
		RunID:     "legacy-run",
		RunDir:    ".opencode/tmp/swarm-runs/legacy-run",
		Workflow:  "legacy-workflow",
		Task:      "legacy task",
		UpdatedAt: "2026-06-13T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := WorkflowStatus(root, "latest"); err == nil || !strings.Contains(err.Error(), "no latest workflow run found") {
		t.Fatalf("WorkflowStatus(latest) err = %v, want no canonical latest workflow run found", err)
	}

	if _, err := WorkflowStatus(root, ".opencode/tmp/swarm-runs/legacy-run"); err == nil || !strings.Contains(err.Error(), "workflow resume path must stay under .runweaver/tmp/swarm-runs") {
		t.Fatalf("WorkflowStatus(legacy path) err = %v, want confined canonical resume error", err)
	}
}
