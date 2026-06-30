package aitools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDoctorOpenCodeReportsReadyWhenRunWeaverAgentIsResolved(t *testing.T) {
	root := t.TempDir()
	makeFakeRunWeaver(t)
	writeTestFile(t, root, ".opencode/agents/runweaver-swarm.md", "---\nmode: primary\n---\n")
	writeTestFile(t, root, ".opencode/skills/repo-onboarding/SKILL.md", "# skill\n")

	result, err := doctorOpenCode(root, OpenCodeDoctorOptions{SkipModelCheck: true}, func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		command := strings.Join(args, " ")
		switch command {
		case "debug config":
			return []byte(`{
  "default_agent": "runweaver-swarm",
  "permission": {
    "task": "allow",
    "todowrite": "allow"
  },
  "agent": {
    "runweaver-swarm": {}
  }
}`), nil
		case "debug agent runweaver-swarm":
			return []byte(`{
  "name": "runweaver-swarm",
  "mode": "primary",
  "tools": {
    "task": true,
    "todowrite": true
  }
}`), nil
		case "agent list":
			return []byte("runweaver-swarm\nrepo-surface-indexer\n"), nil
		default:
			t.Fatalf("unexpected command: %s", command)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ok" {
		t.Fatalf("result = %#v, want ready ok", result)
	}
	if !hasDoctorCheck(result, "resolved-agent-tools", "ok") || !hasDoctorCheck(result, "default-agent", "ok") {
		t.Fatalf("checks = %#v, want default-agent and resolved-agent-tools ok", result.Checks)
	}
}

func TestDoctorOpenCodeWarnsWhenDefaultAgentOrToolsAreWrong(t *testing.T) {
	root := t.TempDir()
	makeFakeRunWeaver(t)
	writeTestFile(t, root, ".opencode/agents/runweaver-swarm.md", "---\nmode: primary\n---\n")
	writeTestFile(t, root, ".opencode/skills/repo-onboarding/SKILL.md", "# skill\n")

	result, err := doctorOpenCode(root, OpenCodeDoctorOptions{SkipModelCheck: true}, func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		command := strings.Join(args, " ")
		switch command {
		case "debug config":
			return []byte(`{
  "default_agent": "build",
  "permission": {
    "task": "ask"
  },
  "agent": {}
}`), nil
		case "debug agent runweaver-swarm":
			return []byte(`{
  "name": "runweaver-swarm",
  "mode": "primary",
  "tools": {
    "task": true
  }
}`), nil
		case "agent list":
			return []byte("build\n"), nil
		default:
			t.Fatalf("unexpected command: %s", command)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status != "error" {
		t.Fatalf("result = %#v, want not ready error", result)
	}
	if !hasDoctorCheck(result, "default-agent", "warning") || !hasDoctorCheck(result, "resolved-agent-tools", "error") {
		t.Fatalf("checks = %#v, want warnings/errors", result.Checks)
	}
}

func TestDoctorOpenCodeDoesNotFailReadinessForMissingTopLevelToolPermissionsWhenAgentAllowsTools(t *testing.T) {
	root := t.TempDir()
	makeFakeRunWeaver(t)
	writeTestFile(t, root, ".opencode/agents/runweaver-swarm.md", "---\nmode: primary\n---\n")
	writeTestFile(t, root, ".opencode/skills/repo-onboarding/SKILL.md", "# skill\n")

	result, err := doctorOpenCode(root, OpenCodeDoctorOptions{SkipModelCheck: true}, func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		command := strings.Join(args, " ")
		switch command {
		case "debug config":
			return []byte(`{
  "default_agent": "runweaver-swarm",
  "permission": {},
  "agent": {
    "runweaver-swarm": {}
  }
}`), nil
		case "debug agent runweaver-swarm":
			return []byte(`{
  "name": "runweaver-swarm",
  "mode": "primary",
  "tools": {
    "task": true,
    "todowrite": true
  }
}`), nil
		case "agent list":
			return []byte("runweaver-swarm\n"), nil
		default:
			t.Fatalf("unexpected command: %s", command)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ok" {
		t.Fatalf("result = %#v, want ready ok because agent-level tools are allowed", result)
	}
	if !hasDoctorCheck(result, "resolved-permissions", "info") {
		t.Fatalf("checks = %#v, want informational top-level permission check", result.Checks)
	}
}

func TestDoctorOpenCodeUsesFreshTimeoutForEachOpenCodeCommand(t *testing.T) {
	root := t.TempDir()
	makeFakeRunWeaver(t)
	writeTestFile(t, root, ".opencode/agents/runweaver-swarm.md", "---\nmode: primary\n---\n")
	writeTestFile(t, root, ".opencode/skills/repo-onboarding/SKILL.md", "# skill\n")

	result, err := doctorOpenCode(root, OpenCodeDoctorOptions{SkipModelCheck: true, Timeout: 30 * time.Millisecond}, func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		command := strings.Join(args, " ")
		switch command {
		case "debug config":
			<-ctx.Done()
			return nil, ctx.Err()
		case "debug agent runweaver-swarm":
			if ctx.Err() != nil {
				t.Fatalf("debug agent received an already-expired context")
			}
			return []byte(`{
  "name": "runweaver-swarm",
  "mode": "primary",
  "tools": {
    "task": true,
    "todowrite": true
  }
}`), nil
		case "agent list":
			if ctx.Err() != nil {
				t.Fatalf("agent list received an already-expired context")
			}
			return []byte("runweaver-swarm\n"), nil
		default:
			t.Fatalf("unexpected command: %s", command)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasDoctorCheck(result, "opencode-debug-config", "error") {
		t.Fatalf("checks = %#v, want debug config timeout error", result.Checks)
	}
	if !hasDoctorCheck(result, "resolved-agent-tools", "ok") || !hasDoctorCheck(result, "opencode-agent-list", "ok") {
		t.Fatalf("checks = %#v, want later OpenCode commands to run with fresh contexts", result.Checks)
	}
}

func hasDoctorCheck(result OpenCodeDoctorResult, name, status string) bool {
	for _, check := range result.Checks {
		if check.Name == name && check.Status == status {
			return true
		}
	}
	return false
}

func makeFakeRunWeaver(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	path := filepath.Join(binDir, "runweaver")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
