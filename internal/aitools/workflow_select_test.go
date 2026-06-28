package aitools

import "testing"

func TestSelectWorkflowRanksSpecificWorkflowForTask(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)

	result, err := SelectWorkflow(root, WorkflowSelectOptions{
		Task: "Fix auth guard public route regression and add a focused test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Selected.ID != "bugfix-swarm" {
		t.Fatalf("selected workflow = %q, want bugfix-swarm; candidates=%#v", result.Selected.ID, result.Candidates)
	}
	if result.Selected.Score <= 0 {
		t.Fatalf("selected score = %d, want positive", result.Selected.Score)
	}
	if len(result.Selected.Rationale) == 0 {
		t.Fatalf("selected rationale is empty: %#v", result.Selected)
	}
	if result.WorkflowPath == "" {
		t.Fatalf("workflow path is empty: %#v", result)
	}
}

func TestSelectWorkflowFallsBackToFeatureDelivery(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)

	result, err := SelectWorkflow(root, WorkflowSelectOptions{
		Task: "Add a new export endpoint for reports",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Selected.ID != "feature-delivery-swarm" {
		t.Fatalf("selected workflow = %q, want feature-delivery-swarm; candidates=%#v", result.Selected.ID, result.Candidates)
	}
}

func TestSelectWorkflowHonorsExplicitWorkflow(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)

	result, err := SelectWorkflow(root, WorkflowSelectOptions{
		Task:     "Fix auth bug",
		Workflow: ".runweaver/workflows/test-hardening-swarm.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Selected.ID != "test-hardening-swarm" {
		t.Fatalf("selected workflow = %q, want explicit test-hardening-swarm", result.Selected.ID)
	}
	if !result.Selected.Explicit {
		t.Fatalf("selected explicit = false, want true")
	}
}

func writeWorkflowSelectionFixtures(t *testing.T, root string) {
	t.Helper()
	writeTestFile(t, root, ".runweaver/workflows/feature-delivery-swarm.json", `{
  "id": "feature-delivery-swarm",
  "name": "Feature Delivery Swarm",
  "description": "Implement new repository features and product behavior.",
  "phases": [
    {"id": "plan", "name": "Plan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "plan feature"},
    {"id": "implement", "name": "Implement", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["repo-surface-engineer"], "prompt": "implement feature"}
  ]
}`)
	writeTestFile(t, root, ".runweaver/workflows/bugfix-swarm.json", `{
  "id": "bugfix-swarm",
  "name": "Bugfix Swarm",
  "description": "Reproduce, fix, and verify defects, bugs, regressions, crashes, and failing tests.",
  "phases": [
    {"id": "reproduce", "name": "Reproduce", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "reproduce failing bug"},
    {"id": "fix", "name": "Fix", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["repo-surface-engineer"], "prompt": "fix bug"}
  ]
}`)
	writeTestFile(t, root, ".runweaver/workflows/test-hardening-swarm.json", `{
  "id": "test-hardening-swarm",
  "name": "Test Hardening Swarm",
  "description": "Add, repair, and harden tests, flakes, coverage, and verification.",
  "phases": [
    {"id": "test", "name": "Test", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["repo-test-quality-reviewer"], "prompt": "add tests"}
  ]
}`)
}
