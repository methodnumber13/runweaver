package aitools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedRuntimeInstructionsKeepStartupProtocolSnapshot(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{"scripts":{"test":"go test ./..."},"devDependencies":{"typescript":"latest"}}`)
	writeTestFile(t, root, "src/app.ts", "export const app = true\n")

	if _, err := InitSmartWithOptions(root, InitOptions{
		Force:          true,
		Runtime:        RuntimeAll,
		Classification: ClassifyOptions{Mode: ClassificationDeterministic},
	}); err != nil {
		t.Fatal(err)
	}

	assertGeneratedInstructionSnapshot(t, root, "AGENTS.md", []string{
		"# Repository AI Rules",
		"## RunWeaver Startup Protocol",
		`RUNWEAVER_BIN=".runweaver/tmp/bin/runweaver"`,
		`"$RUNWEAVER_BIN" start --repo . --task "<full raw user task>"`,
		"first active tool or shell action",
		"Do not read task-specific skills, source files, tests, or index artifacts before this start call",
		"Do not summarize or shorten",
		"executionContract",
		"action: resumed",
		"action: created",
		"$RUNWEAVER_BIN workflow update",
		"--file-changed <path>",
		"--last-result",
		"--rejected-path",
		"--next-verification",
		"--complete-phase",
		"Do not send a final response while the checkpoint status is `in_progress`",
		"$RUNWEAVER_BIN workflow verify --repo . --resume latest",
		"Do not ask the user to run start, resume, status, update, or verify commands manually",
	})
	assertGeneratedInstructionSnapshot(t, root, ".runweaver/START_HERE.md", []string{
		"# RunWeaver Start Here",
		`RUNWEAVER_BIN=".runweaver/tmp/bin/runweaver"`,
		`"$RUNWEAVER_BIN" start --repo . --task "<full raw user task>"`,
		"Do not summarize or shorten",
		`"$RUNWEAVER_BIN" status --repo .`,
		`"$RUNWEAVER_BIN" index --repo . --changed-only --prune`,
		".runweaver/tmp/swarm-runs",
		"selected workflow",
		"participants",
		".runweaver/tmp/current.md",
		".runweaver/workflows",
		"repo-local shim for `runweaver start`",
	})
	assertGeneratedInstructionSnapshot(t, root, ".opencode/agents/runweaver-swarm.md", []string{
		"Primary RunWeaver OpenCode agent",
		"Assume the user may only type a task into OpenCode",
		"## RunWeaver Startup Protocol",
		`RUNWEAVER_BIN=".runweaver/tmp/bin/runweaver"`,
		`"$RUNWEAVER_BIN" start --repo . --task`,
		"first active tool or shell action",
		"Do not read task-specific skills, source files, tests, or index artifacts before this start call",
		"--complete-phase",
		"Do not send a final response while the checkpoint status is `in_progress`",
		"## Planning And Execution Mode",
		"For normal coding, bugfix, refactor, or test tasks, the plan is only the durable checkpoint",
		"## Default Task Flow",
		`"$RUNWEAVER_BIN" start --repo . --runtime opencode`,
		`"$RUNWEAVER_BIN" workflow update --repo . --resume latest`,
		"--file-changed",
		"--last-result",
		"--rejected-path",
		"--next-verification",
		"--complete-phase",
		`"$RUNWEAVER_BIN" workflow verify --repo . --resume latest`,
		"resume is automatic via RunWeaver",
	})
	assertGeneratedInstructionSnapshot(t, root, ".codex/agents/runweaver-swarm.toml", []string{
		`name = "runweaver-swarm"`,
		`description = "Primary RunWeaver workflow coordinator for Codex"`,
		"Read AGENTS.md, .codex/runweaver/profile.json",
		`"$RUNWEAVER_BIN" start --repo . --runtime codex`,
		"## RunWeaver Startup Protocol",
		"first active tool or shell action",
		"Do not read task-specific skills, source files, tests, or index artifacts before this start call",
		"executionContract",
		"--complete-phase",
		"Do not send a final response while the checkpoint status is `in_progress`",
		"Use repo-specific agents from .codex/agents",
		"do not block on child-agent wait",
		"lastResult",
		"rejectedPaths",
		"nextVerification",
		"--file-changed",
		"--verification-result",
	})
	assertGeneratedInstructionSnapshot(t, root, "CLAUDE.md", []string{
		"# RunWeaver Repository AI Rules",
		"## RunWeaver Startup Protocol",
		`RUNWEAVER_BIN=".runweaver/tmp/bin/runweaver"`,
		`"$RUNWEAVER_BIN" start --repo . --task`,
		"first active tool or shell action",
		"Do not read task-specific skills, source files, tests, or index artifacts before this start call",
		"executionContract",
		"--complete-phase",
		"Do not send a final response while the checkpoint status is `in_progress`",
		`"$RUNWEAVER_BIN" start --repo . --runtime claude`,
	})
	assertGeneratedInstructionSnapshot(t, root, ".claude/agents/runweaver-swarm.md", []string{
		"name: runweaver-swarm",
		"description: Primary RunWeaver workflow coordinator for Claude Code",
		"Read CLAUDE.md, .claude/runweaver/profile.json",
		`"$RUNWEAVER_BIN" start --repo . --runtime claude`,
		"## RunWeaver Startup Protocol",
		"executionContract",
		"--complete-phase",
		"Do not send a final response while the checkpoint status is `in_progress`",
		"Use repo-specific agents from .claude/agents",
		"lastResult",
		"rejectedPaths",
		"nextVerification",
		"--file-changed",
		"--verification-result",
	})
}

func TestGeneratedInstructionSnapshotsKeepRuntimeBoundaries(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{"scripts":{"test":"go test ./..."},"devDependencies":{"typescript":"latest"}}`)
	writeTestFile(t, root, "src/app.ts", "export const app = true\n")

	if _, err := InitSmartWithOptions(root, InitOptions{
		Force:          true,
		Runtime:        RuntimeAll,
		Classification: ClassifyOptions{Mode: ClassificationDeterministic},
	}); err != nil {
		t.Fatal(err)
	}

	assertGeneratedInstructionExcludes(t, root, ".codex/agents/runweaver-swarm.toml", []string{
		".opencode/",
		"opencode run",
		".claude/",
		"claude --print",
		"--files-changed",
	})
	assertGeneratedInstructionExcludes(t, root, ".claude/agents/runweaver-swarm.md", []string{
		".opencode/",
		"opencode run",
		".codex/",
		"codex exec",
		"--files-changed",
	})
	assertGeneratedInstructionExcludes(t, root, ".opencode/agents/runweaver-swarm.md", []string{
		"codex exec",
		"claude --print",
		"--files-changed",
	})
	assertGeneratedInstructionExcludes(t, root, "AGENTS.md", []string{
		"--files-changed",
	})
}

func assertGeneratedInstructionSnapshot(t *testing.T, root, relPath string, ordered []string) {
	t.Helper()
	text := readGeneratedInstruction(t, root, relPath)
	assertContainsInOrder(t, text, ordered)
}

func assertGeneratedInstructionExcludes(t *testing.T, root, relPath string, unwanted []string) {
	t.Helper()
	text := readGeneratedInstruction(t, root, relPath)
	for _, value := range unwanted {
		if strings.Contains(text, value) {
			t.Fatalf("%s contains unwanted runtime fragment %q:\n%s", relPath, value, text)
		}
	}
}

func readGeneratedInstruction(t *testing.T, root, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		t.Fatalf("read generated instruction %s: %v", relPath, err)
	}
	return string(data)
}

func assertContainsInOrder(t *testing.T, text string, values []string) {
	t.Helper()
	offset := 0
	for _, value := range values {
		index := strings.Index(text[offset:], value)
		if index < 0 {
			t.Fatalf("text missing ordered fragment %q after offset %d:\n%s", value, offset, text)
		}
		offset += index + len(value)
	}
}
