package aitools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRuntimeSelectionSupportsAllAndStableOrder(t *testing.T) {
	providers, err := ResolveRuntimeSelection("codex,opencode,codex")
	if err != nil {
		t.Fatal(err)
	}
	if got := runtimeProviderIDs(providers); len(got) != 2 || got[0] != RuntimeOpenCode || got[1] != RuntimeCodex {
		t.Fatalf("providers = %#v, want opencode,codex", got)
	}

	providers, err = ResolveRuntimeSelection("all")
	if err != nil {
		t.Fatal(err)
	}
	if got := runtimeProviderIDs(providers); len(got) != 3 || got[0] != RuntimeOpenCode || got[1] != RuntimeCodex || got[2] != RuntimeClaude {
		t.Fatalf("all providers = %#v, want opencode,codex,claude", got)
	}

	if _, err := ResolveRuntimeSelection("missing-runtime"); err == nil {
		t.Fatal("ResolveRuntimeSelection accepted unsupported runtime")
	}
}

func TestDiscoverRuntimesFindsProjectAndGlobalMetadata(t *testing.T) {
	root := t.TempDir()
	codexHome := filepath.Join(root, "home", "codex")
	claudeHome := filepath.Join(root, "home", "claude")
	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("CLAUDE_CONFIG_DIR", claudeHome)
	writeTestFile(t, root, ".codex/config.toml", "[features]\nmulti_agent = true\n")
	writeTestFile(t, root, ".codex/agents/swarm.toml", "name = \"swarm\"\n")
	writeTestFile(t, root, ".agents/skills/repo-onboarding/SKILL.md", "---\nname: repo-onboarding\n---\n")
	writeTestFile(t, codexHome, "config.toml", "model = \"gpt-5.4\"\n")
	writeTestFile(t, codexHome, "auth.json", "{}\n")
	writeTestFile(t, root, ".claude/settings.json", "{}\n")
	writeTestFile(t, root, ".claude/agents/swarm.md", "---\nname: swarm\n---\n")
	writeTestFile(t, claudeHome, "settings.json", "{}\n")
	writeTestFile(t, claudeHome, "auth.json", "{}\n")

	results, err := DiscoverRuntimes(root, RuntimeDiscoveryOptions{Runtime: "codex,claude"})
	if err != nil {
		t.Fatal(err)
	}
	codex := runtimeDiscoveryByID(results, RuntimeCodex)
	claude := runtimeDiscoveryByID(results, RuntimeClaude)
	if codex == nil || claude == nil {
		t.Fatalf("results = %#v, want codex and claude", results)
	}
	if !runtimeCheckReadable(codex.ConfigFiles, filepath.Join(root, ".codex", "config.toml")) {
		t.Fatalf("codex config files = %#v, want project config readable", codex.ConfigFiles)
	}
	if !runtimeCheckReadable(codex.ConfigFiles, filepath.Join(codexHome, "config.toml")) {
		t.Fatalf("codex config files = %#v, want CODEX_HOME config readable", codex.ConfigFiles)
	}
	if !runtimeCheckReadable(codex.MetadataFiles, filepath.Join(root, ".agents", "skills")) {
		t.Fatalf("codex metadata files = %#v, want project skills readable", codex.MetadataFiles)
	}
	if !runtimeCheckReadable(claude.ConfigFiles, filepath.Join(root, ".claude", "settings.json")) {
		t.Fatalf("claude config files = %#v, want project settings readable", claude.ConfigFiles)
	}
	if !runtimeCheckReadable(claude.ConfigFiles, filepath.Join(claudeHome, "settings.json")) {
		t.Fatalf("claude config files = %#v, want CLAUDE_CONFIG_DIR settings readable", claude.ConfigFiles)
	}
}

