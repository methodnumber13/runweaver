package aitools

import "testing"

func TestDriftScansAllRuntimeGeneratedMetadata(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "src/existing.ts", "export const existing = true\n")
	writeTestFile(t, root, ".codex/agents/domain.toml", generatedMarker+"\n`src/missing-codex.ts`\n")
	writeTestFile(t, root, ".agents/skills/domain/SKILL.md", generatedMarker+"\n`src/missing-codex-skill.ts`\n")
	writeTestFile(t, root, ".claude/agents/domain.md", generatedMarker+"\n`src/missing-claude.ts`\n")
	writeTestFile(t, root, ".claude/skills/domain/SKILL.md", generatedMarker+"\n`src/missing-claude-skill.ts`\n")

	report, err := Drift(root, SurfaceIndex{})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		".codex/agents/domain.toml",
		".agents/skills/domain/SKILL.md",
		".claude/agents/domain.md",
		".claude/skills/domain/SKILL.md",
	} {
		if !hasStaleAnchorFile(report, want) {
			t.Fatalf("stale anchors = %#v, want anchor from %s", report.StaleAnchors, want)
		}
	}
}

func hasStaleAnchorFile(report DriftReport, file string) bool {
	for _, anchor := range report.StaleAnchors {
		if anchor.File == file {
			return true
		}
	}
	return false
}
