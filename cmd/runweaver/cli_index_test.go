package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIIndexAndCleanCommands(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "package.json", `{
  "scripts": {
    "test": "go test ./..."
  },
  "dependencies": {
    "express": "latest"
  }
}`)
	writeCLIFile(t, root, "src/routes/orders.ts", "export function orders() { return null }\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"index", "--repo", root, "--changed-only", "--prune"}, &stdout, &stderr, false)

	if code != 0 {
		t.Fatalf("exit code = %d stderr=%q", code, stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("parse index JSON: %v\n%s", err, stdout.String())
	}
	artifacts, ok := payload["artifacts"].(map[string]any)
	if !ok || artifacts["compact"] == "" || artifacts["manifest"] == "" {
		t.Fatalf("payload artifacts = %#v, want compact and manifest", payload["artifacts"])
	}
	if !strings.Contains(stderr.String(), "indexed") {
		t.Fatalf("stderr = %q, want status", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"index", "clean", "--repo", root}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("clean exit code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"removed": true`) {
		t.Fatalf("stdout = %q, want removed true", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".runweaver/tmp")); !os.IsNotExist(err) {
		t.Fatalf("index dir still exists or stat failed unexpectedly: %v", err)
	}
}

func TestCLIScanStdoutAndIndexCleanAlreadyClean(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/clean\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"scan", "--repo", root}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("scan stdout exit code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"schemaVersion": 1`) {
		t.Fatalf("stdout = %q, want scan JSON", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLI([]string{"index", "clean", "--repo", root}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("index clean exit code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"removed": false`) {
		t.Fatalf("stdout = %q, want removed false", stdout.String())
	}
}
