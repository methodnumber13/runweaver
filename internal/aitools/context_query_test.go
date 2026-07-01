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
			{Path: "src/pages/account-recovery/components/reset-form/hooks.ts", Category: "source", Language: "typescript"},
			{Path: "src/pages/account-recovery/components/reset-form/components/options-block.tsx", Category: "source", Language: "typescript"},
			{Path: "src/pages/account-recovery/components/reset-form/__test__/hooks.test.ts", Category: "test", Language: "typescript"},
		},
		Symbols: []SymbolInfo{
			{Kind: "function", Name: "useAccountRecoveryForm", Path: "src/pages/account-recovery/components/reset-form/hooks.ts", Line: 12},
		},
		Edges: []IndexEdge{
			{From: "src/pages/account-recovery/components/reset-form/__test__/hooks.test.ts", To: "src/pages/account-recovery/components/reset-form/hooks.ts", Kind: "tests", Reason: "matched nearby source filename"},
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
	if got := result.Files[0].Path; !strings.HasPrefix(got, "src/pages/account-recovery/") {
		t.Fatalf("top file = %q, want account-recovery form context before generic tests; files=%#v", got, result.Files)
	}
	if !contextFileSelected(result.Files, "src/pages/account-recovery/components/reset-form/hooks.ts") {
		t.Fatalf("files = %#v, want account-recovery form source", result.Files)
	}
	if !contextFileSelected(result.Files, "src/pages/account-recovery/components/reset-form/__test__/hooks.test.ts") {
		t.Fatalf("files = %#v, want related domain test still present", result.Files)
	}
	if contextEdgeSelected(result.Tests, "src/common/hooks/__test__/use-effect-once.test.ts") {
		t.Fatalf("tests = %#v, want unrelated generic test edge filtered out", result.Tests)
	}
	if !contextSymbolSelected(result.Symbols, "useAccountRecoveryForm") {
		t.Fatalf("symbols = %#v, want reset password form symbol", result.Symbols)
	}
}

