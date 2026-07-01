package aitools

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectParticipantsPrefersDomainAgentAndRelevantSkill(t *testing.T) {
	root := t.TempDir()
	writeParticipantSelectionFixtures(t, root)

	result, err := SelectParticipants(root, ParticipantSelectOptions{
		Task:     "Fix auth guard public route regression in src/auth/auth.guard.ts",
		Workflow: ".runweaver/workflows/bugfix-swarm.json",
		Runtime:  RuntimeOpenCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TaskTier != "small" {
		t.Fatalf("task tier = %q, want small", result.TaskTier)
	}
	if result.Cap != 2 {
		t.Fatalf("cap = %d, want tier-aware cap 2", result.Cap)
	}
	if !containsString(result.Participants, "auth-access-agent") {
		t.Fatalf("participants = %#v, want auth-access-agent", result.Participants)
	}
	if !containsString(result.Participants, "security-middleware") {
		t.Fatalf("participants = %#v, want security-middleware skill", result.Participants)
	}
	if len(result.Participants) > result.Cap {
		t.Fatalf("participants = %#v exceed cap %d", result.Participants, result.Cap)
	}
	if len(result.Rationale) == 0 {
		t.Fatalf("rationale is empty: %#v", result)
	}
	if !participantAssignmentHasRole(result.Assignments, "auth-access-agent", "owner") {
		t.Fatalf("assignments = %#v, want auth-access-agent owner role", result.Assignments)
	}
	if !participantAssignmentHasRole(result.Assignments, "security-middleware", "specialist") {
		t.Fatalf("assignments = %#v, want security-middleware specialist role", result.Assignments)
	}
}

func TestSelectParticipantsFallsBackToWorkflowAgents(t *testing.T) {
	root := t.TempDir()
	writeParticipantSelectionFixtures(t, root)

	result, err := SelectParticipants(root, ParticipantSelectOptions{
		Task:     "Update unknown generated metadata",
		Workflow: ".runweaver/workflows/metadata-refresh-swarm.json",
		Runtime:  RuntimeOpenCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(result.Participants, "agent-skill-drift-reviewer") {
		t.Fatalf("participants = %#v, want workflow fallback agent", result.Participants)
	}
	if !stringsContain(result.Warnings, "task context unavailable") {
		t.Fatalf("warnings = %#v, want context degradation warning", result.Warnings)
	}
}

func TestSelectParticipantsUsesTaskScopedContextFocusFiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/bugfix-swarm.json", `{
  "id": "bugfix-swarm",
  "name": "Bugfix Swarm",
  "description": "Fix bugs and regressions",
  "maxParticipants": 1,
  "phases": [
    {"id": "fix", "name": "Fix", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": [], "prompt": "fix bug"}
  ]
}`)
	writeTestFile(t, root, ".opencode/swarm/profile.json", `{
  "workspace": {"name": "api", "repos": ["."]},
  "repos": [{
    "dir": ".",
    "agents": [
      {"name": "aaa-billing-agent", "description": "Primary domain owner", "focusFiles": ["src/billing/billing.service.ts"]},
      {"name": "zzz-auth-agent", "description": "Primary domain owner", "focusFiles": ["src/auth/auth.guard.ts"]}
    ]
  }]
}`)
	writeContextIndexFixture(t, root)

	result, err := SelectParticipants(root, ParticipantSelectOptions{
		Task:     "Fix public route decorator bypass",
		Workflow: ".runweaver/workflows/bugfix-swarm.json",
		Runtime:  RuntimeOpenCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Participants; len(got) != 1 || got[0] != "zzz-auth-agent" {
		t.Fatalf("participants = %#v, want task context to select zzz-auth-agent", got)
	}
	if !stringsContain(result.Rationale, "task context matched focus file: src/auth/auth.guard.ts") {
		t.Fatalf("rationale = %#v, want task-context focus-file evidence", result.Rationale)
	}
}

func TestSelectParticipantsKeepsExecutableAgentWhenSkillsOutscore(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-hardening-swarm.json", `{
  "id": "test-hardening-swarm",
  "name": "Test Hardening Swarm",
  "description": "Improve tests",
  "maxParticipants": 2,
  "phases": [
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["fallback-agent"], "prompt": "verify tests"}
  ]
}`)
	writeTestFile(t, root, ".opencode/swarm/profile.json", `{
  "workspace": {"name": "api", "repos": ["."]},
  "repos": [{
    "dir": ".",
    "customSkills": [
      {"name": "fix-test-skill-a", "description": "Use for fix test work"},
      {"name": "fix-test-skill-b", "description": "Use for fix test verification"}
    ]
  }]
}`)

	result, err := SelectParticipants(root, ParticipantSelectOptions{
		Task:     "fix test",
		Workflow: ".runweaver/workflows/test-hardening-swarm.json",
		Runtime:  RuntimeOpenCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(result.Participants, "fallback-agent") {
		t.Fatalf("participants = %#v, want executable workflow agent even when skills score higher", result.Participants)
	}
	if len(result.Participants) != 2 {
		t.Fatalf("participants = %#v, want cap-sized selection", result.Participants)
	}
}

func TestSelectParticipantsIgnoresSiblingRepoProfiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/bugfix-swarm.json", `{
  "id": "bugfix-swarm",
  "name": "Bugfix Swarm",
  "description": "Fix bugs and regressions",
  "maxParticipants": 1,
  "phases": [
    {"id": "fix", "name": "Fix", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": [], "prompt": "fix bug"}
  ]
}`)
	writeTestFile(t, root, ".opencode/swarm/profile.json", `{
  "workspace": {"name": "workspace", "repos": [".", "sibling-api"]},
  "repos": [
    {
      "dir": ".",
      "agents": [
        {"name": "current-repo-agent", "description": "Current repository owner", "focusFiles": ["src/current/current.go"]}
      ]
    },
    {
      "dir": "sibling-api",
      "agents": [
        {"name": "sibling-auth-agent", "description": "Fix auth route regression public guard token validation", "focusFiles": ["src/auth/auth.guard.ts"]}
      ],
      "customSkills": [
        {"name": "sibling-auth-skill", "description": "Fix auth route regression public guard token validation", "focusFiles": ["src/auth/auth.guard.ts"]}
      ]
    }
  ]
}`)

	result, err := SelectParticipants(root, ParticipantSelectOptions{
		Task:     "Fix auth route regression in public guard token validation",
		Workflow: ".runweaver/workflows/bugfix-swarm.json",
		Runtime:  RuntimeOpenCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Participants; len(got) != 1 || got[0] != "current-repo-agent" {
		t.Fatalf("participants = %#v, want current repo agent only", got)
	}
	if participantCandidateNamed(result.Candidates, "sibling-auth-agent") || participantCandidateNamed(result.Candidates, "sibling-auth-skill") {
		t.Fatalf("candidates = %#v, want sibling repo participants filtered out", result.Candidates)
	}
}

func TestSelectParticipantsDoesNotMatchSiblingWithSameBasename(t *testing.T) {
	workspace := t.TempDir()
	root := filepath.Join(workspace, "services", "api")
	writeTestFile(t, root, ".runweaver/workflows/bugfix-swarm.json", `{
  "id": "bugfix-swarm",
  "name": "Bugfix Swarm",
  "description": "Fix bugs and regressions",
  "maxParticipants": 1,
  "phases": [
    {"id": "fix", "name": "Fix", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": [], "prompt": "fix bug"}
  ]
}`)
	writeTestFile(t, root, ".opencode/swarm/profile.json", `{
  "workspace": {"name": "workspace", "repos": ["services/api", "clients/api"]},
  "repos": [
    {
      "dir": "services/api",
      "agents": [
        {"name": "service-api-agent", "description": "Current service API owner", "focusFiles": ["src/service/api.go"]}
      ]
    },
    {
      "dir": "clients/api",
      "agents": [
        {"name": "client-api-agent", "description": "Fix auth route regression public guard token validation", "focusFiles": ["src/client/auth.go"]}
      ]
    }
  ]
}`)

	result, err := SelectParticipants(root, ParticipantSelectOptions{
		Task:     "Fix auth route regression in public guard token validation",
		Workflow: ".runweaver/workflows/bugfix-swarm.json",
		Runtime:  RuntimeOpenCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Participants; len(got) != 1 || got[0] != "service-api-agent" {
		t.Fatalf("participants = %#v, want service-api-agent only", got)
	}
	if participantCandidateNamed(result.Candidates, "client-api-agent") {
		t.Fatalf("candidates = %#v, want same-basename sibling filtered out", result.Candidates)
	}
}

func TestSelectParticipantsExcludesBackendApiForFrontendOnlyTaskAcrossRuntimes(t *testing.T) {
	for _, runtimeID := range []string{RuntimeOpenCode, RuntimeCodex, RuntimeClaude} {
		t.Run(runtimeID, func(t *testing.T) {
			root := t.TempDir()
			writeTestFile(t, root, ".runweaver/workflows/feature-delivery-swarm.json", `{
  "id": "feature-delivery-swarm",
  "name": "Feature Delivery Swarm",
  "description": "Implement new repository features and product behavior",
  "maxParticipants": 3,
  "phases": [
    {"id": "plan", "name": "Plan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "plan feature"},
    {"id": "implement", "name": "Implement", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["repo-surface-engineer"], "prompt": "implement feature"},
    {"id": "review", "name": "Review", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["post-implementation-reviewer"], "prompt": "review implementation"}
  ]
}`)
			writeParticipantProfileForRuntime(t, root, runtimeID)
			writeFrontendWorkflowContextIndex(t, root)

			result, err := SelectParticipants(root, ParticipantSelectOptions{
				Task:     "Implement frontend FeatureLauncherCard and FeatureLaunchConfigDialog variants. Do not touch backend API routes.",
				Workflow: ".runweaver/workflows/feature-delivery-swarm.json",
				Runtime:  runtimeID,
			})
			if err != nil {
				t.Fatal(err)
			}
			if containsString(result.Participants, "api-route-surface") {
				t.Fatalf("participants = %#v, want api-route-surface excluded for frontend-only task; candidates=%#v", result.Participants, result.Candidates)
			}
			if containsString(result.Participants, "api-test-harness-reviewer") {
				t.Fatalf("participants = %#v, want api-test-harness-reviewer excluded as implementation owner for frontend-only task; candidates=%#v", result.Participants, result.Candidates)
			}
			if containsString(result.Participants, "api-security-boundary-reviewer") {
				t.Fatalf("participants = %#v, want api-security-boundary-reviewer excluded for backend-forbidden frontend task; candidates=%#v", result.Participants, result.Candidates)
			}
			if !containsString(result.Participants, "workflow-launcher-agent") {
				t.Fatalf("participants = %#v, want workflow-launcher-agent selected", result.Participants)
			}
			if !containsString(result.Participants, "post-implementation-reviewer") {
				t.Fatalf("participants = %#v, want post-implementation-reviewer selected for feature implementation; candidates=%#v", result.Participants, result.Candidates)
			}
		})
	}
}

func TestSelectParticipantsPrefersDomainSkillForNamedDomainSurface(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/feature-delivery-swarm.json", `{
  "id": "feature-delivery-swarm",
  "name": "Feature Delivery Swarm",
  "description": "Implement new repository features and product behavior",
  "maxParticipants": 2,
  "phases": [
    {"id": "implement", "name": "Implement", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["repo-surface-engineer"], "prompt": "implement feature"}
  ]
}`)
	writeTestFile(t, root, ".codex/runweaver/profile.json", `{
  "workspace": {"name": "sample-workspace", "repos": ["."]},
  "globalAgents": [
    {"name": "repo-surface-engineer", "description": "Generic implementation fallback"}
  ],
  "repos": [{
    "dir": ".",
    "customSkills": [
      {
        "name": "batch-action-master",
        "description": "Master workflow for Batch Actions behavior, frontend batch action monitoring, action health, retries, and operator summary surfaces.",
        "focusFiles": ["frontend/src/components/TaskLaunches.tsx"]
      },
      {
        "name": "ui-page-surface",
        "description": "Work on frontend UI components, cards, dialogs, page state, and visible user flows.",
        "focusFiles": ["frontend/src/pages/TaskLaunchesPage.tsx"]
      }
    ]
  }]
}`)
	index := RepoIndex{
		SchemaVersion: 1,
		GeneratedAt:   Now(),
		RepoRoot:      root,
		Files: []FileInventoryItem{
			{Path: "frontend/src/pages/TaskLaunchesPage.tsx", Category: "ui", Language: "typescript"},
			{Path: "frontend/src/components/TaskLaunches.tsx", Category: "ui", Language: "typescript"},
		},
		Symbols: []SymbolInfo{
			{Kind: "function", Name: "TaskLaunchesPage", Path: "frontend/src/pages/TaskLaunchesPage.tsx", Line: 10},
		},
	}
	if err := WriteJSON(filepath.Join(root, ".runweaver", "tmp", "index", "repo-index.json"), index); err != nil {
		t.Fatal(err)
	}

	result, err := SelectParticipants(root, ParticipantSelectOptions{
		Task:     "Implement a frontend-only Batch Actions health summary on the Batch Actions page. Do not touch backend/API files.",
		Workflow: ".runweaver/workflows/feature-delivery-swarm.json",
		Runtime:  RuntimeCodex,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !containsString(result.Participants, "batch-action-master") {
		t.Fatalf("participants = %#v, want domain skill batch-action-master; candidates=%#v", result.Participants, result.Candidates)
	}
	if len(result.Participants) > result.Cap {
		t.Fatalf("participants = %#v exceed cap %d", result.Participants, result.Cap)
	}
	if !stringsContain(result.Rationale, "domain-specific participant name matched task domain") {
		t.Fatalf("rationale = %#v, want domain-specific task match rationale", result.Rationale)
	}
}

func TestSelectParticipantsSuppressesNegatedBackendSignalsForFrontendBatchTask(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/feature-delivery-swarm.json", `{
  "id": "feature-delivery-swarm",
  "name": "Feature Delivery Swarm",
  "description": "Implement new repository features and product behavior",
  "maxParticipants": 4,
  "phases": [
    {"id": "plan", "name": "Plan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "plan feature"},
    {"id": "implement", "name": "Implement", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["repo-surface-engineer"], "prompt": "implement feature"},
    {"id": "review", "name": "Review", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["post-implementation-reviewer"], "prompt": "review implementation"}
  ]
}`)
	writeTestFile(t, root, ".codex/runweaver/profile.json", `{
  "workspace": {"name": "sample-workspace", "repos": ["."]},
  "globalAgents": [
    {"name": "repo-surface-indexer", "description": "Scans repository surfaces"},
    {"name": "repo-surface-engineer", "description": "Generic implementation fallback"},
    {"name": "stale-anchor-fixer", "description": "Repairs stale file anchors in generated runtime metadata"},
    {"name": "post-implementation-reviewer", "description": "Reviews completed feature implementations"}
  ],
  "repos": [{
    "dir": ".",
    "agents": [
      {
        "name": "api-route-surface",
        "description": "Work on backend API routes, database handlers, runtime workers, controllers, and endpoint contracts.",
        "focusFiles": ["mcp/sample-workspace-api/src/config/env.ts", "src/api/routes/taskLaunches.ts"]
      }
    ],
    "customSkills": [
      {
        "name": "batch-action-master",
        "description": "Master workflow for Batch Actions behavior, frontend batch action monitoring, action launch settings, and operator readiness summaries.",
        "focusFiles": ["frontend/src/components/TaskLaunches.tsx", "frontend/src/components/TaskLaunchSettingsDialog.tsx"],
        "verification": [
          "cd frontend && npm run test",
          "cd mcp/sample-workspace-api && npm run test",
          "cd mobile-shell && npm run typecheck"
        ]
      },
      {
        "name": "state-management-surface",
        "description": "Work on frontend state management surfaces, dialogs, async loading state, and component data flow.",
        "focusFiles": ["frontend/src/components/TaskLaunchSettingsDialog.tsx", "frontend/src/components/TaskLaunches.tsx"],
        "verification": [
          "cd frontend && npm run test",
          "cd mcp/sample-workspace-api && npm run typecheck",
          "cd mobile-shell && npm run lint"
        ]
      },
      {
        "name": "ui-page-surface",
        "description": "Work on frontend UI components, cards, dialogs, page state, and visible user flows.",
        "focusFiles": ["frontend/src/pages/AccountsPage.tsx", "frontend/src/components/TaskLaunches.tsx"],
        "verification": [
          "cd frontend && npm run test",
          "cd mcp/sample-workspace-api && npm run build",
          "cd mobile-shell && npm run build"
        ]
      }
    ]
  }]
}`)
	writeTaskLaunchesContextIndex(t, root)

	result, err := SelectParticipants(root, ParticipantSelectOptions{
		Task:     "Implement a frontend-only Batch Actions launch readiness improvement. Work only in the frontend Batch Actions surface unless tests prove a shared frontend helper is needed. Do not modify backend/API/database/runtime worker files. Add a compact launch readiness summary and make stale async/template/config loads safe.",
		Workflow: ".runweaver/workflows/feature-delivery-swarm.json",
		Runtime:  RuntimeCodex,
		TaskTier: "large",
	})
	if err != nil {
		t.Fatal(err)
	}
	if containsString(result.Participants, "api-route-surface") {
		t.Fatalf("participants = %#v, want negated backend/API/database/runtime signals suppressed; candidates=%#v", result.Participants, result.Candidates)
	}
	if containsString(result.Participants, "stale-anchor-fixer") {
		t.Fatalf("participants = %#v, want stale async UI wording not to select metadata stale-anchor-fixer", result.Participants)
	}
	if !containsString(result.Participants, "repo-surface-engineer") {
		t.Fatalf("participants = %#v, want generic repo-surface-engineer owner when no repo agent matches", result.Participants)
	}
	if !containsString(result.Participants, "batch-action-master") {
		t.Fatalf("participants = %#v, want domain skill batch-action-master", result.Participants)
	}
	if !containsString(result.Participants, "state-management-surface") {
		t.Fatalf("participants = %#v, want frontend state-management participant", result.Participants)
	}
	joinedRationale := strings.Join(append(result.Rationale, assignmentRationale(result.Assignments)...), "\n")
	for _, forbidden := range []string{"cd mcp/sample-workspace-api", "cd mobile-shell"} {
		if strings.Contains(joinedRationale, forbidden) {
			t.Fatalf("rationale = %q, want frontend-only participant rationale to omit %q", joinedRationale, forbidden)
		}
	}
	if !strings.Contains(joinedRationale, "cd frontend && npm run test") {
		t.Fatalf("rationale = %q, want relevant frontend verification retained", joinedRationale)
	}
	for _, rationale := range append(result.Rationale, assignmentRationale(result.Assignments)...) {
		if len(rationale) > 360 {
			t.Fatalf("rationale item is too long (%d chars): %q", len(rationale), rationale)
		}
	}
}

func TestSelectParticipantsSkipsProfileSkillMissingRuntimeFile(t *testing.T) {
	root := t.TempDir()
	writeWorkflowSelectionFixtures(t, root)
	writeTestFile(t, root, ".agents/skills/context-discipline/SKILL.md", generatedMarker+"\n")
	profile := Profile{
		Repos: []RepoProfile{{
			Dir: ".",
			Agents: []AgentProfile{{
				Name:        "repo-surface-engineer",
				Description: "Owns frontend component work",
				FocusFiles:  []string{"frontend/src/components/TaskLaunches.tsx"},
			}},
			CustomSkills: []SkillProfile{{
				Name:        "batch-action-master",
				Description: "Owns Batch Actions UI behavior",
				FocusFiles:  []string{"frontend/src/components/TaskLaunches.tsx"},
			}},
		}},
	}
	if err := WriteJSON(filepath.Join(root, ".codex", "runweaver", "profile.json"), profile); err != nil {
		t.Fatal(err)
	}
	index := RepoIndex{
		SchemaVersion: 1,
		GeneratedAt:   Now(),
		RepoRoot:      root,
		Tools: ToolchainInfo{
			RecommendedCommands: []string{"npm --prefix frontend test"},
		},
		Files: []FileInventoryItem{
			{Path: "frontend/src/components/TaskLaunches.tsx", Category: "ui", Language: "typescript"},
		},
	}
	if err := WriteJSON(filepath.Join(root, ".runweaver", "tmp", "index", "repo-index.json"), index); err != nil {
		t.Fatal(err)
	}

	result, err := SelectParticipants(root, ParticipantSelectOptions{
		Task:    "Implement frontend-only Batch Actions readiness panel",
		Runtime: RuntimeCodex,
	})
	if err != nil {
		t.Fatal(err)
	}
	if containsString(result.Participants, "batch-action-master") {
		t.Fatalf("participants = %#v, want missing runtime skill excluded; warnings=%#v", result.Participants, result.Warnings)
	}
	if !stringsContain(result.Warnings, "skipped batch-action-master: codex skill metadata file is missing") {
		t.Fatalf("warnings = %#v, want missing skill warning", result.Warnings)
	}
}

func writeParticipantSelectionFixtures(t *testing.T, root string) {
	t.Helper()
	writeTestFile(t, root, ".runweaver/workflows/bugfix-swarm.json", `{
  "id": "bugfix-swarm",
  "name": "Bugfix Swarm",
  "description": "Fix bugs and regressions",
  "maxParticipants": 3,
  "phases": [
    {"id": "reproduce", "name": "Reproduce", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "reproduce bug"},
    {"id": "fix", "name": "Fix", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["repo-surface-engineer"], "prompt": "fix bug"},
    {"id": "verify", "name": "Verify", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-test-quality-reviewer"], "prompt": "verify"}
  ]
}`)
	writeTestFile(t, root, ".runweaver/workflows/metadata-refresh-swarm.json", `{
  "id": "metadata-refresh-swarm",
  "name": "Metadata Refresh Swarm",
  "maxParticipants": 2,
  "phases": [
    {"id": "refresh", "name": "Refresh", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["agent-skill-drift-reviewer"], "prompt": "refresh metadata"}
  ]
}`)
	writeTestFile(t, root, ".opencode/swarm/profile.json", `{
  "workspace": {"name": "api", "repos": ["."]},
  "globalAgents": [
    {"name": "repo-surface-indexer", "description": "Scans repository surfaces"},
    {"name": "agent-skill-drift-reviewer", "description": "Reviews runtime agents and skills for stale anchors"},
    {"name": "repo-test-quality-reviewer", "description": "Reviews tests and verification"}
  ],
  "repos": [{
    "dir": ".",
    "kind": "node-api",
    "domain": "API surface",
    "agents": [
      {
        "name": "auth-access-agent",
        "description": "Owns authentication guards, public routes, and token validation.",
        "focusFiles": ["src/auth/auth.guard.ts", "src/auth/decorators/public.decorator.ts"],
        "verification": ["npm run test -- auth.guard.spec.ts"]
      },
      {
        "name": "catalog-agent",
        "description": "Owns catalog routes and product data.",
        "focusFiles": ["src/catalog/catalog.service.ts"]
      }
    ],
    "customSkills": [
      {
        "name": "security-middleware",
        "description": "Use for guards, decorators, filters, middleware, and auth bypass checks.",
        "focusFiles": ["src/auth/auth.guard.ts"],
        "verification": ["npm run test -- auth.guard.spec.ts"]
      }
    ]
  }]
	}`)
}

func writeParticipantProfileForRuntime(t *testing.T, root, runtimeID string) {
	t.Helper()
	adapter, ok := RuntimeAdapterByID(runtimeID)
	if !ok {
		t.Fatalf("unknown runtime %q", runtimeID)
	}
	writeTestFile(t, root, adapter.ProfilePath(), `{
  "workspace": {"name": "frontend", "repos": ["."]},
  "globalAgents": [
    {"name": "repo-surface-indexer", "description": "Scans repository surfaces"},
    {"name": "repo-surface-engineer", "description": "Generic implementation fallback"},
    {"name": "post-implementation-reviewer", "description": "Reviews completed feature implementations for stale async results, allowed state semantics, disabled states, and unnecessary backend/test scaffolding"}
  ],
  "repos": [{
    "dir": ".",
    "kind": "frontend-app",
    "domain": "frontend workflow launcher UI",
    "agents": [
      {
        "name": "workflow-launcher-agent",
        "description": "Owns frontend workflow launcher cards, config dialogs, component state, and UI launch behavior.",
        "focusFiles": [
          "src/components/workflows/FeatureLauncherCard.tsx",
          "src/components/workflows/FeatureLaunchConfigDialog.tsx"
        ]
      },
      {
        "name": "api-security-boundary-reviewer",
        "description": "Reviews API code, auth middleware, server handlers, and unsafe IO in backend routes.",
        "focusFiles": [
          "src/api/workflows/security.ts"
        ]
      },
      {
        "name": "api-test-harness-reviewer",
        "description": "Maintains route tests, mocks, fixtures, and integration harnesses.",
        "focusFiles": [
          "src/components/workflows/FeatureLauncherCard.test.tsx",
          "src/components/workflows/FeatureLaunchConfigDialog.test.tsx"
        ]
      }
    ],
    "customSkills": [
      {
        "name": "ui-page-surface",
        "description": "Work on frontend UI components, cards, dialogs, page state, and visible user flows.",
        "focusFiles": ["src/components/workflows/FeatureLauncherCard.tsx", "src/components/workflows/FeatureLaunchConfigDialog.tsx"]
      },
      {
        "name": "api-route-surface",
        "description": "Work on backend API routes, controllers, handlers, and endpoint contracts.",
        "focusFiles": ["src/api/workflows/route.ts"]
      }
    ]
  }]
}`)
}

func writeFrontendWorkflowContextIndex(t *testing.T, root string) {
	t.Helper()
	index := RepoIndex{
		SchemaVersion: 1,
		GeneratedAt:   Now(),
		RepoRoot:      root,
		Files: []FileInventoryItem{
			{Path: "src/components/workflows/FeatureLauncherCard.tsx", Category: "ui", Language: "typescript"},
			{Path: "src/components/workflows/FeatureLaunchConfigDialog.tsx", Category: "ui", Language: "typescript"},
			{Path: "src/api/workflows/route.ts", Category: "route", Language: "typescript"},
		},
		Symbols: []SymbolInfo{
			{Kind: "function", Name: "FeatureLauncherCard", Path: "src/components/workflows/FeatureLauncherCard.tsx", Line: 10},
			{Kind: "function", Name: "FeatureLaunchConfigDialog", Path: "src/components/workflows/FeatureLaunchConfigDialog.tsx", Line: 20},
		},
	}
	if err := WriteJSON(filepath.Join(root, ".runweaver", "tmp", "index", "repo-index.json"), index); err != nil {
		t.Fatal(err)
	}
}

func writeTaskLaunchesContextIndex(t *testing.T, root string) {
	t.Helper()
	index := RepoIndex{
		SchemaVersion: 1,
		GeneratedAt:   Now(),
		RepoRoot:      root,
		Files: []FileInventoryItem{
			{Path: "frontend/src/components/TaskLaunches.tsx", Category: "ui", Language: "typescript"},
			{Path: "frontend/src/components/TaskLaunchSettingsDialog.tsx", Category: "ui", Language: "typescript"},
			{Path: "frontend/src/pages/AccountsPage.tsx", Category: "ui", Language: "typescript"},
			{Path: "mcp/sample-workspace-api/src/config/env.ts", Category: "config", Language: "typescript"},
			{Path: "src/api/routes/taskLaunches.ts", Category: "route", Language: "typescript"},
		},
		Symbols: []SymbolInfo{
			{Kind: "function", Name: "TaskLaunches", Path: "frontend/src/components/TaskLaunches.tsx", Line: 10},
			{Kind: "function", Name: "TaskLaunchSettingsDialog", Path: "frontend/src/components/TaskLaunchSettingsDialog.tsx", Line: 20},
		},
	}
	if err := WriteJSON(filepath.Join(root, ".runweaver", "tmp", "index", "repo-index.json"), index); err != nil {
		t.Fatal(err)
	}
}

func stringsContain(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func participantCandidateNamed(candidates []ParticipantSelectionCandidate, name string) bool {
	for _, candidate := range candidates {
		if candidate.Name == name {
			return true
		}
	}
	return false
}

func participantAssignmentHasRole(assignments []ParticipantAssignment, name, role string) bool {
	for _, assignment := range assignments {
		if assignment.Name == name && assignment.Role == role {
			return true
		}
	}
	return false
}

func assignmentRationale(assignments []ParticipantAssignment) []string {
	var out []string
	for _, assignment := range assignments {
		out = append(out, assignment.Rationale...)
	}
	return out
}
