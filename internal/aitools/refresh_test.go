package aitools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRefreshWritesArtifactsAndDetectsStaleAnchors(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{
  "scripts": {
    "test": "mocha"
  },
  "dependencies": {
    "express": "latest"
  }
}`)
	writeTestFile(t, root, "src/index.ts", "export {}\n")
	writeTestFile(t, root, "src/routes/orders.ts", "export {}\n")

	if err := Init(root, false); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, ".opencode/agents/custom.md", "Review `src/old/orders.ts` before editing routes.\n")

	result, err := Refresh(root, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{result.SurfaceIndexPath, result.DriftReportPath, result.ProfilePath} {
		if !Exists(filepath.Join(root, name)) {
			t.Fatalf("expected artifact %s to exist", name)
		}
	}
	if len(result.DriftReport.StaleAnchors) != 1 {
		t.Fatalf("stale anchors = %#v, want one stale anchor", result.DriftReport.StaleAnchors)
	}
	stale := result.DriftReport.StaleAnchors[0]
	if stale.File != ".opencode/agents/custom.md" || stale.Anchor != "src/old/orders.ts" {
		t.Fatalf("stale anchor = %#v, want custom agent old route", stale)
	}
	if result.ProfilePath != ".runweaver/tmp/profile.generated.json" {
		t.Fatalf("profile path = %q, want generated profile", result.ProfilePath)
	}
}

func TestRefreshApplyMaterializesRepoSpecificAgentsAndSkills(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{
  "scripts": {
    "test": "mocha"
  },
  "dependencies": {
    "express": "latest"
  }
}`)
	writeTestFile(t, root, "src/index.ts", "export {}\n")
	writeTestFile(t, root, "src/controllers/orders.ts", "export {}\n")

	result, err := Refresh(root, true)
	if err != nil {
		t.Fatal(err)
	}

	if result.ProfilePath != ".opencode/swarm/profile.json" {
		t.Fatalf("profile path = %q, want applied profile", result.ProfilePath)
	}
	for _, name := range []string{
		".opencode/agents/api-route-engineer.md",
		".opencode/agents/api-security-boundary-reviewer.md",
		".opencode/skills/api-route-surface/SKILL.md",
		".opencode/skills/repo-quality-gates/SKILL.md",
	} {
		if !Exists(filepath.Join(root, name)) {
			t.Fatalf("expected generated metadata %s to exist", name)
		}
	}
}

