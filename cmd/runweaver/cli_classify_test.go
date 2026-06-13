package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCLIClassifyDeterministicCommand(t *testing.T) {
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

	code := runCLI([]string{"classify", "--repo", root, "--classification", "deterministic"}, &stdout, &stderr, false)

	if code != 0 {
		t.Fatalf("exit code = %d stderr=%q", code, stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("parse classify JSON: %v\n%s", err, stdout.String())
	}
	classifier, ok := payload["classifier"].(map[string]any)
	if !ok || classifier["mode"] != "deterministic" || classifier["source"] != "deterministic-semantic-fallback" {
		t.Fatalf("classifier = %#v, want deterministic fallback source", payload["classifier"])
	}
	if !strings.Contains(stderr.String(), "repository classified") {
		t.Fatalf("stderr = %q, want classify status", stderr.String())
	}
}

func TestCLIClassificationDefaultTimeoutIs180Seconds(t *testing.T) {
	fs := newFlagSet("test")
	flags := addClassificationFlags(fs, "ai")
	if err := fs.Parse(nil); err != nil {
		t.Fatal(err)
	}
	opts, err := flags.options()
	if err != nil {
		t.Fatal(err)
	}
	if opts.Timeout != 180*time.Second {
		t.Fatalf("timeout = %s, want 180s", opts.Timeout)
	}
}
