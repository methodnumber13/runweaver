package aitools

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunWeaverStatusReportsIndexFreshness(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/tool\n")
	writeTestFile(t, root, "cmd/tool/main.go", "package main\nfunc main() {}\n")

	if _, err := IndexWithOptions(root, IndexOptions{ChangedOnly: true, Prune: true, Classification: ClassifyOptions{Mode: ClassificationDeterministic}}); err != nil {
		t.Fatal(err)
	}
	status, err := RunWeaverStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if status.IndexFreshness.Status != "ok" || !status.IndexFreshness.Fresh {
		t.Fatalf("freshness = %#v, want fresh ok", status.IndexFreshness)
	}
	if status.IndexFreshness.CheckedFiles == 0 {
		t.Fatalf("freshness checked files = 0, want repository files checked")
	}
}

func TestRunWeaverStatusReportsStaleIndex(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/tool\n")
	writeTestFile(t, root, "cmd/tool/main.go", "package main\nfunc main() {}\n")

	if _, err := IndexWithOptions(root, IndexOptions{ChangedOnly: true, Prune: true, Classification: ClassifyOptions{Mode: ClassificationDeterministic}}); err != nil {
		t.Fatal(err)
	}
	newerPath := filepath.Join(root, "cmd/tool/main.go")
	future := time.Now().Add(5 * time.Second)
	if err := os.Chtimes(newerPath, future, future); err != nil {
		t.Fatal(err)
	}

	status, err := RunWeaverStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if status.IndexFreshness.Status != "stale" || status.IndexFreshness.Fresh {
		t.Fatalf("freshness = %#v, want stale", status.IndexFreshness)
	}
	if !containsString(status.IndexFreshness.StaleFiles, "cmd/tool/main.go") {
		t.Fatalf("stale files = %#v, want changed source file", status.IndexFreshness.StaleFiles)
	}
	if !containsString(status.Recommendations, "refresh stale repository context with runweaver index --repo . --changed-only --prune") {
		t.Fatalf("recommendations = %#v, want index refresh recommendation", status.Recommendations)
	}
}

func TestRunWeaverStatusTreatsStaleIndexAfterCompleteWorkflowAsInfo(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/tool\n")
	writeTestFile(t, root, "cmd/tool/main.go", "package main\nfunc main() {}\n")
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)

	if _, err := IndexWithOptions(root, IndexOptions{ChangedOnly: true, Prune: true, Classification: ClassifyOptions{Mode: ClassificationDeterministic}}); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "finish implementation"); err != nil {
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
	newerPath := filepath.Join(root, "cmd/tool/main.go")
	future := time.Now().Add(5 * time.Second)
	if err := os.Chtimes(newerPath, future, future); err != nil {
		t.Fatal(err)
	}

	status, err := RunWeaverStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "ok" || !status.Ready {
		t.Fatalf("status = %#v, want ok ready despite post-complete stale index", status)
	}
	if status.IndexFreshness.Status != "stale" || status.IndexFreshness.Fresh {
		t.Fatalf("freshness = %#v, want stale evidence retained", status.IndexFreshness)
	}
	if !containsString(status.Recommendations, "index changed after a complete workflow; the next runweaver start will refresh context automatically") {
		t.Fatalf("recommendations = %#v, want informational stale-index recommendation", status.Recommendations)
	}
}

func TestRunWeaverStatusReportsMissingIndex(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/tool\n")

	status, err := RunWeaverStatus(root)
	if err != nil {
		t.Fatal(err)
	}
	if status.IndexFreshness.Status != "missing" || status.IndexFreshness.Fresh {
		t.Fatalf("freshness = %#v, want missing", status.IndexFreshness)
	}
	if !containsString(status.Recommendations, "build repository context with runweaver index --repo . --changed-only --prune") {
		t.Fatalf("recommendations = %#v, want index build recommendation", status.Recommendations)
	}
}
