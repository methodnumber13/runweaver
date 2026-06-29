package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLISmokeCodexDryRunCreatesDisposableRepo(t *testing.T) {
	root := filepath.Join(t.TempDir(), "codex-smoke")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{
		"smoke", "codex",
		"--repo", root,
		"--keep",
		"--codex-bin", "codex-test",
		"--timeout", "1s",
	}, &stdout, &stderr, false)

	if code != 0 {
		t.Fatalf("smoke codex exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{
		`"status": "ok"`,
		`"runtime": "codex"`,
		`"ready": true`,
		`"live": false`,
		`"kept": true`,
		`"repoRoot": "` + filepath.ToSlash(root) + `"`,
		`"evaluation"`,
		`"executionDryRun"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke stdout = %q, want %q", text, want)
		}
	}
	for _, path := range []string{
		"go.mod",
		"cmd/smoke/main.go",
		"cmd/smoke/main_test.go",
		".codex/runweaver/profile.json",
		".runweaver/workflows/feature-delivery-swarm.json",
	} {
		if !fileExists(filepath.Join(root, path)) {
			t.Fatalf("expected smoke artifact %s", filepath.Join(root, path))
		}
	}
}

func TestCLISmokeCodexRejectsExistingNonEmptyRepoWithoutForce(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "README.md", "# existing\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"smoke", "codex", "--repo", root}, &stdout, &stderr, false)

	if code != 1 {
		t.Fatalf("smoke codex exit code = %d, want 1 stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "already exists and is not empty") ||
		!strings.Contains(stderr.String(), "runweaver smoke codex") {
		t.Fatalf("stderr = %q, want actionable non-empty repo error", stderr.String())
	}
}
