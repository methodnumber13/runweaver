package aitools

import (
	"path/filepath"
	"strings"
	"testing"
)

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

func TestStartWorkflowResumeKeepsParticipantContractConsistent(t *testing.T) {
	root := t.TempDir()
	writeStartFixtures(t, root)
	first, err := StartWorkflow(root, StartOptions{
		Task:    "Fix public route auth guard regression",
		Runtime: RuntimeOpenCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, ".opencode/swarm/profile.json", `{
  "workspace": {"name": "api", "repos": ["."]},
  "repos": [{
    "dir": ".",
    "agents": [
      {"name": "billing-agent", "description": "Owns billing changes", "focusFiles": ["src/billing/billing.service.ts"]}
    ]
  }]
}`)

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
	if !sameStrings(second.ExecutionContract.Participants, first.ExecutionContract.Participants) {
		t.Fatalf("contract participants = %#v, want preserved %#v", second.ExecutionContract.Participants, first.ExecutionContract.Participants)
	}
	if !sameStrings(assignmentNames(second.ExecutionContract.Assignments), second.ExecutionContract.Participants) {
		t.Fatalf("contract assignments = %#v, want same actor names as participants %#v", second.ExecutionContract.Assignments, second.ExecutionContract.Participants)
	}
	if !sameStrings(second.Participants.Participants, second.ExecutionContract.Participants) {
		t.Fatalf("top-level participants = %#v, want contract participants %#v", second.Participants.Participants, second.ExecutionContract.Participants)
	}
}

func TestStartWorkflowAutoRuntimeUsesAvailableProfile(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)
	t.Setenv("PATH", t.TempDir())
	writeRuntimeShimOnPath(t, "codex")
	writeTestFile(t, root, ".codex/runweaver/profile.json", `{
  "workspace": {"name": "api", "repos": ["."]},
  "repos": [{
    "dir": ".",
    "agents": [
      {"name": "codex-docs-agent", "description": "Owns README docs and markdown edits", "focusFiles": ["README.md"]}
    ]
  }]
}`)

	result, err := StartWorkflow(root, StartOptions{
		Task:    "Rename typo in README",
		Runtime: RuntimeAuto,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Runtime != RuntimeCodex {
		t.Fatalf("runtime = %q, want codex", result.Runtime)
	}
	if !strings.Contains(result.RuntimeResolution.Source, "profile") {
		t.Fatalf("runtime resolution = %#v, want profile source", result.RuntimeResolution)
	}
	if result.ExecutionContract.TaskTier.Tier != "trivial" {
		t.Fatalf("execution contract = %#v, want trivial tier", result.ExecutionContract)
	}
	if result.Participants.Cap != 1 {
		t.Fatalf("participant cap = %d, want trivial cap 1", result.Participants.Cap)
	}
}

func TestStartWorkflowRefreshesIndexWithSelectedRuntime(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)
	writeTestFile(t, root, "go.mod", "module example.com/api\n")
	writeTestFile(t, root, "cmd/api/main.go", "package main\nfunc main() {}\n")
	writeTestFile(t, root, ".codex/runweaver/profile.json", `{
  "workspace": {"name": "api", "repos": ["."]},
  "repos": [{
    "dir": ".",
    "agents": [
      {"name": "codex-go-agent", "description": "Owns Go entrypoints", "focusFiles": ["cmd/api/main.go"]}
    ]
  }]
}`)

	result, err := StartWorkflow(root, StartOptions{
		Task:    "Fix Go API smoke bug",
		Runtime: RuntimeCodex,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IndexRefreshed {
		t.Fatalf("indexRefreshed = false, want true")
	}
	var index RepoIndex
	if err := ReadJSON(filepath.Join(root, ".runweaver/tmp/index/repo-index.json"), &index); err != nil {
		t.Fatal(err)
	}
	if index.ClassifierRun == nil || index.ClassifierRun.Runtime != RuntimeCodex {
		t.Fatalf("classifierRun = %#v, want codex runtime", index.ClassifierRun)
	}
}

func assignmentNames(assignments []ParticipantAssignment) []string {
	names := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		names = append(names, assignment.Name)
	}
	return names
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func TestStartWorkflowIncludesTaskScopedContext(t *testing.T) {
	root := t.TempDir()
	writeStartFixtures(t, root)
	writeContextIndexFixture(t, root)

	result, err := StartWorkflow(root, StartOptions{
		Task:      "Fix public route auth guard test",
		Runtime:   RuntimeOpenCode,
		SkipIndex: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !contextFileSelected(result.ExecutionContract.Context.Files, "src/auth/auth.guard.ts") {
		t.Fatalf("context = %#v, want auth guard source", result.ExecutionContract.Context)
	}
	if len(result.ExecutionContract.Context.Tests) == 0 {
		t.Fatalf("context = %#v, want related tests", result.ExecutionContract.Context)
	}
}

func TestStartWorkflowReturnsTerminalCompletionContract(t *testing.T) {
	root := t.TempDir()
	writeStartFixtures(t, root)

	result, err := StartWorkflow(root, StartOptions{
		Task:    "Implement product card accessibility and verify",
		Runtime: RuntimeOpenCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.ExecutionContract.PhaseCompletion, "--complete-phase") {
		t.Fatalf("phase completion = %q, want --complete-phase command", result.ExecutionContract.PhaseCompletion)
	}
	for _, want := range []string{"in_progress", "complete all workflow phases", "--blocker", "nextVerification"} {
		if !strings.Contains(result.ExecutionContract.TerminalRule, want) {
			t.Fatalf("terminal rule = %q, want %q", result.ExecutionContract.TerminalRule, want)
		}
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
