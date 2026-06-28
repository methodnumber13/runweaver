package aitools

import "testing"

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
