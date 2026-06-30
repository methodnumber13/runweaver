package aitools

import (
	"context"
	"github.com/methodnumber13/runweaver/internal/aitools/runtimeenv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteWorkflowRunsCodexCommandWithoutOpenCodePreflight(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [{"id": "plan", "name": "Plan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "plan"}]
}`)
	var capturedName string
	var capturedArgs []string
	var capturedEnv []string
	runner := func(ctx context.Context, dir string, name string, args []string, stdoutPath string, stderrPath string, env []string) (int, error) {
		capturedName = name
		capturedArgs = append([]string{}, args...)
		capturedEnv = append([]string{}, env...)
		checkpoints, err := filepath.Glob(filepath.Join(root, ".runweaver/tmp/swarm-runs/*/checkpoint.json"))
		if err != nil {
			t.Fatal(err)
		}
		if len(checkpoints) != 1 {
			t.Fatalf("checkpoints = %#v, want one checkpoint", checkpoints)
		}
		if err := os.WriteFile(checkpoints[0], []byte(`{
  "schemaVersion": 1,
  "workflow": "test-swarm",
  "task": "execute",
  "status": "complete",
  "completedPhases": ["plan"],
  "nextPhase": "",
  "updatedAt": "2026-05-30T00:00:00Z"
}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(stdoutPath, []byte(`{"type":"turn.completed"}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(stderrPath, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
		return 0, nil
	}

	result, err := executeWorkflow(root, WorkflowExecuteOptions{
		Runtime:          RuntimeCodex,
		WorkflowPath:     ".runweaver/workflows/test-swarm.json",
		Task:             "execute",
		CodexBin:         "codex-test",
		SkipRuntimeCheck: true,
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if result.ModelPreflight != nil {
		t.Fatalf("modelPreflight = %#v, want nil for codex", result.ModelPreflight)
	}
	if capturedName != "codex-test" {
		t.Fatalf("name = %q, want codex-test", capturedName)
	}
	args := strings.Join(capturedArgs, " ")
	if !strings.Contains(args, "-a never exec") || !strings.Contains(args, "--json") {
		t.Fatalf("args = %q, want codex exec JSON command", args)
	}
	if envValue(capturedEnv, "RUNWEAVER_BIN") == "" {
		t.Fatalf("env = %#v, want RUNWEAVER_BIN", capturedEnv)
	}
	if envValue(capturedEnv, "RUNWEAVER_MODEL_API_KEY") != "" {
		t.Fatalf("env = %#v, want no OpenCode model env mutation", capturedEnv)
	}
	if !result.Executed || result.Status != "success" {
		t.Fatalf("result = %#v, want codex executed success", result)
	}
}

func TestExecuteWorkflowRunsConfiguredCommand(t *testing.T) {
	root := t.TempDir()
	writeTestOpenCodeConfig(t, root)
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": []
}`)
	var capturedName string
	var capturedArgs []string
	var capturedEnv []string
	runner := func(ctx context.Context, dir string, name string, args []string, stdoutPath string, stderrPath string, env []string) (int, error) {
		capturedName = name
		capturedArgs = append([]string{}, args...)
		capturedEnv = append([]string{}, env...)
		if dir != root {
			t.Fatalf("dir = %q, want root", dir)
		}
		checkpoints, err := filepath.Glob(filepath.Join(root, ".runweaver/tmp/swarm-runs/*/checkpoint.json"))
		if err != nil {
			t.Fatal(err)
		}
		if len(checkpoints) != 1 {
			t.Fatalf("checkpoints = %#v, want one checkpoint", checkpoints)
		}
		if err := os.WriteFile(checkpoints[0], []byte(`{
  "schemaVersion": 1,
  "workflow": "test-swarm",
  "task": "execute",
  "status": "in_progress",
  "completedPhases": ["plan"],
  "nextPhase": "",
  "updatedAt": "2026-05-30T00:00:00Z"
}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(stdoutPath, []byte(`{"type":"done"}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(stderrPath, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
		return 0, nil
	}

	result, err := executeWorkflow(root, WorkflowExecuteOptions{
		WorkflowPath:     ".runweaver/workflows/test-swarm.json",
		Task:             "execute",
		OpencodeBin:      "opencode-test",
		Model:            "openai-compatible/coder-model",
		SkipModelCheck:   true,
		SkipRuntimeCheck: true,
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Executed || result.Status != "success" {
		t.Fatalf("result = %#v, want executed success", result)
	}
	if result.PostCheck == nil || result.PostCheck.CompletedPhases != 1 {
		t.Fatalf("postCheck = %#v, want completed phase evidence", result.PostCheck)
	}
	if capturedName != "opencode-test" {
		t.Fatalf("name = %q, want opencode-test", capturedName)
	}
	args := strings.Join(capturedArgs, " ")
	if !strings.Contains(args, "--model openai-compatible/coder-model") || !strings.Contains(args, "--agent runweaver-swarm") {
		t.Fatalf("args = %q, want model and RunWeaver OpenCode agent", args)
	}
	if !strings.Contains(runtimeenv.EnvValue(capturedEnv, "NO_PROXY"), "llm-provider.example.com") {
		t.Fatalf("NO_PROXY = %q, want OpenAI-compatible host", runtimeenv.EnvValue(capturedEnv, "NO_PROXY"))
	}
	if !Exists(filepath.Join(root, result.StdoutPath)) || !Exists(filepath.Join(root, result.StderrPath)) {
		t.Fatalf("expected stdout/stderr logs: %#v", result)
	}
}

func writeTestOpenCodeConfig(t *testing.T, root string) {
	t.Helper()
	writeTestFile(t, root, "opencode.json", `{
  "model": "openai-compatible/coder-model",
  "provider": {
    "openai-compatible": {
      "name": "openai-compatible",
      "npm": "@ai-sdk/openai-compatible",
      "models": {
        "coder-model": {
          "name": "coder-model"
        }
      },
      "options": {
        "baseURL": "https://llm-provider.example.com/v1",
        "apiKey": "{env:RUNWEAVER_MODEL_API_KEY}"
      }
    }
  }
}`)
}
