package aitools

import (
	"path/filepath"
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
