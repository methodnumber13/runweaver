package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/methodnumber13/runweaver/internal/aitools"
)

func TestCLIContextQueryReturnsScopedContext(t *testing.T) {
	root := t.TempDir()
	index := aitools.RepoIndex{
		SchemaVersion: 1,
		GeneratedAt:   aitools.Now(),
		RepoRoot:      root,
		Files: []aitools.FileInventoryItem{
			{Path: "src/auth/auth.guard.ts", Category: "source", Language: "typescript"},
			{Path: "test/unit/auth.guard.spec.ts", Category: "test", Language: "typescript"},
		},
		Symbols: []aitools.SymbolInfo{{Kind: "class", Name: "AuthGuard", Path: "src/auth/auth.guard.ts"}},
		Edges:   []aitools.IndexEdge{{From: "test/unit/auth.guard.spec.ts", To: "src/auth/auth.guard.ts", Kind: "tests"}},
	}
	if err := aitools.WriteJSON(filepath.Join(root, ".runweaver", "tmp", "index", "repo-index.json"), index); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI([]string{
		"context", "query",
		"--repo", root,
		"--task", "Fix auth guard test",
		"--limit", "3",
		"--json",
	}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("context query exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"src/auth/auth.guard.ts"`) ||
		!strings.Contains(stdout.String(), `"test/unit/auth.guard.spec.ts"`) ||
		!strings.Contains(stdout.String(), `"AuthGuard"`) {
		t.Fatalf("context query stdout = %q, want scoped auth context", stdout.String())
	}
}
