package aitools

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNumberedListUsesDecimalNumbersPastNine(t *testing.T) {
	items := []string{
		"one", "two", "three", "four", "five",
		"six", "seven", "eight", "nine", "ten", "eleven",
	}

	text := numberedList(items)

	if !strings.Contains(text, "10. ten") || !strings.Contains(text, "11. eleven") {
		t.Fatalf("numbered list = %q, want decimal numbering past 9", text)
	}
	if strings.Contains(text, ":. ten") {
		t.Fatalf("numbered list = %q, contains rune-based numbering artifact", text)
	}
}

func TestSafeJSONReturnsErrorInsteadOfPanicking(t *testing.T) {
	_, err := safeJSON(map[string]any{"bad": math.Inf(1)})
	if err == nil {
		t.Fatal("safeJSON returned nil error for unsupported value")
	}
}

func TestGeneratedAgentAndSkillStartWithFrontmatter(t *testing.T) {
	repo := RepoProfile{
		Kind:    "node-api",
		Runtime: "TypeScript/NestJS",
		Domain:  "checkout",
		KeyFiles: []string{
			"src/checkout/checkout.controller.ts",
		},
	}
	agent := agentMarkdown(AgentProfile{
		Name:        "checkout-contract-agent",
		Description: "Maintains checkout contracts",
	}, repo)
	if !strings.HasPrefix(agent, "---\n") {
		t.Fatalf("agent markdown must start with frontmatter:\n%s", agent[:min(len(agent), 80)])
	}
	if strings.HasPrefix(agent, generatedMarker) {
		t.Fatalf("agent markdown starts with generated marker before frontmatter")
	}
	if !strings.Contains(agent, "\n"+generatedMarker+"\n\n") {
		t.Fatalf("agent markdown missing generated marker after frontmatter")
	}
	for _, want := range []string{"edit: allow", `"*": allow`, `"ls *": allow`, `"ls": allow`, `"pwd": allow`} {
		if !strings.Contains(agent, want) {
			t.Fatalf("agent markdown missing permission %q:\n%s", want, agent)
		}
	}
	for _, want := range []string{"context-discipline", "participantRationale", "filesRead", "filesChanged", "lastResult", "rejectedPaths", "nextVerification", "verificationResults", "blockers"} {
		if !strings.Contains(agent, want) {
			t.Fatalf("agent markdown missing checkpoint contract %q:\n%s", want, agent)
		}
	}
	for _, unwanted := range []string{`"npm run test*": allow`, `"pnpm run test*": allow`, `"yarn test*": allow`, `"bun test*": allow`} {
		if strings.Contains(agent, unwanted) {
			t.Fatalf("agent markdown should rely on wildcard bash allow, found package-manager permission %q:\n%s", unwanted, agent)
		}
	}

	skill := skillMarkdown(SkillProfile{
		Name:        "checkout-contract-surface",
		Description: "Work on checkout contracts",
	}, repo)
	if !strings.HasPrefix(skill, "---\n") {
		t.Fatalf("skill markdown must start with frontmatter:\n%s", skill[:min(len(skill), 80)])
	}
	if strings.HasPrefix(skill, generatedMarker) {
		t.Fatalf("skill markdown starts with generated marker before frontmatter")
	}
	if !strings.Contains(skill, "\n"+generatedMarker+"\n\n") {
		t.Fatalf("skill markdown missing generated marker after frontmatter")
	}
	for _, want := range []string{"filesRead", "filesChanged", "lastResult", "rejectedPaths", "nextVerification", "verificationResults", "blockers"} {
		if !strings.Contains(skill, want) {
			t.Fatalf("skill markdown missing checkpoint contract %q:\n%s", want, skill)
		}
	}
}

func TestMaterializeProfileForceRemovesOnlyStaleGeneratedMetadata(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".opencode/agents/old-generated.md", generatedMarker+"\nold\n")
	writeTestFile(t, root, ".opencode/agents/manual.md", "manual agent\n")
	writeTestFile(t, root, ".opencode/skills/old-generated/SKILL.md", generatedMarker+"\nold\n")
	writeTestFile(t, root, ".opencode/skills/manual/SKILL.md", "manual skill\n")

	profile := Profile{Repos: []RepoProfile{{
		Kind: "node-api",
		Agents: []AgentProfile{{
			Name:        "new-agent",
			Description: "New generated agent",
		}},
		CustomSkills: []SkillProfile{{
			Name:        "new-skill",
			Description: "New generated skill",
		}},
	}}}

	if err := MaterializeProfile(root, profile, true); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		".opencode/agents/old-generated.md",
		".opencode/skills/old-generated/SKILL.md",
	} {
		if Exists(filepath.Join(root, path)) {
			t.Fatalf("stale generated metadata still exists: %s", path)
		}
	}
	for _, path := range []string{
		".opencode/agents/manual.md",
		".opencode/skills/manual/SKILL.md",
		".opencode/agents/new-agent.md",
		".opencode/skills/new-skill/SKILL.md",
	} {
		if !Exists(filepath.Join(root, path)) {
			t.Fatalf("expected metadata to exist: %s", path)
		}
	}
}

func TestMaterializeProfileForceDoesNotOverwriteManualNameConflicts(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".opencode/agents/new-agent.md", "manual agent\n")
	writeTestFile(t, root, ".opencode/skills/new-skill/SKILL.md", "manual skill\n")

	profile := Profile{Repos: []RepoProfile{{
		Kind: "node-api",
		Agents: []AgentProfile{{
			Name:        "new-agent",
			Description: "Generated agent with same name",
		}},
		CustomSkills: []SkillProfile{{
			Name:        "new-skill",
			Description: "Generated skill with same name",
		}},
	}}}

	if err := MaterializeProfile(root, profile, true); err != nil {
		t.Fatal(err)
	}
	agent, err := os.ReadFile(filepath.Join(root, ".opencode/agents/new-agent.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(agent) != "manual agent\n" {
		t.Fatalf("manual agent was overwritten:\n%s", string(agent))
	}
	skill, err := os.ReadFile(filepath.Join(root, ".opencode/skills/new-skill/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(skill) != "manual skill\n" {
		t.Fatalf("manual skill was overwritten:\n%s", string(skill))
	}
}

func TestMaterializeProfileForRuntimesForceDoesNotOverwriteManualRuntimeNameConflicts(t *testing.T) {
	root := t.TempDir()
	manualFiles := map[string]string{
		".opencode/agents/new-agent.md":       "manual opencode agent\n",
		".opencode/skills/new-skill/SKILL.md": "manual opencode skill\n",
		".codex/agents/new-agent.toml":        "manual codex agent\n",
		".agents/skills/new-skill/SKILL.md":   "manual codex skill\n",
		".claude/agents/new-agent.md":         "manual claude agent\n",
		".claude/skills/new-skill/SKILL.md":   "manual claude skill\n",
	}
	for path, content := range manualFiles {
		writeTestFile(t, root, path, content)
	}

	profile := Profile{Repos: []RepoProfile{{
		Kind: "node-api",
		Agents: []AgentProfile{{
			Name:        "new-agent",
			Description: "Generated agent with same name",
		}},
		CustomSkills: []SkillProfile{{
			Name:        "new-skill",
			Description: "Generated skill with same name",
		}},
	}}}

	if err := MaterializeProfileForRuntimes(root, profile, true, []string{RuntimeOpenCode, RuntimeCodex, RuntimeClaude}); err != nil {
		t.Fatal(err)
	}
	for path, want := range manualFiles {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != want {
			t.Fatalf("manual runtime metadata %s was overwritten:\n%s", path, string(data))
		}
	}
}
