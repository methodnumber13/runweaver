package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIStartCreatesWorkflowContract(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/api\n")
	writeCLIFile(t, root, "src/auth/auth.guard.ts", "export class AuthGuard {}\n")
	writeCLIFile(t, root, ".runweaver/workflows/feature-delivery-swarm.json", `{
  "id": "feature-delivery-swarm",
  "name": "Feature Delivery Swarm",
  "description": "Implement new features",
  "phases": []
}`)
	writeCLIFile(t, root, ".runweaver/workflows/bugfix-swarm.json", `{
  "id": "bugfix-swarm",
  "name": "Bugfix Swarm",
  "description": "Fix bugs and regressions",
  "maxParticipants": 3,
  "phases": [
    {"id": "fix", "name": "Fix", "scope": "repo", "mode": "parallel", "writeMode": "write", "agents": ["repo-surface-engineer"], "prompt": "fix"}
  ]
}`)
	writeCLIFile(t, root, ".opencode/swarm/profile.json", `{
  "workspace": {"name": "api", "repos": ["."]},
  "repos": [{
    "dir": ".",
    "agents": [
      {"name": "auth-agent", "description": "Owns auth guards and public route checks", "focusFiles": ["src/auth/auth.guard.ts"]}
    ]
  }]
}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI([]string{
		"start",
		"--repo", root,
		"--runtime", "opencode",
		"--task", "Fix public route auth regression in src/auth/auth.guard.ts",
		"--json",
	}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("start exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"action": "created"`) ||
		!strings.Contains(stdout.String(), `"workflow": "bugfix-swarm"`) ||
		!strings.Contains(stdout.String(), `"auth-agent"`) ||
		!strings.Contains(stdout.String(), `"executionContract"`) {
		t.Fatalf("start stdout = %q, want created bugfix contract with auth-agent", stdout.String())
	}
}
