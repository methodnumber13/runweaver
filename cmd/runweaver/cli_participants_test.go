package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIParticipantsSelectReturnsParticipants(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, ".runweaver/workflows/bugfix-swarm.json", `{
  "id": "bugfix-swarm",
  "name": "Bugfix Swarm",
  "maxParticipants": 3,
  "phases": [
    {"id": "fix", "name": "Fix", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["repo-surface-engineer"], "prompt": "fix bug"}
  ]
}`)
	writeCLIFile(t, root, ".opencode/swarm/profile.json", `{
  "workspace": {"name": "api", "repos": ["."]},
  "repos": [{
    "dir": ".",
    "agents": [
      {"name": "auth-agent", "description": "Owns auth guard and public route behavior", "focusFiles": ["src/auth/auth.guard.ts"]}
    ],
    "customSkills": [
      {"name": "security-middleware", "description": "Use for guard and decorator changes", "focusFiles": ["src/auth/auth.guard.ts"]}
    ]
  }]
}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI([]string{
		"participants", "select",
		"--repo", root,
		"--runtime", "opencode",
		"--workflow", ".runweaver/workflows/bugfix-swarm.json",
		"--task", "Fix public route handling in src/auth/auth.guard.ts",
	}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("participants select exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"auth-agent"`) ||
		!strings.Contains(stdout.String(), `"security-middleware"`) ||
		!strings.Contains(stdout.String(), `"cap": 3`) {
		t.Fatalf("participants stdout = %q, want domain agent, skill, and cap", stdout.String())
	}
}
