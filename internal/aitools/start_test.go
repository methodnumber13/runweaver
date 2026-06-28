package aitools

import "testing"

func TestStartWorkflowCreatesWorkflowAndRecordsParticipants(t *testing.T) {
	root := t.TempDir()
	writeStartFixtures(t, root)

	result, err := StartWorkflow(root, StartOptions{
		Task:    "Fix public route auth guard regression in src/auth/auth.guard.ts",
		Runtime: RuntimeOpenCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "created" {
		t.Fatalf("action = %q, want created", result.Action)
	}
	if result.Workflow.Workflow != "bugfix-swarm" {
		t.Fatalf("workflow = %#v, want bugfix-swarm", result.Workflow)
	}
	if !containsString(result.Participants.Participants, "auth-agent") {
		t.Fatalf("participants = %#v, want auth-agent", result.Participants.Participants)
	}
	status, err := WorkflowStatus(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	participants, _ := status["participants"].([]any)
	if len(participants) == 0 {
		t.Fatalf("status = %#v, want participants persisted", status)
	}
	if status["lastResult"] == "" || status["nextAction"] == "" {
		t.Fatalf("status = %#v, want lastResult and nextAction", status)
	}
}

func TestStartWorkflowResumesMatchingActiveWorkflow(t *testing.T) {
	root := t.TempDir()
	writeStartFixtures(t, root)
	first, err := StartWorkflow(root, StartOptions{
		Task:    "Fix public route auth guard regression",
		Runtime: RuntimeOpenCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := StartWorkflow(root, StartOptions{
		Task:    "continue fixing public route auth guard regression",
		Runtime: RuntimeOpenCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Action != "resumed" {
		t.Fatalf("action = %q, want resumed", second.Action)
	}
	if second.Workflow.RunDir != first.Workflow.RunDir {
		t.Fatalf("run dir = %q, want existing %q", second.Workflow.RunDir, first.Workflow.RunDir)
	}
}

func writeStartFixtures(t *testing.T, root string) {
	t.Helper()
	writeTestFile(t, root, "go.mod", "module example.com/api\n")
	writeTestFile(t, root, "src/auth/auth.guard.ts", "export class AuthGuard {}\n")
	writeWorkflowSelectionFixtures(t, root)
	writeTestFile(t, root, ".opencode/swarm/profile.json", `{
  "workspace": {"name": "api", "repos": ["."]},
  "repos": [{
    "dir": ".",
    "agents": [
      {"name": "auth-agent", "description": "Owns auth guards and public route checks", "focusFiles": ["src/auth/auth.guard.ts"]}
    ],
    "customSkills": [
      {"name": "security-middleware", "description": "Use for guard and decorator security checks", "focusFiles": ["src/auth/auth.guard.ts"]}
    ]
  }]
}`)
}
