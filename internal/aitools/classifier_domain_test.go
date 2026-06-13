package aitools

import (
	"context"
	"testing"
)

func TestAIClassifierNormalizesBFFDomainsBeforeLayerAgents(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	writeTestFile(t, root, "src/auth/auth.controller.ts", "@Controller('auth')\nexport class AuthController {}\n")
	writeTestFile(t, root, "src/auth/identity.service.ts", "export class IdentityProviderService {}\n")
	writeTestFile(t, root, "src/scm/scm.controller.ts", "@Controller('scm')\nexport class SourceControlController {}\n")
	writeTestFile(t, root, "src/scm/scm.service.ts", "export class SourceControlService {}\n")
	runner := func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		return []byte(`{
  "summary": "AI-classified NestJS API domain surface.",
  "domains": [
    {"name":"auth","description":"Owns auth guard and IdentityProvider contracts","kind":"auth","files":["src/auth/auth.controller.ts","src/auth/identity.service.ts","src/main.ts"],"confidence":"high"},
    {"name":"scm","description":"Owns source control adapter contracts","kind":"external-integration","files":["src/scm/scm.controller.ts","src/scm/scm.service.ts"],"confidence":"high"}
  ],
  "agents": [
    {"name":"api-controller-agent","description":"Reviews NestJS controllers","focusFiles":["src/auth/auth.controller.ts","src/scm/scm.controller.ts","src/main.ts"],"workflow":["Check decorators"],"verification":["npm test"]}
  ],
  "skills": [
    {"name":"nestjs-test-surface","description":"Uses repo Jest configs","focusFiles":["test/jest-unit.json","src/main.ts","src/checkout/checkout.controller.ts"],"workflow":["Run focused tests"],"verification":["npm test"]}
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
		},
	}, runner)
	if err != nil {
		t.Fatal(err)
	}
	profile := GenerateProfileFromIndex(index)
	repo := profile.Repos[0]
	if len(repo.Agents) < 3 {
		t.Fatalf("profile agents = %#v, want domain agents plus secondary layer agent", repo.Agents)
	}
	if repo.Agents[0].Name != "identity-access-agent" || repo.Agents[1].Name != "scm-integration-agent" {
		t.Fatalf("profile agents = %#v, want auth and scm domain agents first", repo.Agents)
	}
	if !profileHasAgent(repo, "api-controller-agent") {
		t.Fatalf("profile agents = %#v, want AI layer agent retained as secondary", repo.Agents)
	}
	if profileHasAgent(repo, "nestjs-route-contract-engineer") {
		t.Fatalf("profile agents = %#v, want no deterministic NestJS agent in AI mode", repo.Agents)
	}
}

func TestNormalizeDomainFirstClassificationDropsDirectDomainDuplicates(t *testing.T) {
	classification := normalizeDomainFirstClassification(RepoIndex{
		Tools: ToolchainInfo{RecommendedCommands: []string{"npm test"}},
	}, RepoClassification{
		Domains: []DomainClassification{{
			Name:        "prisma",
			Description: "Owns Prisma persistence",
			Kind:        "persistence",
			Files:       []string{"prisma/schema.prisma"},
		}},
		Agents: []AgentProfile{
			{Name: "prisma-domain-agent", FocusFiles: []string{"prisma/schema.prisma"}},
			{Name: "prisma-contract-agent", FocusFiles: []string{"prisma/schema.prisma"}},
		},
		Skills: []SkillProfile{{Name: "prisma-surface", FocusFiles: []string{"prisma/schema.prisma"}}},
	})
	repo := RepoProfile{Agents: classification.Agents}
	if !profileHasAgent(repo, "prisma-persistence-agent") {
		t.Fatalf("agents = %#v, want normalized prisma-persistence-agent", classification.Agents)
	}
	if profileHasAgent(repo, "prisma-domain-agent") {
		t.Fatalf("agents = %#v, want duplicate prisma-domain-agent dropped", classification.Agents)
	}
	if !profileHasAgent(repo, "prisma-contract-agent") {
		t.Fatalf("agents = %#v, want secondary contract agent retained", classification.Agents)
	}
}
