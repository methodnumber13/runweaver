package main

import (
	"bytes"
	"github.com/methodnumber13/runweaver/internal/aitools"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLIRefreshDoctorAndInitCommands(t *testing.T) {
	root := t.TempDir()
	isolateOpenCodeEnv(t)
	writeCLIFile(t, root, "package.json", `{
  "scripts": {
    "test": "vitest run"
  },
  "dependencies": {
    "react": "latest"
  },
  "devDependencies": {
    "vitest": "latest"
  }
}`)
	writeCLIFile(t, root, "src/App.tsx", "export function App() { return null }\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI([]string{"init", "--repo", root, "--force"}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("init exit code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status": "initialized"`) {
		t.Fatalf("init stdout = %q, want initialized", stdout.String())
	}
	if !strings.Contains(stderr.String(), "init [") || !strings.Contains(stderr.String(), "index-classify") {
		t.Fatalf("init stderr = %q, want progress output", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"refresh", "--repo", root}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("refresh exit code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"surfaceIndexPath"`) {
		t.Fatalf("refresh stdout = %q, want surfaceIndexPath", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"doctor", "--repo", root}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("doctor exit code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status"`) {
		t.Fatalf("doctor stdout = %q, want status", stdout.String())
	}
}

func TestCLIInitCodexRuntimeSkipsOpenCodeModelPreflight(t *testing.T) {
	root := t.TempDir()
	isolateOpenCodeEnv(t)
	writeCLIFile(t, root, "package.json", `{"scripts":{"test":"go test ./..."},"dependencies":{"express":"latest"}}`)
	writeCLIFile(t, root, "src/app.ts", "export function app() { return null }\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"init", "--repo", root, "--runtime", "codex", "--classification", "deterministic", "--force"}, &stdout, &stderr, false)

	if code != 0 {
		t.Fatalf("init codex exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"runtime": "codex"`) || !strings.Contains(stdout.String(), `"status": "skipped"`) {
		t.Fatalf("stdout = %q, want codex runtime and skipped model preflight", stdout.String())
	}
	if !strings.Contains(stderr.String(), "OpenCode model preflight was skipped") || strings.Contains(stderr.String(), "warning: initialized repository") {
		t.Fatalf("stderr = %q, want skipped preflight success without warning", stderr.String())
	}
}

func TestCLIBootstrapAliasesInit(t *testing.T) {
	root := t.TempDir()
	isolateOpenCodeEnv(t)
	writeCLIFile(t, root, "package.json", `{"scripts":{"test":"jest"},"dependencies":{"@nestjs/core":"10.0.0"},"devDependencies":{"jest":"29.0.0","typescript":"5.0.0"}}`)
	writeCLIFile(t, root, "src/app.controller.ts", "export class AppController {}\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI([]string{"bootstrap", "--repo", root, "--runtime", "codex", "--classification", "deterministic", "--force"}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("bootstrap exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	for _, path := range []string{
		"AGENTS.md",
		".codex/agents/swarm.toml",
		".codex/runweaver/profile.json",
		".runweaver/START_HERE.md",
	} {
		if !fileExists(filepath.Join(root, path)) {
			t.Fatalf("expected bootstrap artifact %s", path)
		}
	}
	if !strings.Contains(stdout.String(), `"runtime": "codex"`) {
		t.Fatalf("bootstrap stdout = %q, want codex runtime", stdout.String())
	}
}

func TestCLIInteractiveProgressRendersSingleClearedLine(t *testing.T) {
	t.Setenv("COLUMNS", "72")
	var stderr bytes.Buffer
	app := cli{stderr: &stderr, color: true}

	app.printProgress(aitools.InitProgressEvent{
		Current: 5,
		Total:   8,
		Step:    "index-classify",
		Message: "Indexing repository and running ai classifier through opencode/repo-classifier with model configured default model and timeout 2m0s",
		Elapsed: 12 * time.Second,
		Pulse:   0,
	}, "|")

	text := stderr.String()
	if !strings.HasPrefix(text, "\r\x1b[2K") {
		t.Fatalf("progress = %q, want carriage return plus clear-line prefix", text)
	}
	if strings.Contains(text, "\n") {
		t.Fatalf("progress = %q, want no newline for interactive tick", text)
	}
	if !strings.Contains(text, "init [############>-------] 5/8 index-classify 00:12") {
		t.Fatalf("progress = %q, want init progress line", text)
	}
	if !strings.HasSuffix(text, "...") {
		t.Fatalf("progress = %q, want truncated long message", text)
	}
	if len(text) > 100 {
		t.Fatalf("progress length = %d, want compact single line: %q", len(text), text)
	}
}
