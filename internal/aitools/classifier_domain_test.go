package aitools

import (
	"context"
	"testing"
)

func TestAIClassifierNormalizesAPIDomainsBeforeLayerAgents(t *testing.T) {
	root := t.TempDir()
	writeClassifierFixture(t, root)
	writeTestFile(t, root, "src/auth/auth.controller.ts", "@Controller('auth')\nexport class AuthController {}\n")
	writeTestFile(t, root, "src/auth/identity.service.ts", "export class IdentityProviderService {}\n")
	writeTestFile(t, root, "src/orders/orders.controller.ts", "@Controller('orders')\nexport class OrdersController {}\n")
	writeTestFile(t, root, "src/orders/orders.service.ts", "export class OrdersService {}\n")
	runner := func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
		return []byte(`{
  "summary": "AI-classified API domain surface.",
  "domains": [
    {"name":"auth","description":"Owns auth guard and IdentityProvider contracts","kind":"auth","files":["src/auth/auth.controller.ts","src/auth/identity.service.ts","src/main.ts"],"confidence":"high"},
    {"name":"orders","description":"Owns order workflow contracts","kind":"domain","files":["src/orders/orders.controller.ts","src/orders/orders.service.ts"],"confidence":"high"}
  ],
  "agents": [
    {"name":"api-controller-agent","description":"Reviews API controllers","focusFiles":["src/auth/auth.controller.ts","src/orders/orders.controller.ts","src/main.ts"],"workflow":["Check decorators"],"verification":["npm test"]}
  ],
  "skills": [
    {"name":"api-test-surface","description":"Uses repo test configs","focusFiles":["test/unit.json","src/main.ts","src/checkout/checkout.controller.ts"],"workflow":["Run focused tests"],"verification":["npm test"]}
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
	if repo.Agents[0].Name != "auth-security-agent" || repo.Agents[1].Name != "orders-domain-agent" {
		t.Fatalf("profile agents = %#v, want auth and orders domain agents first", repo.Agents)
	}
	if !profileHasAgent(repo, "api-controller-agent") {
		t.Fatalf("profile agents = %#v, want AI layer agent retained as secondary", repo.Agents)
	}
	if profileHasAgent(repo, "api-route-contract-engineer") {
		t.Fatalf("profile agents = %#v, want no deterministic API route agent in AI mode", repo.Agents)
	}
}

func TestNormalizeDomainFirstClassificationDropsDirectDomainDuplicates(t *testing.T) {
	classification := normalizeDomainFirstClassification(RepoIndex{
		Tools: ToolchainInfo{RecommendedCommands: []string{"npm test"}},
	}, RepoClassification{
		Domains: []DomainClassification{{
			Name:        "ledger",
			Description: "Owns ledger persistence",
			Kind:        "persistence",
			Files:       []string{"db/migrations/001_init.sql"},
		}},
		Agents: []AgentProfile{
			{Name: "ledger-domain-agent", FocusFiles: []string{"db/migrations/001_init.sql"}},
			{Name: "ledger-contract-agent", FocusFiles: []string{"db/migrations/001_init.sql"}},
		},
		Skills: []SkillProfile{{Name: "ledger-surface", FocusFiles: []string{"db/migrations/001_init.sql"}}},
	})
	repo := RepoProfile{Agents: classification.Agents}
	if !profileHasAgent(repo, "ledger-persistence-agent") {
		t.Fatalf("agents = %#v, want normalized ledger-persistence-agent", classification.Agents)
	}
	if profileHasAgent(repo, "ledger-domain-agent") {
		t.Fatalf("agents = %#v, want duplicate ledger-domain-agent dropped", classification.Agents)
	}
	if !profileHasAgent(repo, "ledger-contract-agent") {
		t.Fatalf("agents = %#v, want secondary contract agent retained", classification.Agents)
	}
}
