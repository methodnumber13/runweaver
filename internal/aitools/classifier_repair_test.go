package aitools

import (
	"context"
	"strings"
	"testing"
)

func TestAIClassifierRejectsClassificationWithoutAgentsAndSkills(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	runner := func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		return []byte(`{
  "summary": "AI-classified NestJS checkout BFF surface.",
  "domains": [
    {
      "name": "checkout",
      "description": "Owns checkout controller and service",
      "files": ["src/checkout/checkout.controller.ts", "src/checkout/checkout.service.ts"]
    }
  ]
}`), nil
	}

	_, err := indexWithOptions(root, IndexOptions{
		ChangedOnly: true,
		Classification: ClassifyOptions{
			Mode:           ClassificationAI,
			SkipModelCheck: true,
			OpencodeBin:    "opencode-test",
		},
	}, runner)
	if err == nil || !strings.Contains(err.Error(), "at least one agent and one skill") {
		t.Fatalf("err = %v, want missing agent/skill validation error", err)
	}
}

func TestAIClassifierRepairsMissingMandatoryCoverage(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	calls := 0
	runner := func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		calls++
		if calls == 1 {
			return []byte(`{
  "summary": "AI-classified NestJS checkout BFF surface.",
  "domains": [{"name":"checkout","description":"Checkout API","files":["src/checkout/checkout.controller.ts"]}],
  "agents": [{"name":"checkout-contract-agent","description":"Maintains checkout API","focusFiles":["src/checkout/checkout.controller.ts"],"workflow":["Trace controller"],"verification":["npm test"]}],
  "skills": [{"name":"checkout-contract-surface","description":"Checkout contract procedure","focusFiles":["src/checkout/checkout.controller.ts"],"workflow":["Read DTOs"],"verification":["npm test"]}],
  "verification": ["npm test"]
}`), nil
		}
		return []byte(`{
  "summary": "AI-classified NestJS checkout BFF surface.",
  "domains": [{"name":"checkout","description":"Checkout API","files":["src/checkout/checkout.controller.ts","src/main.ts"]}],
  "agents": [{"name":"checkout-contract-agent","description":"Maintains checkout API","focusFiles":["src/checkout/checkout.controller.ts","src/main.ts"],"workflow":["Trace controller"],"verification":["npm test"]}],
  "skills": [{"name":"checkout-contract-surface","description":"Checkout contract procedure","focusFiles":["src/checkout/checkout.controller.ts","src/main.ts","package.json","opencode.json"],"workflow":["Read DTOs"],"verification":["npm test"]}],
  "verification": ["npm test"]
}`), nil
	}

	index, err := indexWithOptions(root, IndexOptions{
		ChangedOnly: true,
		Classification: ClassifyOptions{
			Mode:           ClassificationAI,
			SkipModelCheck: true,
			OpencodeBin:    "opencode-test",
		},
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("classifier calls = %d, want initial call plus repair", calls)
	}
	if index.ClassifierRun == nil || !strings.Contains(index.ClassifierRun.RawOutputPath, "repo-classifier-repair-output.json") {
		t.Fatalf("classifier run = %#v, want repair raw output path", index.ClassifierRun)
	}
	if !containsString(index.Classification.Agents[0].FocusFiles, "src/main.ts") {
		t.Fatalf("agent focusFiles = %#v, want repaired src/main.ts coverage", index.Classification.Agents[0].FocusFiles)
	}
}

func TestAIClassifierKeepsInitialAIClassificationWhenRepairFails(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	calls := 0
	runner := func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		calls++
		if calls == 1 {
			return []byte(`{
  "summary": "AI-classified NestJS checkout BFF surface.",
  "domains": [{"name":"checkout","description":"Checkout API","files":["src/checkout/checkout.controller.ts"]}],
  "agents": [{"name":"checkout-contract-agent","description":"Maintains checkout API","focusFiles":["src/checkout/checkout.controller.ts"],"workflow":["Trace controller"],"verification":["npm test"]}],
  "skills": [{"name":"checkout-contract-surface","description":"Checkout contract procedure","focusFiles":["src/checkout/checkout.controller.ts"],"workflow":["Read DTOs"],"verification":["npm test"]}],
  "verification": ["npm test"]
}`), nil
		}
		return []byte("{"), nil
	}

	index, err := indexWithOptions(root, IndexOptions{
		ChangedOnly: true,
		Classification: ClassifyOptions{
			Mode:           ClassificationAI,
			SkipModelCheck: true,
			OpencodeBin:    "opencode-test",
		},
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("classifier calls = %d, want initial call plus failed repair", calls)
	}
	if index.Classification.Source != "model-backed-opencode" || index.ClassifierRun == nil || index.ClassifierRun.UsedFallback {
		t.Fatalf("classification=%#v run=%#v, want initial AI classification without deterministic fallback", index.Classification, index.ClassifierRun)
	}
	if !containsWarning(index.ClassifierRun.Warnings, "AI classifier repair failed") {
		t.Fatalf("warnings = %#v, want repair failure warning", index.ClassifierRun.Warnings)
	}
	if !containsWarning(index.Classification.Warnings, "AI classifier repair failed") {
		t.Fatalf("classification warnings = %#v, want repair failure warning persisted", index.Classification.Warnings)
	}
}

func TestAIClassifierRepairsInvalidJSONBeforeValidation(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	calls := 0
	runner := func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		calls++
		if calls == 1 {
			return []byte(`{
  "summary": "AI-classified checkout",
  "domains": [{"name":"checkout","files":["src/checkout/checkout.controller.ts"]}],
  "agents": [{"name":"checkout-agent","focusFiles":["src/checkout/checkout.controller.ts"]}:],
  "skills": [{"name":"checkout-surface","focusFiles":["src/checkout/checkout.controller.ts"]}]
}`), nil
		}
		return []byte(`{
  "summary": "AI-classified checkout",
  "domains": [{"name":"checkout","description":"Checkout API","files":["src/checkout/checkout.controller.ts","src/checkout/checkout.service.ts","src/main.ts"]}],
  "agents": [{"name":"checkout-agent","description":"Maintains checkout API","focusFiles":["src/checkout/checkout.controller.ts","src/main.ts"],"workflow":["Trace controller"],"verification":["npm test"]}],
  "skills": [{"name":"checkout-surface","description":"Checkout workflow","focusFiles":["src/checkout/checkout.controller.ts","src/main.ts","package.json","opencode.json","test/jest-unit.json"],"workflow":["Read tests"],"verification":["npm test"]}],
  "verification": ["npm test"]
}`), nil
	}

	index, err := indexWithOptions(root, IndexOptions{
		ChangedOnly: true,
		Classification: ClassifyOptions{
			Mode:           ClassificationAI,
			SkipModelCheck: true,
			OpencodeBin:    "opencode-test",
		},
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("classifier calls = %d, want invalid JSON repair retry", calls)
	}
	if index.ClassifierRun == nil || !strings.Contains(index.ClassifierRun.RawOutputPath, "repo-classifier-json-repair-output.json") {
		t.Fatalf("classifier run = %#v, want JSON repair raw output path", index.ClassifierRun)
	}
	if !containsWarning(index.ClassifierRun.Warnings, "AI classifier JSON repair recovered invalid model output") {
		t.Fatalf("warnings = %#v, want JSON repair warning", index.ClassifierRun.Warnings)
	}
	if !profileHasAgent(RepoProfile{Agents: index.Classification.Agents}, "checkout-domain-agent") {
		t.Fatalf("agents = %#v, want domain-first agent after JSON repair", index.Classification.Agents)
	}
}