func TestRefreshApplySelectedRuntimeDoesNotCreateOpenCodeMetadata(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{
  "scripts": {
    "test": "mocha"
  },
  "dependencies": {
    "express": "latest"
  }
}`)
	writeTestFile(t, root, "src/routes/orders.ts", "export {}\n")

	result, err := RefreshWithOptions(root, RefreshOptions{
		Apply:          true,
		Runtime:        RuntimeCodex,
		Classification: ClassifyOptions{Mode: ClassificationDeterministic},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.ProfilePath != ".codex/runweaver/profile.json" {
		t.Fatalf("profile path = %q, want codex profile", result.ProfilePath)
	}
	for _, path := range []string{
		".codex/runweaver/profile.json",
		".codex/agents/api-route-engineer.toml",
		".agents/skills/api-route-surface/SKILL.md",
	} {
		if !Exists(filepath.Join(root, path)) {
			t.Fatalf("expected Codex metadata %s", path)
		}
	}
	for _, path := range []string{
		".opencode",
		"opencode.json",
		".opencode/swarm/profile.json",
	} {
		if Exists(filepath.Join(root, path)) {
			t.Fatalf("unexpected OpenCode metadata for Codex refresh: %s", path)
		}
	}
}

func TestRefreshApplyMaterializesGenericAPISemanticAgentsAndSkills(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{
  "scripts": {
    "build": "repo-build",
    "lint": "repo-lint \"{src,test}/**/*.ts\"",
    "test": "repo-test --unit-config ./test/unit.json",
    "test:e2e": "repo-test --e2e-config ./test/e2e.json"
  },
  "dependencies": {
    "route-decorator-lib": "^1.0.0",
    "schema-validator-lib": "1.0.0"
  },
  "devDependencies": {
    "lint-helper": "^1.0.0",
    "test-runner-lib": "1.0.0",
    "format-helper": "^1.0.0",
    "build-compiler-lib": "^1.0.0"
  }
}`)
	writeTestFile(t, root, "src/main.ts", "NestFactory.create(AppModule)\napp.listen(3000)\n")
	writeTestFile(t, root, "src/app.module.ts", "@Module({ imports: [] })\nexport class AppModule {}\n")
	writeTestFile(t, root, "src/auth/auth.controller.ts", "@Controller('auth')\nexport class AuthController {}\n")
	writeTestFile(t, root, "src/auth/identity.service.ts", "export class IdentityProviderService {}\n")
	writeTestFile(t, root, "src/orders/orders.service.ts", "export class OrdersService {}\n")
	writeTestFile(t, root, "src/inventory/inventory.controller.ts", "@Controller('inventory')\nexport class InventoryController {}\n")
	writeTestFile(t, root, "src/analytics/analytics.service.ts", "export class AnalyticsService {}\n")
	writeTestFile(t, root, "src/analytics/analytics.repository.ts", "export class AnalyticsRepository {}\n")
	writeTestFile(t, root, "src/assets/assets.service.ts", "export class AssetsService {}\n")
	writeTestFile(t, root, "db/migrations/20250101000000_init/migration.sql", "select 1;\n")
	writeTestFile(t, root, "test/unit.json", "{}\n")
	writeTestFile(t, root, "test/e2e.json", "{}\n")
	writeTestFile(t, root, ".env.example", "PORT=3000\n")

	result, err := Refresh(root, true)
	if err != nil {
		t.Fatal(err)
	}

	if result.ProfilePath != ".opencode/swarm/profile.json" {
		t.Fatalf("profile path = %q, want applied profile", result.ProfilePath)
	}
	for _, name := range []string{
		".opencode/agents/api-route-contract-engineer.md",
		".opencode/agents/config-bootstrap-boundary-reviewer.md",
		".opencode/agents/persistence-boundary-reviewer.md",
		".opencode/agents/auth-domain-agent.md",
		".opencode/agents/orders-domain-agent.md",
		".opencode/agents/assets-domain-agent.md",
		".opencode/skills/bootstrap-config-surface/SKILL.md",
		".opencode/skills/auth-domain-surface/SKILL.md",
		".opencode/skills/persistence-surface/SKILL.md",
		".opencode/skills/assets-domain-surface/SKILL.md",
	} {
		if !Exists(filepath.Join(root, name)) {
			t.Fatalf("expected generated metadata %s to exist", name)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, ".opencode/agents/auth-domain-agent.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "src/auth/identity.service.ts") || strings.Contains(string(data), "src/orders/orders.service.ts") {
		t.Fatalf("auth agent focus is not domain-specific:\n%s", string(data))
	}
}

func TestInitMergesExistingInstructionsAndDoesNotAddMCPConfig(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "AGENTS.md", "custom rules\n")

	if err := Init(root, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"custom rules", "<!-- BEGIN RUNWEAVER -->", "RunWeaver metadata is generated"} {
		if !strings.Contains(text, want) {
			t.Fatalf("AGENTS.md missing %q after init merge:\n%s", want, text)
		}
	}
	backup, err := os.ReadFile(filepath.Join(root, "AGENTS.md.runweaver.bak"))
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != "custom rules\n" {
		t.Fatalf("AGENTS.md backup = %q, want original custom rules", string(backup))
	}

	opencodeData, err := os.ReadFile(filepath.Join(root, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(opencodeData)), "mcp") {
		t.Fatalf("opencode.json should not contain MCP config: %s", string(opencodeData))
	}

	if err := Init(root, true); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	text = string(data)
	if !strings.Contains(text, "custom rules") {
		t.Fatalf("AGENTS.md force refresh removed custom rules:\n%s", text)
	}
	if strings.Count(text, "<!-- BEGIN RUNWEAVER -->") != 1 {
		t.Fatalf("AGENTS.md has duplicate RunWeaver managed blocks:\n%s", text)
	}
}
