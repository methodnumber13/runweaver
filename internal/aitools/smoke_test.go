package aitools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCodexSmokeCleansDisposableRepoByDefault(t *testing.T) {
	result, err := RunCodexSmoke(CodexSmokeOptions{
		CodexBin: "codex-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Created || result.Kept {
		t.Fatalf("result = %#v, want created disposable repo that is not kept", result)
	}
	if _, err := os.Stat(result.RepoRoot); !os.IsNotExist(err) {
		t.Fatalf("repo root %s still exists after disposable smoke; stat err=%v", result.RepoRoot, err)
	}
}

func TestRunCodexSmokeLiveCleansDisposableRepoByDefault(t *testing.T) {
	fakeCodex := filepath.Join(t.TempDir(), "codex")
	writeTestExecutable(t, fakeCodex, `#!/bin/sh
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
  "participantRationale": ["implementation agent owns the smoke edit"],
  "filesChanged": ["cmd/smoke/main.go"],
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

	result, err := RunCodexSmoke(CodexSmokeOptions{
		Live:     true,
		CodexBin: fakeCodex,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready {
		t.Fatalf("result = %#v, want ready live smoke", result)
	}
	if !strings.Contains(result.RepoRoot, "runweaver-codex-smoke-") {
		t.Fatalf("repo root = %q, want disposable smoke path", result.RepoRoot)
	}
	if _, err := os.Stat(result.RepoRoot); !os.IsNotExist(err) {
		t.Fatalf("repo root %s still exists after live disposable smoke; stat err=%v", result.RepoRoot, err)
	}
}

func writeTestExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
