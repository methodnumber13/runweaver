package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIDoctorModelCommandWithProjectConfig(t *testing.T) {
	root := t.TempDir()
	isolateOpenCodeEnv(t)
	t.Setenv("RUNWEAVER_MODEL_API_KEY", "test-key")
	writeCLIFile(t, root, "opencode.json", `{
  "model": "openai-compatible/coder-model",
  "provider": {
    "openai-compatible": {
      "options": {
        "baseURL": "https://llm-provider.example.com/v1",
        "apiKey": "{env:RUNWEAVER_MODEL_API_KEY}"
      }
    }
  }
}`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"doctor", "model", "--repo", root, "--provider", "openai-compatible", "--model", "coder-model", "--base-url", "https://llm-provider.example.com/v1"}, &stdout, &stderr, false)

	if code != 0 {
		t.Fatalf("doctor model exit code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"ready": true`) {
		t.Fatalf("stdout = %q, want ready true", stdout.String())
	}
}

func TestCLIDoctorOpenCodeCommandWithFakeOpenCode(t *testing.T) {
	root := t.TempDir()
	binDir := t.TempDir()
	writeCLIFile(t, root, ".opencode/agents/swarm.md", "---\nmode: primary\n---\n")
	writeCLIFile(t, root, ".opencode/skills/repo-onboarding/SKILL.md", "# skill\n")
	writeExecutable(t, filepath.Join(binDir, "runweaver"), "#!/bin/sh\nexit 0\n")
	writeExecutable(t, filepath.Join(binDir, "opencode"), `#!/bin/sh
case "$1 $2 $3" in
  "debug config ")
    cat <<'JSON'
{"default_agent":"swarm","permission":{"task":"allow","todowrite":"allow"},"agent":{"swarm":{}}}
JSON
    ;;
  "debug agent swarm")
    cat <<'JSON'
{"name":"swarm","mode":"primary","tools":{"task":true,"todowrite":true}}
JSON
    ;;
  "agent list ")
    echo "swarm"
    ;;
  *)
    echo "unexpected $*" >&2
    exit 2
    ;;
esac
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"doctor", "opencode", "--repo", root, "--skip-model-check", "--timeout", "5s"}, &stdout, &stderr, false)

	if code != 0 {
		t.Fatalf("doctor opencode exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"ready": true`) || !strings.Contains(stdout.String(), `"resolved-agent-tools"`) {
		t.Fatalf("stdout = %q, want ready OpenCode doctor result", stdout.String())
	}
}

func TestCLIDoctorRuntimeCommand(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, ".codex/config.toml", "[features]\nmulti_agent = true\n")
	writeCLIFile(t, root, ".codex/agents/swarm.toml", "name = \"swarm\"\n")
	writeCLIFile(t, root, ".agents/skills/context-discipline/SKILL.md", "---\nname: context-discipline\n---\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"doctor", "runtime", "--repo", root, "--runtime", "codex"}, &stdout, &stderr, false)

	if code != 0 {
		t.Fatalf("doctor runtime exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"id": "codex"`) || !strings.Contains(stdout.String(), `"configFiles"`) {
		t.Fatalf("stdout = %q, want codex runtime discovery", stdout.String())
	}
	if !strings.Contains(stderr.String(), "runtime provider discovery complete") {
		t.Fatalf("stderr = %q, want runtime status", stderr.String())
	}
}

func TestCLIDoctorProcessesSummaryCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"doctor", "processes", "--summary"}, &stdout, &stderr, false)

	if code != 0 {
		t.Fatalf("doctor processes exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("parse doctor processes JSON: %v\n%s", err, stdout.String())
	}
	if payload["status"] == "" || payload["summary"] == "" {
		t.Fatalf("payload = %#v, want status and summary", payload)
	}
	if _, ok := payload["vscode"].(map[string]any); !ok {
		t.Fatalf("payload = %#v, want vscode object", payload)
	}
	if !strings.Contains(stderr.String(), "process doctor complete") {
		t.Fatalf("stderr = %q, want process doctor status", stderr.String())
	}
}
