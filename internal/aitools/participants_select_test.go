package aitools

import "testing"

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
	if result.Cap != 3 {
		t.Fatalf("cap = %d, want 3", result.Cap)
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
  "workspace": {"name": "bff", "repos": ["."]},
  "globalAgents": [
    {"name": "repo-surface-indexer", "description": "Scans repository surfaces"},
    {"name": "agent-skill-drift-reviewer", "description": "Reviews runtime agents and skills for stale anchors"},
    {"name": "repo-test-quality-reviewer", "description": "Reviews tests and verification"}
  ],
  "repos": [{
    "dir": ".",
    "kind": "node-api",
    "domain": "BFF API",
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
