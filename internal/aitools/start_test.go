package aitools

import (
	"os"
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

func TestStartWorkflowDoesNotResumeIncompatibleActiveWorkflowForFeatureTask(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)
	writeMetadataRefreshWorkflowFixture(t, root)
	writeParticipantProfileForRuntime(t, root, RuntimeCodex)

	task := `Implement a frontend-only launch preflight UX improvement.
Add a reusable checklist with selected count, disabled reasons, pending template/config loads, and the last refresh moment.
Make stale async loads safe and do not modify backend/API/database/runtime worker files.
Add focused tests and run the smallest lint/test/build verification.`

	if _, err := PlanWorkflow(root, ".runweaver/workflows/metadata-refresh-swarm.json", task); err != nil {
		t.Fatal(err)
	}

	result, err := StartWorkflow(root, StartOptions{
		Task:      task,
		Runtime:   RuntimeCodex,
		SkipIndex: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "created" {
		t.Fatalf("action = %q, want created instead of resuming incompatible workflow", result.Action)
	}
	if result.Workflow.Workflow != "feature-delivery-swarm" {
		t.Fatalf("workflow = %#v, want feature-delivery-swarm", result.Workflow)
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
	if !stringsContain(second.Participants.Rationale, "auth-agent") {
		t.Fatalf("resume rationale = %#v, want preserved checkpoint rationale", second.Participants.Rationale)
	}
	assignmentText := strings.Join(assignmentRationale(second.ExecutionContract.Assignments), "\n")
	if !strings.Contains(assignmentText, "auth-agent") {
		t.Fatalf("assignment rationale = %q, want participant-specific checkpoint rationale", assignmentText)
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

func TestStartWorkflowCompactsParticipantDiagnosticsForRuntimeContract(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)
	writeParticipantProfileForRuntime(t, root, RuntimeOpenCode)
	writeFrontendWorkflowContextIndex(t, root)

	result, err := StartWorkflow(root, StartOptions{
		Task:      "Implement frontend FeatureLauncherCard launch readiness UI. Do not modify API/server/runtime worker files.",
		Runtime:   RuntimeOpenCode,
		SkipIndex: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Participants.Candidates) != 0 {
		t.Fatalf("participants candidates = %#v, want start contract to omit diagnostic candidate dump", result.Participants.Candidates)
	}
	for _, rationale := range append(result.Participants.Rationale, assignmentRationale(result.Participants.Assignments)...) {
		if len(rationale) > 240 {
			t.Fatalf("rationale = %q (%d chars), want compact runtime rationale", rationale, len(rationale))
		}
	}
}

func TestStartWorkflowIncludesPostImplementationReviewContract(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)
	writeRefactorWorkflowFixture(t, root)
	writeParticipantProfileForRuntime(t, root, RuntimeOpenCode)
	writeFrontendWorkflowContextIndex(t, root)

	result, err := StartWorkflow(root, StartOptions{
		Task:      "Implement frontend FeatureLauncherCard and FeatureLaunchConfigDialog variants without backend changes",
		Runtime:   RuntimeOpenCode,
		SkipIndex: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ExecutionContract.PostReview) == 0 {
		t.Fatalf("execution contract = %#v, want post-review checklist", result.ExecutionContract)
	}
	for _, want := range []string{"stale async", "frontend-only", "focused tests", "cohesive UI component"} {
		if !stringsContain(result.ExecutionContract.PostReview, want) {
			t.Fatalf("post review = %#v, want checklist item containing %q", result.ExecutionContract.PostReview, want)
		}
	}
}

func TestStartWorkflowPersistsAcceptanceCriteriaForReview(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)
	writeParticipantProfileForRuntime(t, root, RuntimeOpenCode)
	writeFrontendWorkflowContextIndex(t, root)

	task := `Implement a reusable launch readiness summary.

Requirements:
- Show selected count and delay ranges.
- Existing validation errors must be blocking readiness items.
- Do not modify server/runtime worker files.
- Run focused checks for changed files.`

	result, err := StartWorkflow(root, StartOptions{
		Task:      task,
		Runtime:   RuntimeOpenCode,
		SkipIndex: true,
		ForceNew:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ExecutionContract.AcceptanceCriteria) != 4 {
		t.Fatalf("acceptance criteria = %#v, want four parsed requirements", result.ExecutionContract.AcceptanceCriteria)
	}
	contractText := strings.Join(append(result.ExecutionContract.QualityGates, result.ExecutionContract.PostReview...), "\n")
	for _, want := range []string{"every acceptance criterion", "unmet criterion"} {
		if !strings.Contains(contractText, want) {
			t.Fatalf("contract gates/postReview = %#v/%#v, want %q", result.ExecutionContract.QualityGates, result.ExecutionContract.PostReview, want)
		}
	}
	status, err := WorkflowStatus(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	statusCriteria, _ := status["acceptanceCriteria"].([]any)
	if len(statusCriteria) != 4 {
		t.Fatalf("status = %#v, want persisted acceptance criteria", status)
	}
	current, err := os.ReadFile(filepath.Join(root, ".runweaver/tmp/current.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(current), "## Acceptance Criteria") || !strings.Contains(string(current), "validation errors") {
		t.Fatalf("current.md = %s, want acceptance criteria mirror", string(current))
	}
}

func TestStartWorkflowResumeUsesPersistedAcceptanceCriteria(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)
	writeParticipantProfileForRuntime(t, root, RuntimeCodex)
	writeFrontendWorkflowContextIndex(t, root)

	task := `Implement a launch readiness summary.

Requirements:
- Show pending work before launch.
- Existing validation errors must block readiness.`

	if _, err := StartWorkflow(root, StartOptions{
		Task:      task,
		Runtime:   RuntimeCodex,
		SkipIndex: true,
		ForceNew:  true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:             "latest",
		AcceptanceCriteria: []string{"Manual checkpoint criterion must survive short resume prompts."},
	}); err != nil {
		t.Fatal(err)
	}

	resumed, err := StartWorkflow(root, StartOptions{
		Task:      "continue",
		Runtime:   RuntimeCodex,
		SkipIndex: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Action != "resumed" {
		t.Fatalf("action = %q, want resumed", resumed.Action)
	}
	if !stringsContain(resumed.ExecutionContract.AcceptanceCriteria, "Manual checkpoint criterion") {
		t.Fatalf("acceptance criteria = %#v, want persisted checkpoint criterion after resume", resumed.ExecutionContract.AcceptanceCriteria)
	}
	if len(resumed.ExecutionContract.AcceptanceCriteria) < 3 {
		t.Fatalf("acceptance criteria = %#v, want original plus manual criteria", resumed.ExecutionContract.AcceptanceCriteria)
	}
}

func TestStartWorkflowAddsQualityGatesForExtractedFrontendHelpersAndPageWiring(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)
	writeRefactorWorkflowFixture(t, root)
	writeParticipantProfileForRuntime(t, root, RuntimeCodex)
	writeTaskLaunchesContextIndex(t, root)

	task := `Implement a frontend-only Task Launches launch safety panel.
Work only in frontend Task Launches / launch settings surfaces.
Extract a cohesive small component or pure helper if the summary is more than a few lines.
Pass launch safety data through page/container wiring.
Add focused test coverage.`

	result, err := StartWorkflow(root, StartOptions{
		Task:      task,
		Runtime:   RuntimeCodex,
		SkipIndex: true,
		ForceNew:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	contractText := strings.Join(append(result.ExecutionContract.QualityGates, result.ExecutionContract.PostReview...), "\n")
	for _, want := range []string{
		"separate focused test for the extracted component/helper",
		"focused lint/static check for changed component/helper files",
		"split pure helpers into separate files",
		"page-level verification",
		"affected pages/containers named by context query",
	} {
		if !strings.Contains(contractText, want) {
			t.Fatalf("execution contract quality gates/postReview = %#v/%#v, want %q", result.ExecutionContract.QualityGates, result.ExecutionContract.PostReview, want)
		}
	}
}

func TestStartWorkflowUsesFeatureDeliveryForFullFeaturePromptWithArchitectureConstraints(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)
	writeRefactorWorkflowFixture(t, root)
	writeParticipantProfileForRuntime(t, root, RuntimeCodex)
	writeTaskLaunchesContextIndex(t, root)

	task := `You are working in a temporary copy of sample-workspace. Do not commit, do not push, and do not read secrets or .env files.

Task: implement a frontend-only Batch Actions launch readiness improvement.

Requirements:
1. Work only in the frontend Batch Actions surface unless tests prove a shared frontend helper is needed. Do not modify backend/API/database/runtime worker files.
2. Add a compact launch readiness summary near the Batch Actions launch controls and in the launch settings dialog so an operator can understand the next launch before clicking.
3. Keep the architecture clean: if the summary UI needs more than a few lines, extract a cohesive small component/helper instead of inlining a large block into TaskLaunches.tsx or TaskLaunchSettingsDialog.tsx.
4. Make stale async/template/config loads safe.
5. Add focused test coverage.`

	result, err := StartWorkflow(root, StartOptions{
		Task:      task,
		Runtime:   RuntimeCodex,
		SkipIndex: true,
		ForceNew:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Workflow.Workflow != "feature-delivery-swarm" {
		t.Fatalf("workflow = %#v, want feature-delivery-swarm; selection=%#v", result.Workflow, result.WorkflowSelection)
	}
	if result.WorkflowSelection.Selected.ID != result.Workflow.Workflow {
		t.Fatalf("selection = %#v workflow = %#v, want same selected workflow", result.WorkflowSelection, result.Workflow)
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
