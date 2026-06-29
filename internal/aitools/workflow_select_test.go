package aitools

import (
	"strings"
	"testing"
)

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

func TestSelectWorkflowReportsTaskTier(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)

	result, err := SelectWorkflow(root, WorkflowSelectOptions{
		Task: "Rename typo in README",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TaskTier.Tier != "trivial" {
		t.Fatalf("tier = %#v, want trivial", result.TaskTier)
	}
	if len(result.TaskTier.Rationale) == 0 {
		t.Fatalf("tier rationale is empty: %#v", result.TaskTier)
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

func TestSelectWorkflowDoesNotRouteOnStopwordsOrGenericWorkflowToken(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)
	writeTestFile(t, root, ".runweaver/workflows/metadata-refresh-swarm.json", `{
  "id": "metadata-refresh-swarm",
  "name": "Runtime Metadata Refresh Swarm",
  "description": "Refresh repository surface indexes and detect stale agent and skill metadata.",
  "phases": [
    {"id": "refresh", "name": "Refresh", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["agent-skill-drift-reviewer"], "prompt": "refresh metadata"}
  ]
}`)

	result, err := SelectWorkflow(root, WorkflowSelectOptions{
		Task: "Read-only Codex adoption smoke. Do not edit source files. Inspect RunWeaver workflow state and update the checkpoint with a smoke result only.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Selected.ID == "metadata-refresh-swarm" {
		t.Fatalf("selected workflow = %q, want generic smoke text not to route to metadata refresh; candidates=%#v", result.Selected.ID, result.Candidates)
	}
	for _, candidate := range result.Candidates {
		for _, rationale := range candidate.Rationale {
			if strings.Contains(rationale, "task token matches workflow text: and") ||
				strings.Contains(rationale, "task token matches workflow text: the") {
				t.Fatalf("candidate rationale = %#v, want stopwords filtered", candidate.Rationale)
			}
		}
	}
}

func TestTokenizeSelectionTextFiltersStopwords(t *testing.T) {
	tokens := tokenizeSelectionText("Do not edit the source files and update workflow state with a result")
	for _, token := range []string{"do", "not", "the", "and", "with", "a"} {
		if tokens[token] {
			t.Fatalf("tokens = %#v, want stopword %q filtered", tokens, token)
		}
	}
	for _, token := range []string{"edit", "source", "files", "update", "workflow", "state", "result"} {
		if !tokens[token] {
			t.Fatalf("tokens = %#v, want token %q preserved", tokens, token)
		}
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
