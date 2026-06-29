package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIEvalAdoptionRunsSmoke(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/tool\n")
	writeCLIFile(t, root, "cmd/tool/main.go", "package main\nfunc main() {}\n")
	code := runCLI([]string{"init", "--repo", root, "--runtime", "opencode", "--force", "--classification", "deterministic"}, &bytes.Buffer{}, &bytes.Buffer{}, false)
	if code != 0 {
		t.Fatalf("init exit code = %d", code)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code = runCLI([]string{"eval", "adoption", "--repo", root, "--runtime", "opencode", "--task", "Add CLI smoke behavior"}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("eval adoption exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"status": "ok"`) ||
		!strings.Contains(stdout.String(), `"action": "created"`) ||
		!strings.Contains(stdout.String(), `"doctor"`) {
		t.Fatalf("eval stdout = %q, want ok doctor/start smoke", stdout.String())
	}
}

func TestCLIEvalAdoptionPreparesCodexDryRunWithOverrides(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/tool\n")
	writeCLIFile(t, root, "cmd/tool/main.go", "package main\nfunc main() {}\n")
	code := runCLI([]string{"init", "--repo", root, "--runtime", "codex", "--force", "--classification", "deterministic"}, &bytes.Buffer{}, &bytes.Buffer{}, false)
	if code != 0 {
		t.Fatalf("init exit code = %d", code)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code = runCLI([]string{
		"eval", "adoption",
		"--repo", root,
		"--runtime", "codex",
		"--task", "Add Codex adoption smoke behavior",
		"--codex-bin", "codex-test",
		"--model", "test-model",
		"--timeout", "1s",
		"--skip-git-repo-check",
	}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("eval adoption exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{`"runtime": "codex"`, `"executionDryRun"`, `"dryRun": true`, "codex-test", "test-model", "--skip-git-repo-check"} {
		if !strings.Contains(text, want) {
			t.Fatalf("eval stdout = %q, want %q", text, want)
		}
	}
}
