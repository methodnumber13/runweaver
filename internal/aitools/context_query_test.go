package aitools

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestQueryContextReturnsTaskScopedFilesSymbolsRoutesAndTests(t *testing.T) {
	root := t.TempDir()
	writeContextIndexFixture(t, root)

	result, err := QueryContext(root, ContextQueryOptions{
		Task:  "Fix public route auth guard test",
		Limit: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "success" {
		t.Fatalf("status = %q, want success", result.Status)
	}
	if !contextFileSelected(result.Files, "src/auth/auth.guard.ts") {
		t.Fatalf("files = %#v, want auth guard source", result.Files)
	}
	if !contextFileSelected(result.Files, "test/unit/auth.guard.spec.ts") {
		t.Fatalf("files = %#v, want related auth guard test", result.Files)
	}
	if !contextSymbolSelected(result.Symbols, "AuthGuard") {
		t.Fatalf("symbols = %#v, want AuthGuard", result.Symbols)
	}
	if len(result.Tests) == 0 || result.Tests[0].From != "test/unit/auth.guard.spec.ts" {
		t.Fatalf("tests = %#v, want auth guard test edge", result.Tests)
	}
	if len(result.Routes) == 0 || result.Routes[0].To != "GET /auth/public" {
		t.Fatalf("routes = %#v, want auth public route", result.Routes)
	}
}

func TestQueryContextPrefersDomainTokensOverGenericTestToken(t *testing.T) {
	root := t.TempDir()
	index := RepoIndex{
		SchemaVersion: 1,
		GeneratedAt:   Now(),
		RepoRoot:      root,
		Tools: ToolchainInfo{
			RecommendedCommands: []string{"npm run test", "npm run typecheck"},
		},
		Files: []FileInventoryItem{
			{Path: "src/common/components/__test__/in-browser.test.tsx", Category: "test", Language: "typescript"},
			{Path: "src/common/hooks/__test__/use-effect-once.test.ts", Category: "test", Language: "typescript"},
			{Path: "src/pages/reset-password/components/reset-form/hooks.ts", Category: "source", Language: "typescript"},
			{Path: "src/pages/reset-password/components/reset-form/components/products-block.tsx", Category: "source", Language: "typescript"},
			{Path: "src/pages/reset-password/components/reset-form/__test__/hooks.test.ts", Category: "test", Language: "typescript"},
		},
		Symbols: []SymbolInfo{
			{Kind: "function", Name: "useResetPasswordForm", Path: "src/pages/reset-password/components/reset-form/hooks.ts", Line: 12},
		},
		Edges: []IndexEdge{
			{From: "src/pages/reset-password/components/reset-form/__test__/hooks.test.ts", To: "src/pages/reset-password/components/reset-form/hooks.ts", Kind: "tests", Reason: "matched nearby source filename"},
			{From: "src/common/hooks/__test__/use-effect-once.test.ts", To: "src/common/hooks/use-effect-once.ts", Kind: "tests", Reason: "matched nearby source filename"},
		},
	}
	if err := WriteJSON(filepath.Join(root, ".runweaver", "tmp", "index", "repo-index.json"), index); err != nil {
		t.Fatal(err)
	}

	result, err := QueryContext(root, ContextQueryOptions{
		Task:  "fix reset password form validation test",
		Limit: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) == 0 {
		t.Fatal("files are empty")
	}
	if got := result.Files[0].Path; !strings.HasPrefix(got, "src/pages/reset-password/") {
		t.Fatalf("top file = %q, want reset-password form context before generic tests; files=%#v", got, result.Files)
	}
	if !contextFileSelected(result.Files, "src/pages/reset-password/components/reset-form/hooks.ts") {
		t.Fatalf("files = %#v, want reset-password form source", result.Files)
	}
	if !contextFileSelected(result.Files, "src/pages/reset-password/components/reset-form/__test__/hooks.test.ts") {
		t.Fatalf("files = %#v, want related domain test still present", result.Files)
	}
	if contextEdgeSelected(result.Tests, "src/common/hooks/__test__/use-effect-once.test.ts") {
		t.Fatalf("tests = %#v, want unrelated generic test edge filtered out", result.Tests)
	}
	if !contextSymbolSelected(result.Symbols, "useResetPasswordForm") {
		t.Fatalf("symbols = %#v, want reset password form symbol", result.Symbols)
	}
}

func TestQueryContextDoesNotReturnUnmatchedCommands(t *testing.T) {
	root := t.TempDir()
	index := RepoIndex{
		SchemaVersion: 1,
		GeneratedAt:   Now(),
		RepoRoot:      root,
		Tools: ToolchainInfo{
			RecommendedCommands: []string{"npm run lint", "npm run build"},
		},
		Files: []FileInventoryItem{
			{Path: "src/pricing/pricing.service.ts", Category: "source", Language: "typescript"},
		},
	}
	if err := WriteJSON(filepath.Join(root, ".runweaver", "tmp", "index", "repo-index.json"), index); err != nil {
		t.Fatal(err)
	}

	result, err := QueryContext(root, ContextQueryOptions{
		Task:  "change catalog pricing flow",
		Limit: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Commands) != 0 {
		t.Fatalf("commands = %#v, want no unmatched recommended commands", result.Commands)
	}
	if !stringsContain(result.Explanation, "no recommended command matched task tokens") {
		t.Fatalf("explanation = %#v, want command relevance note", result.Explanation)
	}
}

func writeContextIndexFixture(t *testing.T, root string) {
	t.Helper()
	index := RepoIndex{
		SchemaVersion: 1,
		GeneratedAt:   Now(),
		RepoRoot:      root,
		Tools: ToolchainInfo{
			RecommendedCommands: []string{"npm run test -- auth.guard.spec.ts", "npm run build"},
		},
		Files: []FileInventoryItem{
			{Path: "src/auth/auth.guard.ts", Category: "source", Language: "typescript"},
			{Path: "src/auth/public.decorator.ts", Category: "source", Language: "typescript"},
			{Path: "test/unit/auth.guard.spec.ts", Category: "test", Language: "typescript"},
			{Path: "README.md", Category: "documentation", Language: "markdown"},
		},
		Symbols: []SymbolInfo{
			{Kind: "class", Name: "AuthGuard", Path: "src/auth/auth.guard.ts", Line: 10},
			{Kind: "function", Name: "Public", Path: "src/auth/public.decorator.ts", Line: 3},
		},
		Edges: []IndexEdge{
			{From: "test/unit/auth.guard.spec.ts", To: "src/auth/auth.guard.ts", Kind: "tests", Reason: "matched nearby source filename"},
			{From: "src/auth/auth.guard.ts", To: "GET /auth/public", Kind: "declares-route"},
		},
	}
	if err := WriteJSON(filepath.Join(root, ".runweaver", "tmp", "index", "repo-index.json"), index); err != nil {
		t.Fatal(err)
	}
}

func contextFileSelected(files []ContextFileHit, path string) bool {
	for _, file := range files {
		if file.Path == path {
			return true
		}
	}
	return false
}

func contextSymbolSelected(symbols []SymbolInfo, name string) bool {
	for _, symbol := range symbols {
		if symbol.Name == name {
			return true
		}
	}
	return false
}

func contextEdgeSelected(edges []IndexEdge, from string) bool {
	for _, edge := range edges {
		if edge.From == from {
			return true
		}
	}
	return false
}
