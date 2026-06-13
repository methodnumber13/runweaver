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
    "lint": "eslint .",
    "test": "vitest run",
    "build": "vite build"
  },
  "dependencies": {
    "@reduxjs/toolkit": "latest",
    "axios": "latest",
    "express": "latest",
    "react": "latest",
    "zod": "latest"
  },
  "devDependencies": {
    "eslint": "latest",
    "prettier": "latest",
    "typescript": "latest",
    "vitest": "latest"
  }
}`)
	writeTestFile(t, root, "src/routes/orders.ts", "import { z } from 'zod'\nexport function register(router) { router.get('/orders', () => {}) }\n")
	writeTestFile(t, root, "src/components/Orders.tsx", "import axios from 'axios'\nexport function Orders() { return null }\n")
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
	if !hasPackageRole(first, "frontend-framework") || !hasPackageRole(first, "api-framework") || !hasPackageRole(first, "state-management") || !hasPackageRole(first, "api-client") || !hasPackageRole(first, "validation") {
		t.Fatalf("package roles = %#v, want framework/state/client/validation roles", first.Packages)
	}
	if !containsString(first.Tools.TestTools, "vitest") || !containsString(first.Tools.Linters, "eslint") || !containsString(first.Tools.Formatters, "prettier") {
		t.Fatalf("tools = %#v, want vitest/eslint/prettier", first.Tools)
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
