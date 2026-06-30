package main

import (
	"bytes"
	"os"
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

func TestCLISmokeCodexLiveUsesRuntimeAndVerifiesFixture(t *testing.T) {
	root := filepath.Join(t.TempDir(), "codex-live-smoke")
	fakeCodex := filepath.Join(t.TempDir(), "codex")
	writeExecutable(t, fakeCodex, `#!/bin/sh
set -eu
repo=""
output=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -C)
      repo="$2"
      shift 2
      ;;
    --output-last-message)
      output="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
if [ -z "$repo" ]; then
  echo "missing repo" >&2
  exit 2
fi
sed 's/"before"/"after"/' "$repo/cmd/smoke/main.go" > "$repo/cmd/smoke/main.go.tmp"
mv "$repo/cmd/smoke/main.go.tmp" "$repo/cmd/smoke/main.go"
checkpoint=$(find "$repo/.runweaver/tmp/swarm-runs" -name checkpoint.json | sort | tail -n 1)
cat > "$checkpoint" <<'JSON'
{
  "schemaVersion": 1,
  "status": "complete",
  "completedPhases": ["reproduce", "fix", "verify"],
  "nextPhase": "",
  "participants": ["repo-implementation-agent"],
  "verification": ["go test ./..."],
  "verificationResults": ["go test ./... passed"]
}
JSON
if [ -n "$output" ]; then
  mkdir -p "$(dirname "$output")"
  printf 'fake codex final message\n' > "$output"
fi
printf '{"type":"message","message":"fake codex completed"}\n'
`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{
		"smoke", "codex",
		"--repo", root,
		"--live",
		"--keep",
		"--codex-bin", fakeCodex,
		"--timeout", "5s",
	}, &stdout, &stderr, false)

	if code != 0 {
		t.Fatalf("smoke codex live exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{
		`"status": "ok"`,
		`"live": true`,
		`"executed": true`,
		`"codex-smoke-go-test"`,
		`"Codex smoke fixture tests passed after live execution"`,
		"codex-final-message.md",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke stdout = %q, want %q", text, want)
		}
	}
	updated, err := os.ReadFile(filepath.Join(root, "cmd/smoke/main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), `"after"`) {
		t.Fatalf("main.go = %q, want live fake Codex edit", string(updated))
	}
}
