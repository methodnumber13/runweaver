package aitools

import (
	"strings"
	"testing"
)

func TestValidateRuntimeMetadataFindsNoCrossRuntimeDriftAfterInitAll(t *testing.T) {
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

	issues, err := ValidateRuntimeMetadata(root, []string{RuntimeOpenCode, RuntimeCodex, RuntimeClaude})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) > 0 {
		t.Fatalf("runtime metadata drift issues = %#v, want none", issues)
	}
}

func TestValidateRuntimeMetadataReportsForeignRuntimePaths(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".codex/agents/swarm.toml", `name = "swarm"
description = "bad"
developer_instructions = "Use .opencode/agents/swarm.md and opencode run"
`)
	writeTestFile(t, root, ".agents/skills/context-discipline/SKILL.md", "---\nname: context-discipline\n---\n")
	writeTestFile(t, root, ".codex/runweaver/profile.json", "{}\n")

	issues, err := ValidateRuntimeMetadata(root, []string{RuntimeCodex})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) == 0 {
		t.Fatal("expected runtime drift issue for foreign OpenCode path in Codex metadata")
	}
}

func TestValidateRuntimeMetadataReportsForeignRuntimeMarkersForEachRuntime(t *testing.T) {
	for _, test := range []struct {
		name       string
		runtimeID  string
		file       string
		content    string
		wantMarker string
	}{
		{
			name:       "opencode_contains_codex_marker",
			runtimeID:  RuntimeOpenCode,
			file:       ".opencode/agents/swarm.md",
			content:    "Delegate through .codex/agents/swarm.toml with codex exec\n",
			wantMarker: ".codex/",
		},
		{
			name:       "opencode_contains_claude_marker",
			runtimeID:  RuntimeOpenCode,
			file:       ".opencode/skills/context-discipline/SKILL.md",
			content:    "Use .claude/agents/swarm.md and claude --print\n",
			wantMarker: ".claude/",
		},
		{
			name:       "codex_contains_opencode_marker",
			runtimeID:  RuntimeCodex,
			file:       ".codex/agents/swarm.toml",
			content:    "developer_instructions = \"Use .opencode/agents/swarm.md and opencode run\"\n",
			wantMarker: ".opencode/",
		},
		{
			name:       "codex_contains_claude_marker",
			runtimeID:  RuntimeCodex,
			file:       ".agents/skills/context-discipline/SKILL.md",
			content:    "Use .claude/skills/context-discipline/SKILL.md and claude --print\n",
			wantMarker: ".claude/",
		},
		{
			name:       "claude_contains_opencode_marker",
			runtimeID:  RuntimeClaude,
			file:       ".claude/agents/swarm.md",
			content:    "Use .opencode/agents/swarm.md and opencode run\n",
			wantMarker: ".opencode/",
		},
		{
			name:       "claude_contains_codex_marker",
			runtimeID:  RuntimeClaude,
			file:       ".claude/skills/context-discipline/SKILL.md",
			content:    "Use .codex/agents/swarm.toml and codex exec\n",
			wantMarker: ".codex/",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeMinimalRuntimeMetadata(t, root, test.runtimeID)
			writeTestFile(t, root, test.file, test.content)

			issues, err := ValidateRuntimeMetadata(root, []string{test.runtimeID})
			if err != nil {
				t.Fatal(err)
			}
			if !hasRuntimeIssue(issues, test.runtimeID, test.file, test.wantMarker) {
				t.Fatalf("issues = %#v, want %s marker issue for %s", issues, test.wantMarker, test.file)
			}
		})
	}
}

func TestValidateRuntimeMetadataReportsMissingRequiredFiles(t *testing.T) {
	root := t.TempDir()

	issues, err := ValidateRuntimeMetadata(root, []string{RuntimeOpenCode, RuntimeCodex, RuntimeClaude})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		runtimeID string
		file      string
	}{
		{RuntimeOpenCode, "opencode.json"},
		{RuntimeCodex, "AGENTS.md"},
		{RuntimeClaude, "CLAUDE.md"},
	} {
		if !hasRuntimeIssue(issues, want.runtimeID, want.file, "required runtime metadata is missing") {
			t.Fatalf("issues = %#v, want missing %s for %s", issues, want.file, want.runtimeID)
		}
	}
}

func writeMinimalRuntimeMetadata(t *testing.T, root, runtimeID string) {
	t.Helper()
	switch runtimeID {
	case RuntimeOpenCode:
		writeTestFile(t, root, "opencode.json", "{}\n")
		writeTestFile(t, root, ".opencode/agents/swarm.md", "---\ndescription: swarm\n---\n")
		writeTestFile(t, root, ".opencode/skills/context-discipline/SKILL.md", "---\nname: context-discipline\n---\n")
		writeTestFile(t, root, ".opencode/swarm/profile.json", "{}\n")
	case RuntimeCodex:
		writeTestFile(t, root, "AGENTS.md", "# Rules\n")
		writeTestFile(t, root, ".codex/agents/swarm.toml", "name = \"swarm\"\n")
		writeTestFile(t, root, ".agents/skills/context-discipline/SKILL.md", "---\nname: context-discipline\n---\n")
		writeTestFile(t, root, ".codex/runweaver/profile.json", "{}\n")
	case RuntimeClaude:
		writeTestFile(t, root, "CLAUDE.md", "# Rules\n")
		writeTestFile(t, root, ".claude/agents/swarm.md", "---\nname: swarm\n---\n")
		writeTestFile(t, root, ".claude/skills/context-discipline/SKILL.md", "---\nname: context-discipline\n---\n")
		writeTestFile(t, root, ".claude/runweaver/profile.json", "{}\n")
	default:
		t.Fatalf("unsupported runtime %q", runtimeID)
	}
}

func hasRuntimeIssue(issues []RuntimeMetadataIssue, runtimeID, file, reasonPart string) bool {
	for _, issue := range issues {
		if issue.Runtime == runtimeID && issue.File == file && strings.Contains(issue.Reason, reasonPart) {
			return true
		}
	}
	return false
}