func TestMaterializeProfileForCodexAndClaude(t *testing.T) {
	root := t.TempDir()
	profile := Profile{Repos: []RepoProfile{{
		Kind:     "node-api",
		Runtime:  "TypeScript/NestJS",
		Domain:   "checkout",
		KeyFiles: []string{"src/checkout/checkout.controller.ts"},
		Agents: []AgentProfile{{
			Name:        "checkout-agent",
			Description: "Owns checkout flows",
			FocusFiles:  []string{"src/checkout/checkout.controller.ts"},
			Workflow:    []string{"Read checkout contracts", "Patch scoped code"},
		}},
		CustomSkills: []SkillProfile{{
			Name:         "checkout-skill",
			Description:  "Checkout procedure",
			FocusFiles:   []string{"src/checkout/checkout.controller.ts"},
			Workflow:     []string{"Inspect DTOs"},
			Verification: []string{"npm test"},
		}},
	}}}

	if err := MaterializeProfileForRuntimes(root, profile, true, []string{RuntimeCodex, RuntimeClaude}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		".codex/agents/checkout-agent.toml",
		".agents/skills/checkout-skill/SKILL.md",
		".claude/agents/checkout-agent.md",
		".claude/skills/checkout-skill/SKILL.md",
	} {
		if !Exists(filepath.Join(root, path)) {
			t.Fatalf("expected generated runtime metadata: %s", path)
		}
	}
	if Exists(filepath.Join(root, ".opencode/agents/checkout-agent.md")) {
		t.Fatal("codex+claude materialization should not write OpenCode agents")
	}
	assertFileContains(t, filepath.Join(root, ".codex/agents/checkout-agent.toml"), generatedMarker)
	assertFileContains(t, filepath.Join(root, ".codex/agents/checkout-agent.toml"), "developer_instructions")
	assertFileContains(t, filepath.Join(root, ".agents/skills/checkout-skill/SKILL.md"), "compatibility: codex")
	assertFileContains(t, filepath.Join(root, ".claude/agents/checkout-agent.md"), "name: checkout-agent")
	assertFileContains(t, filepath.Join(root, ".claude/skills/checkout-skill/SKILL.md"), "compatibility: claude")
}

func TestClassifyApplyMaterializesSelectedRuntimes(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{"scripts":{"test":"jest"},"dependencies":{"@nestjs/core":"10.0.0"},"devDependencies":{"jest":"29.0.0","typescript":"5.0.0"}}`)
	writeTestFile(t, root, "src/app.controller.ts", "export class AppController {}\n")

	result, err := ClassifyRepository(root, ClassifyOptions{
		Mode:         ClassificationDeterministic,
		Apply:        true,
		ApplyRuntime: "codex,claude",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Applied || len(result.ProfilePaths) != 2 {
		t.Fatalf("result = %#v, want applied codex+claude profile paths", result)
	}
	for _, path := range []string{
		".codex/runweaver/profile.json",
		".claude/runweaver/profile.json",
	} {
		if !Exists(filepath.Join(root, path)) {
			t.Fatalf("expected classify --apply artifact: %s", path)
		}
	}
	codexAgents, err := filepath.Glob(filepath.Join(root, ".codex/agents/*.toml"))
	if err != nil {
		t.Fatal(err)
	}
	claudeAgents, err := filepath.Glob(filepath.Join(root, ".claude/agents/*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(codexAgents) == 0 || len(claudeAgents) == 0 {
		t.Fatalf("codexAgents=%#v claudeAgents=%#v, want repo-specific agents", codexAgents, claudeAgents)
	}
	if Exists(filepath.Join(root, ".opencode/swarm/profile.json")) {
		t.Fatal("codex+claude classify apply should not write OpenCode profile")
	}
}

func TestInitSmartCanBootstrapCodexRuntimeWithoutOpenCodeMetadata(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{"scripts":{"test":"jest"},"dependencies":{"@nestjs/core":"10.0.0"},"devDependencies":{"jest":"29.0.0","typescript":"5.0.0"}}`)
	writeTestFile(t, root, "src/app.controller.ts", "export class AppController {}\n")

	result, err := InitSmartWithOptions(root, InitOptions{
		Force:          true,
		Runtime:        RuntimeCodex,
		Classification: ClassifyOptions{Mode: ClassificationDeterministic},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Runtime != RuntimeCodex {
		t.Fatalf("runtime = %q, want codex", result.Runtime)
	}
	if result.ModelPreflight.Status != "skipped" {
		t.Fatalf("model preflight status = %q, want skipped", result.ModelPreflight.Status)
	}
	for _, path := range []string{
		"AGENTS.md",
		".codex/agents/swarm.toml",
		".codex/runweaver/profile.json",
		".agents/skills/context-discipline/SKILL.md",
		".runweaver/workflows/feature-delivery-swarm.json",
	} {
		if !Exists(filepath.Join(root, path)) {
			t.Fatalf("expected init artifact: %s", path)
		}
	}
	for _, path := range []string{
		"opencode.json",
		".opencode/agents/swarm.md",
	} {
		if Exists(filepath.Join(root, path)) {
			t.Fatalf("unexpected OpenCode runtime artifact for codex-only init: %s", path)
		}
	}
}

func runtimeDiscoveryByID(results []RuntimeDiscoveryResult, id string) *RuntimeDiscoveryResult {
	for i := range results {
		if results[i].ID == id {
			return &results[i]
		}
	}
	return nil
}

func runtimeCheckReadable(checks []RuntimeFileCheck, path string) bool {
	for _, check := range checks {
		if filepath.Clean(check.Path) == filepath.Clean(path) && check.Readable {
			return true
		}
	}
	return false
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s = %q, want %q", path, string(data), want)
	}
}
