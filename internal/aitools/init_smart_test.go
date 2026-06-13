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
