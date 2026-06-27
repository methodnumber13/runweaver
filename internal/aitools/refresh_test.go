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
    "test": "vitest run"
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
    "test": "vitest run"
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
    "test": "vitest run"
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

func TestRefreshApplyMaterializesNestJSBFFSemanticAgentsAndSkills(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{
  "scripts": {
    "build": "nest build",
    "lint": "eslint \"{src,test}/**/*.ts\"",
    "test": "jest --config ./test/jest-unit.json --runInBand",
    "test:e2e": "jest --config ./test/jest-e2e.json --runInBand"
  },
  "dependencies": {
    "@nestjs/common": "^10.0.0",
    "@nestjs/core": "^10.0.0",
    "@prisma/client": "6.0.0",
    "class-validator": "0.14.0",
    "prisma": "6.0.0"
  },
  "devDependencies": {
    "eslint": "^8.0.0",
    "jest": "29.0.0",
    "prettier": "^3.0.0",
    "typescript": "^5.0.0"
  }
}`)
	writeTestFile(t, root, "src/main.ts", "NestFactory.create(AppModule)\napp.listen(3000)\n")
	writeTestFile(t, root, "src/app.module.ts", "@Module({ imports: [] })\nexport class AppModule {}\n")
	writeTestFile(t, root, "src/auth/auth.controller.ts", "@Controller('auth')\nexport class AuthController {}\n")
	writeTestFile(t, root, "src/auth/identity.service.ts", "export class IdentityProviderService {}\n")
	writeTestFile(t, root, "src/scm/scm.service.ts", "export class SourceControlService {}\n")
	writeTestFile(t, root, "src/kubernetes/kubernetes.controller.ts", "@Controller('kubernetes')\nexport class KubernetesController {}\n")
	writeTestFile(t, root, "src/templates/templates.service.ts", "export class TemplatesService {}\n")
	writeTestFile(t, root, "src/templates/templates.repository.ts", "export class TemplatesRepository {}\n")
	writeTestFile(t, root, "src/object-storage/object-storage.service.ts", "export class ObjectStorageService {}\n")
	writeTestFile(t, root, "prisma/schema.prisma", "model TemplateTrigger { id Int @id }\n")
	writeTestFile(t, root, "test/jest-unit.json", "{}\n")
	writeTestFile(t, root, "test/jest-e2e.json", "{}\n")
	writeTestFile(t, root, ".env.example", "PORT=3000\n")

	result, err := Refresh(root, true)
	if err != nil {
		t.Fatal(err)
	}

	if result.ProfilePath != ".opencode/swarm/profile.json" {
		t.Fatalf("profile path = %q, want applied profile", result.ProfilePath)
	}
	for _, name := range []string{
		".opencode/agents/nestjs-route-contract-engineer.md",
		".opencode/agents/nestjs-config-boundary-reviewer.md",
		".opencode/agents/prisma-persistence-reviewer.md",
		".opencode/agents/identity-access-agent.md",
		".opencode/agents/scm-integration-agent.md",
		".opencode/agents/object-storage-agent.md",
		".opencode/skills/nestjs-bootstrap-config/SKILL.md",
		".opencode/skills/identity-auth-surface/SKILL.md",
		".opencode/skills/templates-prisma-surface/SKILL.md",
		".opencode/skills/object-storage-surface/SKILL.md",
	} {
		if !Exists(filepath.Join(root, name)) {
			t.Fatalf("expected generated metadata %s to exist", name)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, ".opencode/agents/identity-access-agent.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "src/auth/identity.service.ts") || strings.Contains(string(data), "src/scm/scm.service.ts") {
		t.Fatalf("auth agent focus is not domain-specific:\n%s", string(data))
	}
}

func TestInitDoesNotOverwriteWithoutForceAndDoesNotAddMCPConfig(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "AGENTS.md", "custom rules\n")

	if err := Init(root, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "custom rules\n" {
		t.Fatalf("AGENTS.md was overwritten without force: %q", string(data))
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
	if string(data) == "custom rules\n" {
		t.Fatal("AGENTS.md was not overwritten with force")
	}
}
