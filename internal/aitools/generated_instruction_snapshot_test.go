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
		`runweaver start --repo . --task "<user task>"`,
		"executionContract",
		"action: resumed",
		"action: created",
		"runweaver workflow update",
		"lastResult",
		"rejectedPaths",
		"nextVerification",
		"--complete-phase",
		"Do not send a final response while the checkpoint status is `in_progress`",
		"runweaver workflow verify --repo . --resume latest",
		"Do not ask the user to run start, resume, status, update, or verify commands manually",
	})
	assertGeneratedInstructionSnapshot(t, root, ".runweaver/START_HERE.md", []string{
		"# RunWeaver Start Here",
		`runweaver start --repo . --task "<user task>"`,
		"runweaver status --repo .",
		"runweaver index --repo . --changed-only --prune",
		".runweaver/tmp/swarm-runs",
		"selected workflow",
		"participants",
		".runweaver/tmp/current.md",
		".runweaver/workflows",
		"Agents should call `runweaver start` for non-trivial tasks",
	})
	assertGeneratedInstructionSnapshot(t, root, ".opencode/agents/runweaver-swarm.md", []string{
		"Primary RunWeaver OpenCode agent",
		"Assume the user may only type a task into OpenCode",
		"## RunWeaver Startup Protocol",
		"runweaver start --repo . --task",
		"--complete-phase",
		"Do not send a final response while the checkpoint status is `in_progress`",
		"## Planning And Execution Mode",
		"For normal coding, bugfix, refactor, or test tasks, the plan is only the durable checkpoint",
		"## Default Task Flow",
		"runweaver start --repo . --runtime opencode",
		"runweaver workflow update --repo . --resume latest",
		"--last-result",
		"--rejected-path",
		"--next-verification",
		"--complete-phase",
		"runweaver workflow verify --repo . --resume latest",
		"resume is automatic via RunWeaver",
	})
	assertGeneratedInstructionSnapshot(t, root, ".codex/agents/runweaver-swarm.toml", []string{
		`name = "runweaver-swarm"`,
		`description = "Primary RunWeaver workflow coordinator for Codex"`,
		"Read AGENTS.md, .codex/runweaver/profile.json",
		"runweaver start --repo . --runtime codex",
		"## RunWeaver Startup Protocol",
		"executionContract",
		"--complete-phase",
		"Do not send a final response while the checkpoint status is `in_progress`",
		"Use repo-specific agents from .codex/agents",
		"do not block on child-agent wait",
		"lastResult",
		"rejectedPaths",
		"nextVerification",
	})
	assertGeneratedInstructionSnapshot(t, root, "CLAUDE.md", []string{
		"# RunWeaver Repository AI Rules",
		"## RunWeaver Startup Protocol",
		"runweaver start --repo . --task",
		"executionContract",
		"--complete-phase",
		"Do not send a final response while the checkpoint status is `in_progress`",
		"runweaver start --repo . --runtime claude",
	})
	assertGeneratedInstructionSnapshot(t, root, ".claude/agents/runweaver-swarm.md", []string{
		"name: runweaver-swarm",
		"description: Primary RunWeaver workflow coordinator for Claude Code",
		"Read CLAUDE.md, .claude/runweaver/profile.json",
		"runweaver start --repo . --runtime claude",
		"## RunWeaver Startup Protocol",
		"executionContract",
		"--complete-phase",
		"Do not send a final response while the checkpoint status is `in_progress`",
		"Use repo-specific agents from .claude/agents",
		"lastResult",
		"rejectedPaths",
		"nextVerification",
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
	})
	assertGeneratedInstructionExcludes(t, root, ".claude/agents/runweaver-swarm.md", []string{
		".opencode/",
		"opencode run",
		".codex/",
		"codex exec",
	})
	assertGeneratedInstructionExcludes(t, root, ".opencode/agents/runweaver-swarm.md", []string{
		"codex exec",
		"claude --print",
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
