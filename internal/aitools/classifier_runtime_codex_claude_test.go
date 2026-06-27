package aitools

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestIndexUsesCodexAIClassifierFinalMessage(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	var capturedName string
	var capturedArgs []string
	runner := func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		capturedName = name
		capturedArgs = append([]string{}, args...)
		outputPath := argValue(args, "--output-last-message")
		if outputPath == "" {
			t.Fatalf("args = %#v, want --output-last-message", args)
		}
		if err := os.WriteFile(outputPath, []byte(testClassifierJSON("checkout")), 0o644); err != nil {
			t.Fatal(err)
		}
		return []byte(`{"type":"turn.started"}` + "\n"), nil
	}

	index, err := indexWithOptions(root, IndexOptions{
		ChangedOnly: true,
		Classification: ClassifyOptions{
			Mode:             ClassificationAI,
			Runtime:          RuntimeCodex,
			CodexBin:         "codex-test",
			Model:            "gpt-5.4",
			SkipRuntimeCheck: true,
			SkipGitRepoCheck: true,
		},
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if capturedName != "codex-test" {
		t.Fatalf("name = %q, want codex-test", capturedName)
	}
	args := strings.Join(capturedArgs, " ")
	for _, want := range []string{"-a never exec", "--json", "--ephemeral", "--sandbox read-only", "--output-last-message", "--model gpt-5.4", "--skip-git-repo-check"} {
		if !strings.Contains(args, want) {
			t.Fatalf("args = %q, want %q", args, want)
		}
	}
	if index.Classification.Source != "model-backed-codex" || index.ClassifierRun == nil || index.ClassifierRun.Runtime != RuntimeCodex {
		t.Fatalf("classification=%#v run=%#v, want codex model-backed classification", index.Classification, index.ClassifierRun)
	}
}

func TestIndexUsesClaudeAIClassifierTextOutput(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	var capturedName string
	var capturedArgs []string
	runner := func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		capturedName = name
		capturedArgs = append([]string{}, args...)
		return []byte(testClassifierJSON("checkout")), nil
	}

	index, err := indexWithOptions(root, IndexOptions{
		ChangedOnly: true,
		Classification: ClassifyOptions{
			Mode:             ClassificationAI,
			Runtime:          RuntimeClaude,
			ClaudeBin:        "claude-test",
			Model:            "sonnet",
			PermissionMode:   "dontAsk",
			ClaudeTools:      "Read,Glob",
			SkipRuntimeCheck: true,
		},
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if capturedName != "claude-test" {
		t.Fatalf("name = %q, want claude-test", capturedName)
	}
	args := strings.Join(capturedArgs, " ")
	for _, want := range []string{"--print", "--output-format text", "--permission-mode dontAsk", "--tools Read,Glob", "--model sonnet"} {
		if !strings.Contains(args, want) {
			t.Fatalf("args = %q, want %q", args, want)
		}
	}
	if index.Classification.Source != "model-backed-claude" || index.ClassifierRun == nil || index.ClassifierRun.Runtime != RuntimeClaude {
		t.Fatalf("classification=%#v run=%#v, want claude model-backed classification", index.Classification, index.ClassifierRun)
	}
}
