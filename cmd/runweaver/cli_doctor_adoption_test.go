package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIDoctorAdoptionReportsRuntimeContract(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, ".runweaver/workflows/feature-delivery-swarm.json", `{"id":"feature-delivery-swarm","name":"Feature","phases":[]}`)
	writeCLIFile(t, root, ".opencode/swarm/profile.json", `{"workspace":{"name":"repo"},"repos":[{"dir":"."}]}`)
	writeCLIFile(t, root, ".opencode/agents/swarm.md", "Run `runweaver start --repo . --task \"<task>\"` first.\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI([]string{"doctor", "adoption", "--repo", root, "--runtime", "opencode"}, &stdout, &stderr, false)
	if code != 0 {
		t.Fatalf("doctor adoption exit code = %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"runtime": "opencode"`) ||
		!strings.Contains(stdout.String(), `"startup-contract"`) ||
		!strings.Contains(stdout.String(), `"ready": true`) {
		t.Fatalf("doctor adoption stdout = %q, want opencode adoption contract", stdout.String())
	}
}
