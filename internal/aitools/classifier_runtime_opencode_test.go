package aitools

import (
	"context"
	"github.com/methodnumber13/runweaver/internal/aitools/runtimeenv"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexUsesAIClassifierWhenJSONIsValid(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	var capturedArgs []string
	var capturedEnv []string
	runner := func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		if dir != root {
			t.Fatalf("dir = %q, want root", dir)
		}
		if name != "opencode-test" {
			t.Fatalf("name = %q, want opencode-test", name)
		}
		capturedArgs = append([]string{}, args...)
		capturedEnv = append([]string{}, env...)
		writeTestFile(t, root, ".opencode/package.json", `{"dependencies":{"@opencode-ai/plugin":"1.15.13"}}`)
		writeTestFile(t, root, ".opencode/package-lock.json", `{"name":".opencode"}`)
		writeTestFile(t, root, ".opencode/node_modules/@opencode-ai/plugin/package.json", `{"name":"@opencode-ai/plugin"}`)
		return []byte(`{
  "summary": "AI-classified NestJS checkout BFF surface.",
  "domains": [
    {
      "name": "checkout",
      "description": "Owns checkout controller, DTO, service orchestration, and tests",
      "kind": "domain",
      "files": ["src/checkout/checkout.controller.ts", "src/checkout/checkout.service.ts", "src/missing.ts"],
      "evidence": ["src/checkout/*"],
      "confidence": "high"
    }
  ],
  "agents": [
    {
      "name": "checkout-contract-agent",
      "description": "Maintains checkout API contracts and service orchestration",
      "focusFiles": ["src/checkout/checkout.controller.ts", "src/checkout/checkout.service.ts"],
      "workflow": ["Trace controller decorators before service changes"],
      "verification": ["npm test"]
    }
  ],
  "skills": [
    {
      "name": "checkout-contract-surface",
      "description": "Reusable checkout contract update procedure",
      "focusFiles": ["src/checkout/checkout.controller.ts", "src/checkout/checkout.service.ts"],
      "workflow": ["Read DTOs, controller, service, and tests together"],
      "risks": ["External checkout contract drift"],
      "verification": ["npm test"]
    }
  ],
  "verification": ["npm test"]
}`), nil
	}

	index, err := indexWithOptions(root, IndexOptions{
		ChangedOnly: true,
		Classification: ClassifyOptions{
			Mode:           ClassificationAI,
			SkipModelCheck: true,
			OpencodeBin:    "opencode-test",
			Agent:          "repo-classifier",
			Model:          "openai-compatible/coder-model",
		},
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if index.Classification.Source != "model-backed-opencode" {
		t.Fatalf("classification source = %q, want model-backed-opencode", index.Classification.Source)
	}
	if index.ClassifierRun == nil || !index.ClassifierRun.ModelUsed || index.ClassifierRun.UsedFallback {
		t.Fatalf("classifier run = %#v, want model used without fallback", index.ClassifierRun)
	}
	if !classificationHasDomain(index.Classification, "checkout") {
		t.Fatalf("domains = %#v, want checkout", index.Classification.Domains)
	}
	if len(index.Classification.Domains) != 1 {
		t.Fatalf("domains = %#v, want only AI-provided checkout domain", index.Classification.Domains)
	}
	files := index.Classification.Domains[0].Files
	if containsString(files, "src/missing.ts") {
		t.Fatalf("domain files = %#v, want missing file dropped", files)
	}
	profile := GenerateProfileFromIndex(index)
	if len(profile.Repos) != 1 {
		t.Fatalf("profile repos = %#v, want one repo", profile.Repos)
	}
	repo := profile.Repos[0]
	if len(repo.Agents) != 2 || repo.Agents[0].Name != "checkout-domain-agent" || !profileHasAgent(repo, "checkout-contract-agent") {
		t.Fatalf("profile agents = %#v, want AI domain agent before AI-provided checkout support agent", repo.Agents)
	}
	if profileHasAgent(repo, "api-route-engineer") || profileHasAgent(repo, "validation-contract-reviewer") || profileHasAgent(repo, "nestjs-route-contract-engineer") {
		t.Fatalf("profile agents = %#v, want no deterministic/package agent merge", repo.Agents)
	}
	if len(repo.CustomSkills) != 1 || !profileHasSkill(repo, "checkout-contract-surface") {
		t.Fatalf("profile skills = %#v, want only AI-provided checkout skill", repo.CustomSkills)
	}
	if profileHasSkill(repo, "repo-test-tooling") || profileHasSkill(repo, "repo-quality-gates") || profileHasSkill(repo, "nestjs-test-harness") {
		t.Fatalf("profile skills = %#v, want no deterministic/package skill merge", repo.CustomSkills)
	}
	args := strings.Join(capturedArgs, " ")
	for _, want := range []string{"run", "--pure", "--agent repo-classifier", "--model openai-compatible/coder-model"} {
		if !strings.Contains(args, want) {
			t.Fatalf("args = %q, want %q", args, want)
		}
	}
	for _, path := range []string{".opencode/package.json", ".opencode/package-lock.json", ".opencode/node_modules"} {
		if Exists(filepath.Join(root, path)) {
			t.Fatalf("OpenCode dependency artifact %s was not cleaned", path)
		}
	}
	if !strings.Contains(runtimeenv.EnvValue(capturedEnv, "NO_PROXY"), "llm-provider.example.com") {
		t.Fatalf("NO_PROXY = %q, want OpenAI-compatible host", runtimeenv.EnvValue(capturedEnv, "NO_PROXY"))
	}
	if !strings.Contains(runtimeenv.EnvValue(capturedEnv, "no_proxy"), "llm-provider.example.com") {
		t.Fatalf("no_proxy = %q, want OpenAI-compatible host", runtimeenv.EnvValue(capturedEnv, "no_proxy"))
	}
}

func TestAIClassifierPreservesPreexistingOpenCodeDependencyArtifacts(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	writeTestFile(t, root, ".opencode/package.json", `{"name":"user-owned-opencode-package"}`)
	writeTestFile(t, root, ".opencode/node_modules/user-owned/package.json", `{"name":"user-owned"}`)
	runner := func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		writeTestFile(t, root, ".opencode/package-lock.json", `{"name":"created-by-opencode"}`)
		return []byte(`{
  "summary": "AI-classified checkout",
  "domains": [{"name":"checkout","description":"Checkout API","files":["src/checkout/checkout.controller.ts","src/checkout/checkout.service.ts","src/main.ts"]}],
  "agents": [{"name":"checkout-agent","description":"Maintains checkout API","focusFiles":["src/checkout/checkout.controller.ts","src/main.ts"],"workflow":["Trace controller"],"verification":["npm test"]}],
  "skills": [{"name":"checkout-surface","description":"Checkout workflow","focusFiles":["src/checkout/checkout.controller.ts","src/main.ts","package.json","opencode.json","test/jest-unit.json"],"workflow":["Read tests"],"verification":["npm test"]}],
  "verification": ["npm test"]
}`), nil
	}

	if _, err := indexWithOptions(root, IndexOptions{
		ChangedOnly: true,
		Classification: ClassifyOptions{
			Mode:           ClassificationAI,
			SkipModelCheck: true,
			OpencodeBin:    "opencode-test",
		},
	}, runner); err != nil {
		t.Fatal(err)
	}

	if !Exists(filepath.Join(root, ".opencode/package.json")) {
		t.Fatal("pre-existing .opencode/package.json was removed")
	}
	if !Exists(filepath.Join(root, ".opencode/node_modules/user-owned/package.json")) {
		t.Fatal("pre-existing .opencode/node_modules content was removed")
	}
	if Exists(filepath.Join(root, ".opencode/package-lock.json")) {
		t.Fatal("newly-created .opencode/package-lock.json was not cleaned")
	}
}
