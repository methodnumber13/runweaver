package aitools

import (
	"path/filepath"
	"testing"
)

func TestPlanWorkflowWritesCheckpointTodoAndLatest(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {
      "id": "scan",
      "name": "Scan",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-surface-indexer"],
      "prompt": "scan"
    },
    {
      "id": "verify",
      "name": "Verify",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-test-quality-reviewer"],
      "prompt": "verify"
    }
  ]
}`)

	result, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "check tests")
	if err != nil {
		t.Fatal(err)
	}
	if result.Workflow != "test-swarm" || result.Status != "planned" {
		t.Fatalf("summary = %#v, want planned test-swarm", result)
	}
	for _, name := range []string{
		result.CheckpointPath,
		result.TodoPath,
		filepath.Join(result.RunDir, "plan.json"),
		filepath.Join(result.RunDir, "events.ndjson"),
		".runweaver/tmp/swarm-runs/latest.json",
	} {
		if !Exists(filepath.Join(root, name)) {
			t.Fatalf("expected workflow artifact %s to exist", name)
		}
	}

	status, err := WorkflowStatus(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if status["workflow"] != "test-swarm" || status["nextPhase"] != "scan" {
		t.Fatalf("status = %#v, want workflow test-swarm next scan", status)
	}
}

func TestPlanWorkflowCreatesPhaseArtifacts(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "maxParticipants": 3,
  "phases": [
    {"id": "reproduce", "name": "Reproduce", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "reproduce"},
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	result, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "map auth bug")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(result.RunDir, "agents"),
		filepath.Join(result.RunDir, "phases/reproduce/handoff.md"),
		filepath.Join(result.RunDir, "phases/reproduce/notes.md"),
		filepath.Join(result.RunDir, "phases/reproduce/verification.jsonl"),
		filepath.Join(result.RunDir, "phases/verify/handoff.md"),
		filepath.Join(result.RunDir, "phases/verify/notes.md"),
		filepath.Join(result.RunDir, "phases/verify/verification.jsonl"),
	} {
		if !Exists(filepath.Join(root, path)) {
			t.Fatalf("expected workflow artifact %s to exist", path)
		}
	}
}

func TestPlanWorkflowFallsBackToLegacyFeatureWorkflow(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".opencode/workflows/feature-swarm.json", `{
  "id": "feature-swarm",
  "name": "Feature Swarm",
  "phases": []
}`)

	result, err := PlanWorkflow(root, "", "legacy feature")
	if err != nil {
		t.Fatal(err)
	}
	if result.Workflow != "feature-swarm" {
		t.Fatalf("workflow = %q, want legacy feature-swarm fallback", result.Workflow)
	}
}

func TestPlanWorkflowRunIDsAreUniqueForRapidRepeatedPlans(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": []
}`)

	first, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "first")
	if err != nil {
		t.Fatal(err)
	}
	second, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "second")
	if err != nil {
		t.Fatal(err)
	}
	if first.RunDir == second.RunDir {
		t.Fatalf("run directories are identical: %q", first.RunDir)
	}
}
