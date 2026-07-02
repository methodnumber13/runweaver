package aitools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateWorkflowPersistsParticipantsFindingsAndEvents(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {"id": "reproduce", "name": "Reproduce", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "reproduce"},
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	if _, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "check auth"); err != nil {
		t.Fatal(err)
	}

	status, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:       "latest",
		Phase:        "reproduce",
		Status:       "in_progress",
		Participants: []string{"auth-domain-agent", "repo-test-quality-reviewer"},
		ParticipantRationale: []string{
			"auth-domain-agent owns src/auth focus files",
			"repo-test-quality-reviewer checks focused regression coverage",
		},
		Findings:            []string{"auth.guard.ts checks IS_PUBLIC_KEY before token extraction"},
		Decisions:           []string{"add regression around public route bypass"},
		FilesRead:           []string{"src/auth/auth.guard.ts", "test/unit/auth.guard.spec.ts"},
		FilesChanged:        []string{"test/unit/auth.guard.spec.ts"},
		Artifacts:           []string{".runweaver/tmp/swarm-runs/latest/phases/reproduce/handoff.md"},
		LastResult:          "reproduce failed before public-route bypass was covered",
		RejectedPaths:       []string{"changing guard behavior rejected: source already returns true for public routes"},
		NextAction:          "add public route test",
		NextVerification:    "run focused auth guard unit test after adding regression",
		Verification:        []string{"npm run test -- auth.guard.spec.ts"},
		VerificationResults: []string{"not run yet; reproduce phase only"},
		Blockers:            []string{"none"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if status["status"] != "in_progress" || status["currentPhase"] != "reproduce" {
		t.Fatalf("status = %#v, want reproduce in_progress", status)
	}
	checkpointPath := filepath.Join(root, ".runweaver/tmp/swarm-runs/latest.json")
	if !Exists(checkpointPath) {
		t.Fatalf("latest checkpoint pointer missing: %s", checkpointPath)
	}
	status, err = WorkflowStatus(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	participants, _ := status["participants"].([]any)
	if len(participants) != 2 {
		t.Fatalf("status = %#v, want persisted participants", status)
	}
	if status["nextAction"] != "add public route test" {
		t.Fatalf("status = %#v, want next action", status)
	}
	if status["lastResult"] != "reproduce failed before public-route bypass was covered" {
		t.Fatalf("status = %#v, want last result", status)
	}
	if status["nextVerification"] != "run focused auth guard unit test after adding regression" {
		t.Fatalf("status = %#v, want next verification", status)
	}
	if status["indexFreshnessStatus"] != "missing" || status["staleIndex"] != true {
		t.Fatalf("status = %#v, want missing stale index signal", status)
	}
	for key, wantLen := range map[string]int{
		"participantRationale": 2,
		"decisions":            1,
		"filesRead":            2,
		"filesChanged":         1,
		"artifacts":            1,
		"rejectedPaths":        1,
		"verificationResults":  1,
		"blockers":             1,
	} {
		values, _ := status[key].([]any)
		if len(values) != wantLen {
			t.Fatalf("status[%s] = %#v, want %d item(s); full status=%#v", key, status[key], wantLen, status)
		}
	}

	latest := WorkflowLatest{}
	if err := ReadJSON(filepath.Join(root, ".runweaver/tmp/swarm-runs/latest.json"), &latest); err != nil {
		t.Fatal(err)
	}
	events, err := os.ReadFile(filepath.Join(root, latest.RunDir, "events.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"type":"updated"`, "auth-domain-agent", "filesChanged", "lastResult", "rejectedPaths", "nextVerification", "indexFreshnessStatus", "verificationResults", "participantRationale"} {
		if !strings.Contains(string(events), want) {
			t.Fatalf("events = %s, want update event detail %q", string(events), want)
		}
	}
	current, err := os.ReadFile(filepath.Join(root, latest.RunDir, "current.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Current phase: reproduce",
		"auth-domain-agent",
		"reproduce failed before public-route bypass was covered",
		"changing guard behavior rejected",
		"add public route test",
		"run focused auth guard unit test",
		"not run yet; reproduce phase only",
		"Index Freshness At Last Update",
		"Stale index: true",
	} {
		if !strings.Contains(string(current), want) {
			t.Fatalf("current.md missing %q:\n%s", want, string(current))
		}
	}
}

func TestUpdateWorkflowCapsParticipantRationaleAtPersistenceBoundary(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {"id": "plan", "name": "Plan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "plan"}
  ]
}`)
	if _, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "check rationale caps"); err != nil {
		t.Fatal(err)
	}
	oversized := "repo-surface-engineer: " + strings.Repeat("long evidence with focus files and verification commands ", 20)
	status, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:               "latest",
		Phase:                "plan",
		Participants:         []string{"repo-surface-engineer"},
		ParticipantRationale: []string{oversized},
	})
	if err != nil {
		t.Fatal(err)
	}
	rationale, _ := status["participantRationale"].([]any)
	if len(rationale) != 1 || len(rationale[0].(string)) > 240 {
		t.Fatalf("status rationale = %#v, want capped rationale", rationale)
	}
	latest := WorkflowLatest{}
	if err := ReadJSON(filepath.Join(root, ".runweaver/tmp/swarm-runs/latest.json"), &latest); err != nil {
		t.Fatal(err)
	}
	events, err := os.ReadFile(filepath.Join(root, latest.RunDir, "events.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(events)), "\n")
	var event WorkflowEvent
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &event); err != nil {
		t.Fatal(err)
	}
	if len(event.ParticipantRationale) != 1 || len(event.ParticipantRationale[0]) > 240 {
		t.Fatalf("event rationale = %#v, want capped rationale", event.ParticipantRationale)
	}
}

func TestUpdateWorkflowCompletePhaseSynchronizesTodo(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {"id": "reproduce", "name": "Reproduce", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "reproduce"},
    {"id": "fix", "name": "Fix", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["repo-surface-engineer"], "prompt": "fix"},
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	if _, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "check auth"); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{Resume: "latest", Phase: "reproduce", CompletePhase: true}); err != nil {
		t.Fatal(err)
	}
	latest := WorkflowLatest{}
	if err := ReadJSON(filepath.Join(root, ".runweaver/tmp/swarm-runs/latest.json"), &latest); err != nil {
		t.Fatal(err)
	}
	todo, err := os.ReadFile(filepath.Join(root, latest.RunDir, "todo.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(todo)
	for _, want := range []string{
		"- [x] reproduce - Reproduce",
		"- [>] fix - Fix",
		"- [ ] verify - Verify",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("todo after reproduce complete missing %q:\n%s", want, text)
		}
	}
	events, err := os.ReadFile(filepath.Join(root, latest.RunDir, "events.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(events), `"phase":"reproduce"`) {
		t.Fatalf("events after reproduce complete = %s, want event for completed phase", string(events))
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{Resume: "latest", Phase: "fix", CompletePhase: true}); err != nil {
		t.Fatal(err)
	}
	status, err := UpdateWorkflow(root, WorkflowUpdateOptions{Resume: "latest", Phase: "verify", CompletePhase: true})
	if err != nil {
		t.Fatal(err)
	}
	if status["status"] != "complete" {
		t.Fatalf("status = %#v, want complete", status)
	}
	todo, err = os.ReadFile(filepath.Join(root, latest.RunDir, "todo.md"))
	if err != nil {
		t.Fatal(err)
	}
	text = string(todo)
	for _, want := range []string{
		"- [x] reproduce - Reproduce",
		"- [x] fix - Fix",
		"- [x] verify - Verify",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("todo after workflow complete missing %q:\n%s", want, text)
		}
	}
}

func TestUpdateWorkflowCompletePhaseBackfillsEarlierPlanPhases(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/feature-delivery-swarm.json", `{
  "id": "feature-delivery-swarm",
  "name": "Feature Delivery",
  "phases": [
    {"id": "plan", "name": "Plan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "plan"},
    {"id": "implement", "name": "Implement", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["repo-surface-engineer"], "prompt": "implement"},
    {"id": "review", "name": "Review", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "review"}
  ]
}`)
	if _, err := PlanWorkflow(root, ".runweaver/workflows/feature-delivery-swarm.json", "implement feature"); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:        "latest",
		Phase:         "implement",
		CompletePhase: true,
		Participants:  []string{"repo-surface-engineer", "repo-test-quality-reviewer"},
		ParticipantRationale: []string{
			"repo-surface-engineer: implementation owner",
			"repo-test-quality-reviewer: verification reviewer",
		},
		FilesChanged: []string{"src/feature/example.go"},
		LastResult:   "implementation complete",
	}); err != nil {
		t.Fatal(err)
	}
	status, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:              "latest",
		Phase:               "review",
		CompletePhase:       true,
		Verification:        []string{"focused static check for changed files", "focused tests passed"},
		VerificationResults: []string{"focused static check for changed files passed", "focused tests passed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if status["status"] != "complete" {
		t.Fatalf("status = %#v, want complete", status)
	}
	completed, _ := status["completedPhases"].([]any)
	if !sameAnyStrings(completed, []string{"plan", "implement", "review"}) {
		t.Fatalf("completedPhases = %#v, want plan/implement/review backfilled", completed)
	}
	result, err := VerifyWorkflowRun(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ok" {
		t.Fatalf("verification = %#v, want complete phase state to verify", result)
	}
}

func TestUpdateWorkflowCompletePhaseNormalizesNextAction(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	if _, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "check auth"); err != nil {
		t.Fatal(err)
	}
	status, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:           "latest",
		Phase:            "verify",
		NextAction:       "rerun workflow verify",
		NextVerification: "run runweaver workflow verify --repo . --resume latest",
		CompletePhase:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if status["status"] != "complete" {
		t.Fatalf("status = %#v, want complete", status)
	}
	if status["nextAction"] != completeWorkflowNextAction {
		t.Fatalf("status nextAction = %#v, want normalized complete action", status["nextAction"])
	}
	if _, ok := status["nextVerification"]; ok {
		t.Fatalf("status = %#v, want nextVerification cleared", status)
	}
}

func TestUpdateWorkflowCompletePhaseDoesNotPromoteCommandEvidence(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	if _, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "check evidence"); err != nil {
		t.Fatal(err)
	}
	status, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:        "latest",
		Phase:         "verify",
		Verification:  []string{"focused checks passed; static check passed"},
		CompletePhase: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	results, _ := status["verificationResults"].([]any)
	if len(results) != 0 {
		t.Fatalf("status = %#v, want command evidence not promoted into structured verification results", status)
	}
	result, err := VerifyWorkflowRun(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status != "warning" {
		t.Fatalf("verification = %#v, want warning until verificationResults or phase evidence is recorded", result)
	}
}

func sameAnyStrings(values []any, want []string) bool {
	if len(values) != len(want) {
		return false
	}
	for i := range values {
		if values[i] != want[i] {
			return false
		}
	}
	return true
}

func TestUpdateWorkflowCapturesVerificationWarningsAsResidualRisks(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	if _, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "check warning capture"); err != nil {
		t.Fatal(err)
	}
	status, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:              "latest",
		Phase:               "verify",
		Verification:        []string{"tool test"},
		VerificationResults: []string{"PASS\nWarning: async state update escaped the verification boundary"},
	})
	if err != nil {
		t.Fatal(err)
	}
	risks, _ := status["residualRisks"].([]any)
	if len(risks) != 1 {
		t.Fatalf("status = %#v, want auto residual risk from warning output", status)
	}
	if !strings.Contains(risks[0].(string), "Warning: async state update") {
		t.Fatalf("residual risks = %#v, want warning evidence", risks)
	}
}

func TestUpdateWorkflowCanReplaceParticipants(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "maxParticipants": 2,
  "phases": [
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	if _, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "verify participants"); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:       "latest",
		Phase:        "verify",
		Participants: []string{"repo-test-quality-reviewer", "repo-quality-gates", "stale-fallback"},
	}); err != nil {
		t.Fatal(err)
	}
	status, err := UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:              "latest",
		Phase:               "verify",
		Participants:        []string{"repo-test-quality-reviewer", "repo-quality-gates"},
		ReplaceParticipants: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	participants, _ := status["participants"].([]any)
	if got := len(participants); got != 2 {
		t.Fatalf("participants = %#v, want 2 after replace", participants)
	}
	for _, unexpected := range participants {
		if unexpected == "stale-fallback" {
			t.Fatalf("participants = %#v, stale fallback should have been replaced", participants)
		}
	}
}

func TestUpdateWorkflowCompletePhaseRequiresReadablePlan(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {"id": "scan", "name": "Scan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "scan"}
  ]
}`)
	latest, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "map repo")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, latest.RunDir, "plan.json")); err != nil {
		t.Fatal(err)
	}

	_, err = UpdateWorkflow(root, WorkflowUpdateOptions{Resume: "latest", Phase: "scan", CompletePhase: true})
	if err == nil || !strings.Contains(err.Error(), "load workflow plan") {
		t.Fatalf("err = %v, want missing plan error", err)
	}

	var checkpoint WorkflowCheckpoint
	if err := ReadJSON(filepath.Join(root, latest.CheckpointPath), &checkpoint); err != nil {
		t.Fatal(err)
	}
	if checkpoint.Status == "complete" || len(checkpoint.CompletedPhases) != 0 {
		t.Fatalf("checkpoint = %#v, want unchanged planned checkpoint", checkpoint)
	}
}

func TestUpdateWorkflowCompletePhaseRejectsUnknownPhase(t *testing.T) {
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

	_, err := UpdateWorkflow(root, WorkflowUpdateOptions{Resume: "latest", Phase: "missing", CompletePhase: true})
	if err == nil || !strings.Contains(err.Error(), "workflow phase missing was not found") {
		t.Fatalf("err = %v, want unknown phase error", err)
	}
}
