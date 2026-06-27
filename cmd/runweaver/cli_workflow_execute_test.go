package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIWorkflowExecuteDryRunCommand(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": []
}`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{
		"workflow", "run",
		"--repo", root,
		"--workflow", ".runweaver/workflows/test-swarm.json",
		"--task", "dry execute",
		"--execute",
		"--dry-run",
		"--skip-model-check",
		"--model", "openai-compatible/coder-model",
	}, &stdout, &stderr, false)

	if code != 0 {
		t.Fatalf("workflow execute dry-run exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"dryRun": true`) || !strings.Contains(stdout.String(), `"opencode"`) {
		t.Fatalf("stdout = %q, want dry-run opencode command", stdout.String())
	}
}

func TestCLIWorkflowExecuteCodexDryRunCommand(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": []
}`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{
		"workflow", "run",
		"--repo", root,
		"--workflow", ".runweaver/workflows/test-swarm.json",
		"--task", "dry execute",
		"--execute",
		"--dry-run",
		"--runtime", "codex",
		"--codex-bin", "codex-test",
		"--model", "gpt-5.4",
		"--skip-git-repo-check",
	}, &stdout, &stderr, false)

	if code != 0 {
		t.Fatalf("workflow codex dry-run exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("parse codex dry-run JSON: %v\n%s", err, stdout.String())
	}
	if payload["runtime"] != "codex" || !strings.Contains(fmt.Sprint(payload["command"]), "codex-test") || !strings.Contains(fmt.Sprint(payload["stdoutPath"]), "codex-stdout.jsonl") {
		t.Fatalf("payload = %#v, want dry-run codex command", payload)
	}
}

func TestCLIWorkflowExecuteClaudeDryRunCommand(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": []
}`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{
		"workflow", "run",
		"--repo", root,
		"--workflow", ".runweaver/workflows/test-swarm.json",
		"--task", "dry execute",
		"--execute",
		"--dry-run",
		"--runtime", "claude",
		"--claude-bin", "claude-test",
		"--model", "sonnet",
		"--permission-mode", "plan",
		"--claude-tools", "Read,Glob",
	}, &stdout, &stderr, false)

	if code != 0 {
		t.Fatalf("workflow claude dry-run exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("parse claude dry-run JSON: %v\n%s", err, stdout.String())
	}
	if payload["runtime"] != "claude" || !strings.Contains(fmt.Sprint(payload["command"]), "claude-test") || !strings.Contains(fmt.Sprint(payload["stdoutPath"]), "claude-stdout.jsonl") {
		t.Fatalf("payload = %#v, want dry-run claude command", payload)
	}
}

func TestCLIWorkflowResumeExecuteDryRunCommand(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": []
}`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI([]string{"workflow", "run", "--repo", root, "--workflow", ".runweaver/workflows/test-swarm.json", "--task", "plan"}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("workflow plan exit code = %d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"workflow", "run", "--repo", root, "--resume", "latest", "--execute", "--dry-run", "--skip-model-check"}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("workflow resume execute exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"status": "planned"`) || !strings.Contains(stdout.String(), `"dryRun": true`) {
		t.Fatalf("stdout = %q, want resumed dry-run", stdout.String())
	}
}

func TestCLIWorkflowExecuteRuntimeErrorIsNotUsageError(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": []
}`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{
		"workflow", "run",
		"--repo", root,
		"--workflow", ".runweaver/workflows/test-swarm.json",
		"--task", "execute",
		"--execute",
		"--skip-model-check",
		"--skip-runtime-check",
		"--opencode-bin", filepath.Join(root, "missing-opencode"),
	}, &stdout, &stderr, false)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), `"command"`) || !strings.Contains(stdout.String(), `"status": "error"`) {
		t.Fatalf("stdout = %q, want partial execution JSON", stdout.String())
	}
	if !strings.Contains(stderr.String(), "execute workflow") {
		t.Fatalf("stderr = %q, want execution error", stderr.String())
	}
	if strings.Contains(stderr.String(), "hint:") {
		t.Fatalf("stderr = %q, want no usage hint for runtime execution error", stderr.String())
	}
}
