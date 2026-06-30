package aitools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteWorkflowDryRunWritesPromptAndCommand(t *testing.T) {
	root := t.TempDir()
	configHome := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("OPENCODE_CONFIG_CONTENT", "")
	t.Setenv("RUNWEAVER_MODEL_API_KEY", "test-key")
	writeTestFile(t, configHome, "opencode/opencode.json", `{
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
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {
      "id": "plan",
      "name": "Plan",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-surface-indexer"],
      "prompt": "plan"
    }
  ]
}`)

	result, err := ExecuteWorkflow(root, WorkflowExecuteOptions{
		WorkflowPath: ".runweaver/workflows/test-swarm.json",
		Task:         "update checkout flow",
		DryRun:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Executed {
		t.Fatalf("executed = true, want dry-run not executed")
	}
	if result.Status != "planned" || result.Plan.Workflow != "test-swarm" {
		t.Fatalf("result = %#v, want planned test-swarm", result)
	}
	command := strings.Join(result.Command, " ")
	for _, want := range []string{"opencode run", "--agent runweaver-swarm", "--dir " + root, "--format json"} {
		if !strings.Contains(command, want) {
			t.Fatalf("command = %q, want %q", command, want)
		}
	}
	prompt, err := os.ReadFile(filepath.Join(root, result.PromptPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"RunWeaver Swarm Execution", "RunWeaver command for this execution", "checkpoint.json", "repo-specific participants", "Delegate through OpenCode", "context-discipline", "maxParticipants", "participantRationale", "verificationResults", "workflow verify"} {
		if !strings.Contains(string(prompt), want) {
			t.Fatalf("prompt missing %q:\n%s", want, string(prompt))
		}
	}
}

func TestExecuteWorkflowCodexDryRunBuildsCodexCommandAndPrompt(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [{"id": "plan", "name": "Plan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "plan"}]
}`)

	result, err := ExecuteWorkflow(root, WorkflowExecuteOptions{
		Runtime:          RuntimeCodex,
		WorkflowPath:     ".runweaver/workflows/test-swarm.json",
		Task:             "update checkout flow",
		CodexBin:         "codex-test",
		Model:            "gpt-5.4",
		SkipGitRepoCheck: true,
		DryRun:           true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Runtime != RuntimeCodex || result.ModelPreflight != nil {
		t.Fatalf("result = %#v, want codex runtime without OpenCode preflight", result)
	}
	command := strings.Join(result.Command, " ")
	for _, want := range []string{"codex-test -a never exec", "--json", "--ephemeral", "-C " + root, "--sandbox workspace-write", "--model gpt-5.4", "--skip-git-repo-check"} {
		if !strings.Contains(command, want) {
			t.Fatalf("command = %q, want %q", command, want)
		}
	}
	for _, wantPath := range []string{"codex-exec-prompt.md", "codex-stdout.jsonl", "codex-stderr.log", "codex-final-message.md"} {
		if !strings.Contains(strings.Join([]string{result.PromptPath, result.StdoutPath, result.StderrPath, result.OutputPath}, " "), wantPath) {
			t.Fatalf("result paths = %#v, want %s", result, wantPath)
		}
	}
	prompt, err := os.ReadFile(filepath.Join(root, result.PromptPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"running through Codex", ".codex/runweaver/profile.json", "--classifier-runtime codex", "--replace-participants", "emulate selected roles", "Do not call child-agent spawn/wait", "Do not use web search"} {
		if !strings.Contains(string(prompt), want) {
			t.Fatalf("prompt missing %q:\n%s", want, string(prompt))
		}
	}
}

func TestExecuteWorkflowPassesRunWeaverBinaryToRuntimeEnv(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [{"id": "plan", "name": "Plan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "plan"}]
}`)
	var capturedEnv []string
	_, err := executeWorkflow(root, WorkflowExecuteOptions{
		Runtime:          RuntimeCodex,
		WorkflowPath:     ".runweaver/workflows/test-swarm.json",
		Task:             "update checkout flow",
		CodexBin:         "codex-test",
		SkipRuntimeCheck: true,
		SkipGitRepoCheck: true,
		Timeout:          0,
	}, func(ctx context.Context, dir string, name string, args []string, stdoutPath string, stderrPath string, env []string) (int, error) {
		capturedEnv = env
		return 0, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if envValue(capturedEnv, "RUNWEAVER_BIN") == "" {
		t.Fatalf("env = %#v, want RUNWEAVER_BIN", capturedEnv)
	}
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

func TestExecuteWorkflowClaudeDryRunBuildsClaudeCommandAndPrompt(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [{"id": "plan", "name": "Plan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "plan"}]
}`)

	result, err := ExecuteWorkflow(root, WorkflowExecuteOptions{
		Runtime:      RuntimeClaude,
		WorkflowPath: ".runweaver/workflows/test-swarm.json",
		Task:         "update checkout flow",
		ClaudeBin:    "claude-test",
		Model:        "sonnet",
		DryRun:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Runtime != RuntimeClaude || result.ModelPreflight != nil {
		t.Fatalf("result = %#v, want claude runtime without OpenCode preflight", result)
	}
	command := strings.Join(result.Command, " ")
	for _, want := range []string{"claude-test --print", "--output-format stream-json", "--permission-mode dontAsk", "--tools Read,Glob,Grep,Bash,Edit,MultiEdit,Write", "--model sonnet"} {
		if !strings.Contains(command, want) {
			t.Fatalf("command = %q, want %q", command, want)
		}
	}
	for _, wantPath := range []string{"claude-exec-prompt.md", "claude-stdout.jsonl", "claude-stderr.log"} {
		if !strings.Contains(strings.Join([]string{result.PromptPath, result.StdoutPath, result.StderrPath}, " "), wantPath) {
			t.Fatalf("result paths = %#v, want %s", result, wantPath)
		}
	}
	prompt, err := os.ReadFile(filepath.Join(root, result.PromptPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"running through Claude Code", ".claude/runweaver/profile.json", "Delegate through Claude Code"} {
		if !strings.Contains(string(prompt), want) {
			t.Fatalf("prompt missing %q:\n%s", want, string(prompt))
		}
	}
}

func TestExecuteWorkflowFailsWithoutReadyModelUnlessSkipped(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("OPENCODE_CONFIG_CONTENT", "")
	t.Setenv("RUNWEAVER_MODEL_API_KEY", "")
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": []
}`)

	_, err := ExecuteWorkflow(root, WorkflowExecuteOptions{
		WorkflowPath: ".runweaver/workflows/test-swarm.json",
		Task:         "check model",
		DryRun:       true,
	})
	if err == nil || !strings.Contains(err.Error(), "OpenCode model preflight failed") {
		t.Fatalf("err = %v, want model preflight failure", err)
	}
}
