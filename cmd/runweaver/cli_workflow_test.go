package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLIStatusWorksBeforeWorkflowExists(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/tool\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI([]string{"status", "--repo", root}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("status exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"ready": false`) ||
		!strings.Contains(stdout.String(), `"latestWorkflow": false`) ||
		!strings.Contains(stdout.String(), `runweaver workflow run`) {
		t.Fatalf("status stdout = %q, want actionable no-workflow state", stdout.String())
	}
}

func TestCLIStatusPrintsIndexFreshness(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/tool\n")
	writeCLIFile(t, root, "cmd/tool/main.go", "package main\nfunc main() {}\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI([]string{"index", "--repo", root, "--classification", "deterministic"}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("index exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	future := time.Now().Add(5 * time.Second)
	if err := os.Chtimes(filepath.Join(root, "cmd/tool/main.go"), future, future); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"status", "--repo", root}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("status exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"indexFreshness"`) ||
		!strings.Contains(stdout.String(), `"status": "stale"`) ||
		!strings.Contains(stdout.String(), "cmd/tool/main.go") {
		t.Fatalf("status stdout = %q, want stale index freshness", stdout.String())
	}
}

func TestCLIScanAndWorkflowCommands(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/tool\n")
	writeCLIFile(t, root, "cmd/tool/main.go", "package main\nfunc main() {}\n")
	writeCLIFile(t, root, ".runweaver/workflows/test-swarm.json", `{
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
    }
  ]
}`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	outPath := filepath.Join(root, ".runweaver/tmp/surface-index.json")

	code := runCLI([]string{"scan", "--repo", root, "--out", outPath}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("scan exit code = %d stderr=%q", code, stderr.String())
	}
	if !fileExists(outPath) || !strings.Contains(stderr.String(), "surface index written") {
		t.Fatalf("scan did not write expected output; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workflow", "run", "--repo", root, "--workflow", ".runweaver/workflows/test-swarm.json", "--task", "map repo"}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("workflow run exit code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"workflow": "test-swarm"`) {
		t.Fatalf("workflow stdout = %q, want test-swarm", stdout.String())
	}
	if !fileExists(filepath.Join(root, ".runweaver/tmp/swarm-runs/latest.json")) {
		t.Fatal("workflow run did not write latest pointer")
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"status", "--repo", root}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("status exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"workflow": "test-swarm"`) ||
		!strings.Contains(stdout.String(), `"currentPath"`) ||
		!strings.Contains(stdout.String(), `"ready": true`) {
		t.Fatalf("status stdout = %q, want active workflow summary", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workflow", "run", "--repo", root, "--resume", "latest", "--status"}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("workflow status exit code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"nextPhase": "scan"`) {
		t.Fatalf("status stdout = %q, want next phase scan", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{
		"workflow", "update",
		"--repo", root,
		"--resume", "latest",
		"--phase", "scan",
		"--status", "in_progress",
		"--participants", "repo-surface-indexer,identity-access-agent",
		"--participant-rationale", "identity-access-agent owns auth files",
		"--finding", "selected auth domain",
		"--decision", "inspect auth guard first",
		"--file-read", "src/auth/auth.guard.ts",
		"--file-changed", "test/unit/auth.guard.spec.ts",
		"--artifact", ".runweaver/tmp/swarm-runs/latest/phases/scan/handoff.md",
		"--next-action", "read auth guard",
		"--verification", "npm run test -- auth.guard.spec.ts",
		"--verification-result", "not run yet",
		"--blocker", "none",
	}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("workflow update exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"currentPhase": "scan"`) ||
		!strings.Contains(stdout.String(), `"identity-access-agent"`) ||
		!strings.Contains(stdout.String(), `"participantRationale"`) ||
		!strings.Contains(stdout.String(), `"filesRead"`) ||
		!strings.Contains(stdout.String(), `"verificationResults"`) ||
		!strings.Contains(stdout.String(), `"read auth guard"`) {
		t.Fatalf("workflow update stdout = %q, want persisted checkpoint details", stdout.String())
	}
	currentPath := filepath.Join(root, ".runweaver/tmp/swarm-runs/latest.json")
	if !fileExists(currentPath) {
		t.Fatalf("latest pointer missing after update: %s", currentPath)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workflow", "verify", "--repo", root, "--resume", "latest"}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("workflow verify exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"status": "warning"`) || !strings.Contains(stdout.String(), `"phase-state"`) {
		t.Fatalf("workflow verify stdout = %q, want warning phase-state JSON", stdout.String())
	}
}
