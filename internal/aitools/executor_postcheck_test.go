package aitools

import (
	"context"
	"os"
	"testing"
)

func TestExecuteWorkflowWarnsWhenCheckpointDoesNotAdvance(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {
      "id": "plan",
      "name": "Plan",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-surface-indexer"],
      "prompt": "plan"
    }
  ]
}`)
	runner := func(ctx context.Context, dir string, name string, args []string, stdoutPath string, stderrPath string, env []string) (int, error) {
		if err := os.WriteFile(stdoutPath, []byte(`{"type":"done"}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(stderrPath, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
		return 0, nil
	}

	result, err := executeWorkflow(root, WorkflowExecuteOptions{
		WorkflowPath:     ".runweaver/workflows/test-swarm.json",
		Task:             "execute",
		SkipModelCheck:   true,
		SkipRuntimeCheck: true,
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "warning" || result.PostCheck == nil || result.PostCheck.Status != "warning" {
		t.Fatalf("result = %#v, want post-check warning", result)
	}
}

func TestExecuteWorkflowResumeLoadsPlanPhasesForPostCheck(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {
      "id": "plan",
      "name": "Plan",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-surface-indexer"],
      "prompt": "plan"
    }
  ]
}`)
	if _, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "resume task"); err != nil {
		t.Fatal(err)
	}
	runner := func(ctx context.Context, dir string, name string, args []string, stdoutPath string, stderrPath string, env []string) (int, error) {
		if err := os.WriteFile(stdoutPath, []byte(`{"type":"done"}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(stderrPath, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
		return 0, nil
	}

	result, err := executeWorkflow(root, WorkflowExecuteOptions{
		Resume:           "latest",
		SkipModelCheck:   true,
		SkipRuntimeCheck: true,
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if result.PostCheck == nil || result.PostCheck.Status != "warning" {
		t.Fatalf("postCheck = %#v, want warning from resumed plan phases", result.PostCheck)
	}
}
