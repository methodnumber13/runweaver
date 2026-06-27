package aitools

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParseRepoClassificationUsesFirstBalancedJSONObject(t *testing.T) {
	classification, err := parseRepoClassification([]byte(`noise
{"summary":"ok","domains":[{"name":"checkout","files":["src/checkout/checkout.controller.ts"]}],"agents":[{"name":"checkout-agent"}],"skills":[{"name":"checkout-surface","focusFiles":["src/checkout/checkout.controller.ts"]}]}
debug {"not":"part of classification"}`))
	if err != nil {
		t.Fatal(err)
	}
	if classification.Summary != "ok" || !classificationHasDomain(classification, "checkout") {
		t.Fatalf("classification = %#v, want first balanced object", classification)
	}
}

func TestNormalizeClassifyOptionsDefaultTimeoutIs180Seconds(t *testing.T) {
	opts, err := normalizeClassifyOptions(ClassifyOptions{Mode: ClassificationAI}, ClassificationAI)
	if err != nil {
		t.Fatal(err)
	}
	if opts.Timeout != 180*time.Second {
		t.Fatalf("timeout = %s, want 180s", opts.Timeout)
	}
}

func TestAutoClassifierFallsBackWhenModelOutputIsInvalid(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	runner := func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		return []byte("not-json"), nil
	}

	index, err := indexWithOptions(root, IndexOptions{
		ChangedOnly: true,
		Classification: ClassifyOptions{
			Mode:           ClassificationAuto,
			SkipModelCheck: true,
			OpencodeBin:    "opencode-test",
		},
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if index.ClassifierRun == nil || !index.ClassifierRun.UsedFallback {
		t.Fatalf("classifier run = %#v, want fallback", index.ClassifierRun)
	}
	if index.Classification.Source != "deterministic-semantic-fallback" {
		t.Fatalf("source = %q, want deterministic fallback", index.Classification.Source)
	}
}

func TestAIClassifierFailsWhenModelOutputIsInvalid(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	runner := func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		return []byte("not-json"), nil
	}

	_, err := indexWithOptions(root, IndexOptions{
		ChangedOnly: true,
		Classification: ClassifyOptions{
			Mode:           ClassificationAI,
			SkipModelCheck: true,
			OpencodeBin:    "opencode-test",
		},
	}, runner)
	if err == nil || !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("err = %v, want invalid JSON error", err)
	}
}

func TestValidateModelClassificationRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	index, err := Index(root, true)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = ValidateModelClassification(index, RepoClassification{
		Summary: "unsafe",
		Domains: []DomainClassification{{
			Name:  "checkout",
			Files: []string{"../secret.txt", "/abs/path.ts", "src/checkout/checkout.service.ts"},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "unsafe classified file paths") {
		t.Fatalf("err = %v, want unsafe path validation error", err)
	}
}

func TestValidateModelClassificationExpandsSafeDirectoryPaths(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	writeTestFile(t, root, "src/auth/dto/identity-user.dto.ts", "export class IdentityProviderUserDto {}\n")
	writeTestFile(t, root, "src/auth/dto/auth-groups.dto.ts", "export class AuthGroupsDto {}\n")
	writeTestFile(t, root, "test/mocks/identity.mock.ts", "export const identityMock = {}\n")
	index, err := Index(root, true)
	if err != nil {
		t.Fatal(err)
	}

	classification, warnings, err := ValidateModelClassification(index, RepoClassification{
		Summary: "directory paths",
		Domains: []DomainClassification{{
			Name:  "auth",
			Files: []string{"src/auth/dto/", "test/mocks/"},
		}},
		Agents: []AgentProfile{{
			Name:       "auth-agent",
			FocusFiles: []string{"src/auth/dto/"},
		}},
		Skills: []SkillProfile{{
			Name:       "auth-surface",
			FocusFiles: []string{"test/mocks/"},
		}},
	})
	if err != nil {
		t.Fatalf("err = %v, want directory paths expanded", err)
	}
	files := classification.Domains[0].Files
	for _, want := range []string{"src/auth/dto/identity-user.dto.ts", "src/auth/dto/auth-groups.dto.ts", "test/mocks/identity.mock.ts"} {
		if !containsString(files, want) {
			t.Fatalf("domain files = %#v, want expanded %s", files, want)
		}
	}
	if !containsWarning(warnings, "expanded classified directory") {
		t.Fatalf("warnings = %#v, want directory expansion warning", warnings)
	}
}

func TestValidateModelClassificationDropsGeneratedMetadataPaths(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	index, err := Index(root, true)
	if err != nil {
		t.Fatal(err)
	}

	classification, warnings, err := ValidateModelClassification(index, RepoClassification{
		Summary: "metadata paths",
		Agents: []AgentProfile{{
			Name:       "checkout-contract-agent",
			FocusFiles: []string{".opencode/", ".opencode/swarm/profile.json", "src/checkout/checkout.controller.ts"},
		}},
		Skills: []SkillProfile{{
			Name:       "checkout-contract-surface",
			FocusFiles: []string{"src/checkout/checkout.service.ts"},
		}},
	})
	if err != nil {
		t.Fatalf("err = %v, want generated metadata paths dropped without fatal validation", err)
	}
	if containsString(classification.Agents[0].FocusFiles, ".opencode/") || containsString(classification.Agents[0].FocusFiles, ".opencode/swarm/profile.json") {
		t.Fatalf("agent focusFiles = %#v, want generated metadata paths dropped", classification.Agents[0].FocusFiles)
	}
	if !containsString(warnings, "dropped generated metadata path from classification: .opencode/") {
		t.Fatalf("warnings = %#v, want generated metadata warning", warnings)
	}
}

func TestValidateModelClassificationDropsReservedBaselineNames(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	index, err := Index(root, true)
	if err != nil {
		t.Fatal(err)
	}

	classification, _, err := ValidateModelClassification(index, RepoClassification{
		Summary: "reserved names",
		Agents: []AgentProfile{
			{Name: "repo-surface-indexer", FocusFiles: []string{"src/main.ts"}},
			{Name: "checkout-contract-agent", FocusFiles: []string{"src/checkout/checkout.controller.ts"}},
		},
		Skills: []SkillProfile{
			{Name: "metadata-refresh", FocusFiles: []string{"src/main.ts"}},
			{Name: "checkout-contract-surface", FocusFiles: []string{"src/checkout/checkout.service.ts"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if profileHasAgent(RepoProfile{Agents: classification.Agents}, "repo-surface-indexer") {
		t.Fatalf("agents = %#v, want reserved baseline agent dropped", classification.Agents)
	}
	if profileHasSkill(RepoProfile{CustomSkills: classification.Skills}, "metadata-refresh") {
		t.Fatalf("skills = %#v, want reserved baseline skill dropped", classification.Skills)
	}
	if !profileHasAgent(RepoProfile{Agents: classification.Agents}, "checkout-contract-agent") || !profileHasSkill(RepoProfile{CustomSkills: classification.Skills}, "checkout-contract-surface") {
		t.Fatalf("classification = %#v, want non-reserved profiles kept", classification)
	}
}