func TestQueryContextRanksNamedComponentsBeforeTaskDocs(t *testing.T) {
	root := t.TempDir()
	index := RepoIndex{
		SchemaVersion: 1,
		GeneratedAt:   Now(),
		RepoRoot:      root,
		Tools: ToolchainInfo{
			RecommendedCommands: []string{"pnpm test", "pnpm typecheck"},
		},
		Files: []FileInventoryItem{
			{Path: "docs/tasks/workflow-launcher-plan.md", Category: "documentation", Language: "markdown"},
			{Path: "docs/tasks/workflow-launch-config-dialog-notes.md", Category: "documentation", Language: "markdown"},
			{Path: "src/api/routes/workflow-launch-config.ts", Category: "route", Language: "typescript"},
			{Path: "server/workflows/workflow-launcher-api.ts", Category: "source", Language: "typescript"},
			{Path: "src/workflows/FeatureLauncherService.ts", Category: "service", Language: "typescript"},
			{Path: "src/components/workflows/FeatureLauncherCard.tsx", Category: "ui", Language: "typescript"},
			{Path: "src/components/workflows/FeatureLaunchConfigDialog.tsx", Category: "ui", Language: "typescript"},
			{Path: "src/components/workflows/FeatureLauncherCard.test.tsx", Category: "test", Language: "typescript"},
		},
		Symbols: []SymbolInfo{
			{Kind: "function", Name: "FeatureLauncherCard", Path: "src/components/workflows/FeatureLauncherCard.tsx", Line: 10},
			{Kind: "function", Name: "FeatureLaunchConfigDialog", Path: "src/components/workflows/FeatureLaunchConfigDialog.tsx", Line: 22},
			{Kind: "function", Name: "FeatureLauncherService", Path: "src/workflows/FeatureLauncherService.ts", Line: 8},
		},
		Edges: []IndexEdge{
			{From: "src/components/workflows/FeatureLauncherCard.test.tsx", To: "src/components/workflows/FeatureLauncherCard.tsx", Kind: "tests", Reason: "matched nearby source filename"},
		},
	}
	if err := WriteJSON(filepath.Join(root, ".runweaver", "tmp", "index", "repo-index.json"), index); err != nil {
		t.Fatal(err)
	}

	result, err := QueryContext(root, ContextQueryOptions{
		Task:  "Implement three frontend variants for FeatureLauncherCard and FeatureLaunchConfigDialog without backend changes",
		Limit: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) < 2 {
		t.Fatalf("files = %#v, want named component files", result.Files)
	}
	if result.Files[0].Path != "src/components/workflows/FeatureLaunchConfigDialog.tsx" &&
		result.Files[0].Path != "src/components/workflows/FeatureLauncherCard.tsx" {
		t.Fatalf("top file = %q, want named source component before docs; files=%#v", result.Files[0].Path, result.Files)
	}
	if !contextFileSelected(result.Files[:2], "src/components/workflows/FeatureLauncherCard.tsx") ||
		!contextFileSelected(result.Files[:2], "src/components/workflows/FeatureLaunchConfigDialog.tsx") {
		t.Fatalf("top files = %#v, want both named components before docs/tasks", result.Files[:2])
	}
	if strings.HasPrefix(result.Files[0].Path, "docs/tasks/") || strings.HasPrefix(result.Files[1].Path, "docs/tasks/") {
		t.Fatalf("files = %#v, want docs/tasks below named implementation files", result.Files)
	}
	for _, file := range result.Files {
		if strings.HasPrefix(file.Path, "src/api/") || strings.HasPrefix(file.Path, "server/") || strings.Contains(file.Path, "Service") {
			t.Fatalf("files = %#v, want backend files excluded from frontend-only top context", result.Files)
		}
	}
}

func TestQueryContextSuppressesNegatedBackendAndEnvFilesForFrontendBatchTask(t *testing.T) {
	root := t.TempDir()
	index := RepoIndex{
		SchemaVersion: 1,
		GeneratedAt:   Now(),
		RepoRoot:      root,
		Tools: ToolchainInfo{
			RecommendedCommands: []string{"npm --prefix frontend test", "cd mcp/sample-workspace-api && npm run test", "cd mobile-shell && npm run build", "npm run test"},
		},
		Files: []FileInventoryItem{
			{Path: "docs/tasks/batch-actions-launch-plan.md", Category: "documentation", Language: "markdown"},
			{Path: "mcp/sample-workspace-api/.env.example", Category: "config", Language: "dotenv"},
			{Path: "mcp/sample-workspace-api/src/config/env.ts", Category: "config", Language: "typescript"},
			{Path: "mcp/sample-workspace-api/src/tools/result.ts", Category: "source", Language: "typescript"},
			{Path: "backend/src/api/routes.ts", Category: "route", Language: "typescript"},
			{Path: "src/api/routes/taskLaunches.ts", Category: "route", Language: "typescript"},
			{Path: "src/execution/server.ts", Category: "source", Language: "typescript"},
			{Path: "src/browser-worker/worker.ts", Category: "source", Language: "typescript"},
			{Path: "frontend/src/components/TaskLaunches.tsx", Category: "ui", Language: "typescript"},
			{Path: "frontend/src/components/TaskLaunchSettingsDialog.tsx", Category: "ui", Language: "typescript"},
			{Path: "frontend/src/components/TaskRuntimeNotice.tsx", Category: "ui", Language: "typescript"},
			{Path: "frontend/src/components/TaskLaunches.test.tsx", Category: "test", Language: "typescript"},
			{Path: "frontend/src/components/TaskLaunchSettingsDialog.test.tsx", Category: "test", Language: "typescript"},
			{Path: "tdc-solvers/captcha-solve/requirements.txt", Category: "config", Language: "text"},
			{Path: "akamai-solvers/random.ts", Category: "source", Language: "typescript"},
		},
		Symbols: []SymbolInfo{
			{Kind: "function", Name: "TaskLaunches", Path: "frontend/src/components/TaskLaunches.tsx", Line: 10},
			{Kind: "function", Name: "TaskLaunchSettingsDialog", Path: "frontend/src/components/TaskLaunchSettingsDialog.tsx", Line: 20},
			{Kind: "function", Name: "loadEnv", Path: "mcp/sample-workspace-api/src/config/env.ts", Line: 3},
			{Kind: "function", Name: "randIntRange", Path: "akamai-solvers/random.ts", Line: 7},
		},
		Edges: []IndexEdge{
			{From: "frontend/src/components/TaskLaunches.test.tsx", To: "frontend/src/components/TaskLaunches.tsx", Kind: "tests", Reason: "matched nearby source filename"},
			{From: "frontend/src/components/TaskLaunchSettingsDialog.test.tsx", To: "frontend/src/components/TaskLaunchSettingsDialog.tsx", Kind: "tests", Reason: "matched nearby source filename"},
			{From: "src/api/routes/taskLaunches.ts", To: "POST /api/batch-actions", Kind: "declares-route"},
		},
	}
	if err := WriteJSON(filepath.Join(root, ".runweaver", "tmp", "index", "repo-index.json"), index); err != nil {
		t.Fatal(err)
	}

	result, err := QueryContext(root, ContextQueryOptions{
		Task:  "Implement a frontend-only Batch Actions launch readiness improvement. Work only in the frontend Batch Actions surface unless tests prove a shared frontend helper is needed. Do not modify backend/API/database/runtime worker files. Add a compact launch readiness summary and make stale async/template/config loads safe.",
		Limit: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !contextFileSelected(result.Files[:minInt(len(result.Files), 2)], "frontend/src/components/TaskLaunches.tsx") ||
		!contextFileSelected(result.Files[:minInt(len(result.Files), 2)], "frontend/src/components/TaskLaunchSettingsDialog.tsx") {
		t.Fatalf("top files = %#v, want frontend TaskLaunches and TaskLaunchSettingsDialog before backend/env files", result.Files)
	}
	if !containsString(result.Commands, "npm --prefix frontend test") {
		t.Fatalf("commands = %#v, want frontend verification command retained", result.Commands)
	}
	if containsString(result.Tokens, "requirements") || containsString(result.Tokens, "requirement") {
		t.Fatalf("tokens = %#v, want low-signal requirement wording filtered", result.Tokens)
	}
	for _, suppressed := range []string{"api", "database", "runtime", "server", "worker"} {
		if containsString(result.Tokens, suppressed) {
			t.Fatalf("tokens = %#v, want negated target token %q omitted from reported context tokens", result.Tokens, suppressed)
		}
	}
	for _, file := range result.Files {
		if strings.HasPrefix(file.Path, "mcp/sample-workspace-api/") ||
			strings.HasPrefix(file.Path, "backend/") ||
			strings.HasPrefix(file.Path, "src/api/") ||
			strings.HasSuffix(file.Path, "/server.ts") ||
			strings.HasSuffix(file.Path, "/worker.ts") ||
			strings.Contains(file.Path, ".env") ||
			strings.Contains(file.Path, "requirements.txt") {
			t.Fatalf("files = %#v, want backend/API/env/requirements files suppressed for frontend-only task", result.Files)
		}
	}
	if contextSymbolSelected(result.Symbols, "randIntRange") {
		t.Fatalf("symbols = %#v, want utility symbols outside selected context suppressed", result.Symbols)
	}
	for _, command := range result.Commands {
		if strings.Contains(command, "mcp/sample-workspace-api") || strings.Contains(command, "cd mobile-shell") {
			t.Fatalf("commands = %#v, want non-frontend verification commands suppressed for frontend-only task", result.Commands)
		}
	}
}

func TestQueryContextPrefersFrontendSpecificCommandsOverBroadRootCommands(t *testing.T) {
	root := t.TempDir()
	index := RepoIndex{
		SchemaVersion: 1,
		GeneratedAt:   Now(),
		RepoRoot:      root,
		Tools: ToolchainInfo{
			RecommendedCommands: []string{
				"cd frontend && npm run lint",
				"cd frontend && npm run build",
				"cd frontend && npm run test",
				"yarn run lint",
				"yarn run build",
				"yarn run test",
				"cd mcp/sample-workspace-api && npm run test",
				"cd mobile-shell && npm run build",
			},
		},
		Files: []FileInventoryItem{
			{Path: "frontend/src/components/TaskLaunches.tsx", Category: "ui", Language: "typescript"},
			{Path: "frontend/src/components/TaskLaunchSettingsDialog.tsx", Category: "ui", Language: "typescript"},
			{Path: "frontend/src/components/TaskLaunches.test.tsx", Category: "test", Language: "typescript"},
		},
	}
	if err := WriteJSON(filepath.Join(root, ".runweaver", "tmp", "index", "repo-index.json"), index); err != nil {
		t.Fatal(err)
	}

	result, err := QueryContext(root, ContextQueryOptions{
		Task:  "Implement a frontend-only Batch Actions readiness panel. Run lint, test, and build. Do not modify backend/API/mobile shell files.",
		Limit: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"cd frontend && npm run lint", "cd frontend && npm run build", "cd frontend && npm run test"} {
		if !containsString(result.Commands, want) {
			t.Fatalf("commands = %#v, want frontend command %q", result.Commands, want)
		}
	}
	for _, unwanted := range []string{"yarn run lint", "yarn run build", "yarn run test", "mcp/sample-workspace-api", "cd mobile-shell"} {
		for _, command := range result.Commands {
			if strings.Contains(command, unwanted) {
				t.Fatalf("commands = %#v, want no broad/non-frontend command containing %q", result.Commands, unwanted)
			}
		}
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
