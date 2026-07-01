package aitools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexWritesRepoLocalArtifactsAndReusesContentHashCache(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{
  "scripts": {
    "lint": "repo-lint .",
    "test": "repo-test",
    "build": "repo-build"
  },
  "dependencies": {
    "state-store-lib": "latest",
    "http-client-lib": "latest",
    "schema-validator-lib": "latest"
  },
  "devDependencies": {
    "lint-helper": "latest",
    "format-helper": "latest",
    "test-runner-lib": "latest",
    "build-compiler-lib": "latest"
  }
}`)
	writeTestFile(t, root, "src/routes/orders.ts", "import { schema } from 'schema-validator-lib'\nexport function register(router) { router.get('/orders', () => schema) }\n")
	writeTestFile(t, root, "src/components/Orders.tsx", "import client from 'http-client-lib'\nexport function Orders() { return client }\n")
	writeTestFile(t, root, "src/routes/orders.test.ts", "import { register } from './orders'\n")

	first, err := Index(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if first.Artifacts.Root != ".runweaver/tmp" {
		t.Fatalf("artifact root = %q, want .runweaver/tmp", first.Artifacts.Root)
	}
	for _, path := range []string{
		first.Artifacts.Files,
		first.Artifacts.Packages,
		first.Artifacts.Symbols,
		first.Artifacts.Edges,
		first.Artifacts.Index,
		first.Artifacts.Compact,
		first.Artifacts.Context,
		first.Artifacts.Manifest,
	} {
		if !Exists(filepath.Join(root, path)) {
			t.Fatalf("expected index artifact %s", path)
		}
	}
	if first.Stats.CacheEntries == 0 || first.Stats.CacheBytes == 0 {
		t.Fatalf("cache stats = %#v, want cache entries/bytes", first.Stats)
	}
	contextData, err := os.ReadFile(filepath.Join(root, first.Artifacts.Context))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contextData), "Repository Context") || !strings.Contains(string(contextData), "Package Roles") {
		t.Fatalf("repo context is not useful:\n%s", string(contextData))
	}
	for _, role := range []string{"state-management", "api-client", "validation", "test-tool", "linter", "formatter", "build-tool"} {
		if !hasPackageRole(first, role) {
			t.Fatalf("package roles = %#v, want generic role %q", first.Packages, role)
		}
	}
	if !containsString(first.Tools.TestTools, "repository test command") ||
		!containsString(first.Tools.Linters, "repository lint command") ||
		!containsString(first.Tools.Formatters, "package-level format dependency") {
		t.Fatalf("tools = %#v, want generic command/package-derived tooling", first.Tools)
	}
	if len(first.Symbols) == 0 || len(first.Edges) == 0 {
		t.Fatalf("symbols=%d edges=%d, want both", len(first.Symbols), len(first.Edges))
	}

	second, err := Index(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if second.Stats.CacheHits == 0 {
		t.Fatalf("second index cache hits = %d, want > 0", second.Stats.CacheHits)
	}
}

func TestIndexPruneAndCleanKeepRepoLocalTmpBounded(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "src/a.ts", "export function a() { return 1 }\n")

	first, err := IndexWithOptions(root, IndexOptions{ChangedOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(root, ".runweaver/tmp/cache/deadbeef.json")
	writeTestFile(t, root, ".runweaver/tmp/cache/deadbeef.json", `{"hash":"deadbeef"}`)

	second, err := IndexWithOptions(root, IndexOptions{ChangedOnly: true, Prune: true, MaxCacheMB: 256})
	if err != nil {
		t.Fatal(err)
	}
	if Exists(stale) {
		t.Fatalf("stale cache entry still exists at %s", stale)
	}
	if second.Stats.CachePruned == 0 {
		t.Fatalf("cache pruned = %d, want > 0", second.Stats.CachePruned)
	}
	if !Exists(filepath.Join(root, first.Artifacts.Manifest)) {
		t.Fatalf("manifest missing before clean")
	}

	clean, err := CleanIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if !clean.Removed || clean.Path != ".runweaver/tmp" {
		t.Fatalf("clean result = %#v, want removed .runweaver/tmp", clean)
	}
	if Exists(filepath.Join(root, ".runweaver/tmp")) {
		t.Fatalf("index directory still exists after clean")
	}
}
