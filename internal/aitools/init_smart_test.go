package aitools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitSmartIndexesPlansIntelligenceWorkflowAndMaterializesByPackages(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{
  "scripts": {
    "test": "vitest run"
  },
  "dependencies": {
    "@reduxjs/toolkit": "latest",
    "axios": "latest",
    "react": "latest"
  },
  "devDependencies": {
    "vitest": "latest"
  }
}`)
	writeTestFile(t, root, "src/components/App.tsx", "export function App() { return null }\n")

	result, err := InitSmart(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.IndexPath != ".runweaver/tmp/index/repo-index.json" {
		t.Fatalf("index path = %q, want repo index", result.IndexPath)
	}
	if result.IntelligenceRun.Workflow != "repo-intelligence-swarm" {
		t.Fatalf("workflow = %q, want repo-intelligence-swarm", result.IntelligenceRun.Workflow)
	}
	for _, path := range []string{
		".runweaver/workflows/repo-intelligence-swarm.json",
		".runweaver/tmp/index/repo-index.json",
		".opencode/swarm/profile.json",
		".opencode/agents/swarm.md",
		".opencode/skills/context-discipline/SKILL.md",
		".opencode/agents/state-flow-reviewer.md",
		".opencode/agents/api-client-contract-reviewer.md",
		".opencode/skills/repo-test-tooling/SKILL.md",
		".opencode/skills/state-management-surface/SKILL.md",
	} {
		if !Exists(filepath.Join(root, path)) {
			t.Fatalf("expected smart init artifact %s", path)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, ".opencode/agents/swarm.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "edit: allow") || !strings.Contains(string(data), "task: allow") || !strings.Contains(string(data), "todowrite: allow") {
		t.Fatalf("swarm agent missing edit/task/todowrite permissions:\n%s", string(data))
	}
	for _, want := range []string{
		`"runweaver workflow run *": allow`,
		`"runweaver workflow update *": allow`,
		`"runweaver workflow verify *": allow`,
		`"*": allow`,
		`"ls -la *": allow`,
		`"ls -l *": allow`,
		`"ls -a *": allow`,
		`"ls *": allow`,
		`"ls": allow`,
		`"pwd": allow`,
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("swarm agent missing bash permission %q:\n%s", want, string(data))
		}
	}
	for _, want := range []string{
		"## Shell Safety",
		"`2>&1`",
		"`2>/dev/null`",
		"`||`",
		"fallback `echo`",
		"For optional file or directory discovery, prefer OpenCode",
		"Do not probe optional paths",
		"Planning-only mode is active only when",
		"Do not ask the user to run the resume command manually",
		"Treat matching `customSkills` as participants",
		"context-discipline",
		"participantRationale",
		"filesChanged",
		"workflow verify",
		"--complete-phase",
		"resume is automatic via swarm",
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("swarm agent missing shell safety guidance %q:\n%s", want, string(data))
		}
	}
	if !strings.Contains(string(data), "runweaver workflow run") || !strings.Contains(string(data), "repo-context.md") {
		t.Fatalf("swarm agent missing workflow/index guidance:\n%s", string(data))
	}
	opencodeData, err := os.ReadFile(filepath.Join(root, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"*": "allow"`,
	} {
		if !strings.Contains(string(opencodeData), want) {
			t.Fatalf("opencode.json missing bash permission %q:\n%s", want, string(opencodeData))
		}
	}
	for _, unwanted := range []string{`"npm run test*": "allow"`, `"pnpm run test*": "allow"`, `"yarn test*": "allow"`, `"bun test*": "allow"`} {
		if strings.Contains(string(opencodeData), unwanted) {
			t.Fatalf("opencode.json should rely on wildcard bash allow, found package-manager permission %q:\n%s", unwanted, string(opencodeData))
		}
	}
	if !strings.Contains(string(opencodeData), `"edit": "allow"`) {
		t.Fatalf("opencode.json missing edit allow permission:\n%s", string(opencodeData))
	}
	workflow, err := os.ReadFile(filepath.Join(root, ".runweaver/workflows/feature-delivery-swarm.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"runtime profile", "repo-specific participants", "Use portable agents only when no specific owner matches", `"maxParticipants": 4`, "participant rationale"} {
		if !strings.Contains(string(workflow), want) {
			t.Fatalf("feature workflow missing %q:\n%s", want, string(workflow))
		}
	}
	bugfixWorkflow, err := os.ReadFile(filepath.Join(root, ".runweaver/workflows/bugfix-swarm.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Treat matching customSkills as participants", "security-middleware", "--complete-phase"} {
		if !strings.Contains(string(bugfixWorkflow), want) {
			t.Fatalf("bugfix workflow missing %q:\n%s", want, string(bugfixWorkflow))
		}
	}
	contextSkill, err := os.ReadFile(filepath.Join(root, ".opencode/skills/context-discipline/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"filesystem-backed state", "Locked", "Editable through CLI only", "Append-only", "runweaver workflow verify", "participantRationale"} {
		if !strings.Contains(string(contextSkill), want) {
			t.Fatalf("context-discipline skill missing %q:\n%s", want, string(contextSkill))
		}
	}
}

func TestInitSmartMergesExistingOpenCodeConfigAndKeepsBackup(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{"scripts":{"test":"go test ./..."},"devDependencies":{"typescript":"latest"}}`)
	writeTestFile(t, root, "src/app.ts", "export const app = true\n")
	writeTestFile(t, root, "opencode.json", `{
  "$schema": "https://opencode.ai/config.json",
  "default_agent": "custom-agent",
  "instructions": "CUSTOM.md",
  "plugin": ["custom-plugin"],
  "provider": {
    "company-llm": {
      "npm": "@ai-sdk/openai-compatible",
      "options": {
        "baseURL": "https://llm-provider.example.com/v1",
        "apiKey": "{env:RUNWEAVER_MODEL_API_KEY}"
      }
    }
  },
  "mcp": {
    "custom": {
      "type": "local",
      "command": ["custom-mcp"]
    }
  },
  "permission": {
    "bash": {
      "custom-tool *": "allow"
    }
  },
  "watcher": {
    "ignore": ["custom/**"]
  }
}`)

	_, err := InitSmartWithOptions(root, InitOptions{
		Force:          true,
		Runtime:        RuntimeOpenCode,
		Classification: ClassifyOptions{Mode: ClassificationDeterministic},
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	value := string(data)
	for _, want := range []string{
		`"default_agent": "swarm"`,
		`"CUSTOM.md"`,
		`"AGENTS.md"`,
		`"custom-plugin"`,
		`"company-llm"`,
		`"mcp"`,
		`"custom-tool *": "allow"`,
		`"runweaver *": "allow"`,
		`"custom/**"`,
		`".runweaver/tmp/**"`,
	} {
		if !strings.Contains(value, want) {
			t.Fatalf("merged opencode.json missing %q:\n%s", want, value)
		}
	}
	backup, err := os.ReadFile(filepath.Join(root, "opencode.json.runweaver.bak"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(backup), `"default_agent": "custom-agent"`) {
		t.Fatalf("backup does not contain original config:\n%s", string(backup))
	}
}

func TestInitSmartMergesExistingOpenCodeJSONCInsteadOfCreatingCompetingConfig(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{"scripts":{"test":"go test ./..."},"devDependencies":{"typescript":"latest"}}`)
	writeTestFile(t, root, "src/app.ts", "export const app = true\n")
	writeTestFile(t, root, "opencode.jsonc", `{
  // project model must survive RunWeaver init
  "model": "custom-provider/custom-model",
  "provider": {
    "custom-provider": {
      "name": "Custom Provider"
    }
  }
}`)

	_, err := InitSmartWithOptions(root, InitOptions{
		Force:          true,
		Runtime:        RuntimeOpenCode,
		Classification: ClassifyOptions{Mode: ClassificationDeterministic},
	})
	if err != nil {
		t.Fatal(err)
	}
	if Exists(filepath.Join(root, "opencode.json")) {
		t.Fatal("init created opencode.json even though project opencode.jsonc already existed")
	}
	data, err := os.ReadFile(filepath.Join(root, "opencode.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	value := string(data)
	for _, want := range []string{
		`"default_agent": "swarm"`,
		`"model": "custom-provider/custom-model"`,
		`"custom-provider"`,
		`"runweaver *": "allow"`,
	} {
		if !strings.Contains(value, want) {
			t.Fatalf("merged opencode.jsonc missing %q:\n%s", want, value)
		}
	}
	if !Exists(filepath.Join(root, "opencode.jsonc.runweaver.bak")) {
		t.Fatal("expected opencode.jsonc backup")
	}
}

func TestInitSmartPreservesExistingInstructionFilesWithManagedBlock(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{"scripts":{"test":"go test ./..."}}`)
	writeTestFile(t, root, "src/app.ts", "export const app = true\n")
	writeTestFile(t, root, "AGENTS.md", "# Existing Agents\n\nKeep this rule.\n")
	writeTestFile(t, root, "CLAUDE.md", "# Existing Claude\n\nKeep this claude rule.\n")

	_, err := InitSmartWithOptions(root, InitOptions{
		Force:          true,
		Runtime:        RuntimeAll,
		Classification: ClassifyOptions{Mode: ClassificationDeterministic},
	})
	if err != nil {
		t.Fatal(err)
	}

	agentsData, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	agentsText := string(agentsData)
	for _, want := range []string{"# Existing Agents", "Keep this rule.", "<!-- BEGIN RUNWEAVER -->", "RunWeaver metadata is generated"} {
		if !strings.Contains(agentsText, want) {
			t.Fatalf("AGENTS.md missing %q:\n%s", want, agentsText)
		}
	}

	claudeData, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	claudeText := string(claudeData)
	for _, want := range []string{"# Existing Claude", "Keep this claude rule.", "<!-- BEGIN RUNWEAVER -->", "RunWeaver metadata is generated"} {
		if !strings.Contains(claudeText, want) {
			t.Fatalf("CLAUDE.md missing %q:\n%s", want, claudeText)
		}
	}
}

func TestInitSmartWritesRunWeaverStartupProtocolForAllRuntimes(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{"scripts":{"test":"go test ./..."},"devDependencies":{"typescript":"latest"}}`)
	writeTestFile(t, root, "src/app.ts", "export const app = true\n")

	_, err := InitSmartWithOptions(root, InitOptions{
		Force:          true,
		Runtime:        RuntimeAll,
		Classification: ClassifyOptions{Mode: ClassificationDeterministic},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"AGENTS.md",
		"CLAUDE.md",
		".opencode/agents/swarm.md",
		".codex/agents/swarm.toml",
		".claude/agents/swarm.md",
	} {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, want := range []string{
			"RunWeaver Startup Protocol",
			"runweaver status --repo .",
			"resume automatically",
			"Do not ask the user to run resume",
			"runweaver workflow verify --repo . --resume latest",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing startup protocol fragment %q:\n%s", path, want, text)
			}
		}
	}
}

func TestInitSmartWritesStartHereAndPreservesManualStartHere(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{"scripts":{"test":"go test ./..."},"devDependencies":{"typescript":"latest"}}`)
	writeTestFile(t, root, "src/app.ts", "export const app = true\n")

	_, err := InitSmartWithOptions(root, InitOptions{
		Force:          true,
		Runtime:        RuntimeAll,
		Classification: ClassifyOptions{Mode: ClassificationDeterministic},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".runweaver/START_HERE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"# RunWeaver Start Here",
		"runweaver status --repo .",
		".runweaver/tmp/current.md",
		".runweaver/workflows",
		"AGENTS.md",
		"CLAUDE.md",
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("START_HERE missing %q:\n%s", want, string(data))
		}
	}

	manualRoot := t.TempDir()
	writeTestFile(t, manualRoot, "package.json", `{"scripts":{"test":"go test ./..."},"devDependencies":{"typescript":"latest"}}`)
	writeTestFile(t, manualRoot, "src/app.ts", "export const app = true\n")
	writeTestFile(t, manualRoot, ".runweaver/START_HERE.md", "manual runweaver notes\n")
	_, err = InitSmartWithOptions(manualRoot, InitOptions{
		Force:          true,
		Runtime:        RuntimeAll,
		Classification: ClassifyOptions{Mode: ClassificationDeterministic},
	})
	if err != nil {
		t.Fatal(err)
	}
	manual, err := os.ReadFile(filepath.Join(manualRoot, ".runweaver/START_HERE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(manual) != "manual runweaver notes\n" {
		t.Fatalf("manual START_HERE was overwritten:\n%s", string(manual))
	}
}
