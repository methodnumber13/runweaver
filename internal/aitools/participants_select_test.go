package aitools

import (
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
