package main

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLIDoctorAdoptionReportsRuntimeContract(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, ".runweaver/workflows/feature-delivery-swarm.json", `{"id":"feature-delivery-swarm","name":"Feature","phases":[]}`)
	binName := ".runweaver/tmp/bin/runweaver"
	binContent := "#!/bin/sh\nif [ \"$1\" = \"help\" ]; then\nprintf '%s\\n' 'start --repo <path>' 'status --repo <path>' 'context query --task <text>'\nexit 0\nfi\nexit 0\n"
	if runtime.GOOS == "windows" {
		binName = ".runweaver/tmp/bin/runweaver.cmd"
		binContent = "@echo off\r\nif \"%1\"==\"help\" (\r\necho start --repo ^<path^>\r\necho status --repo ^<path^>\r\necho context query --task ^<text^>\r\nexit /b 0\r\n)\r\nexit /b 0\r\n"
	}
	writeExecutable(t, filepath.Join(root, binName), binContent)
	writeCLIFile(t, root, ".opencode/swarm/profile.json", `{"workspace":{"name":"repo"},"repos":[{"dir":"."}]}`)
	writeCLIFile(t, root, ".opencode/agents/runweaver-swarm.md", "permission:\n  bash:\n    \".runweaver/tmp/bin/runweaver *\": allow\nRun `RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --task \"<task>\"` first.\n")
	writeCLIFile(t, root, ".opencode/commands/runweaver-start.md", "RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --runtime opencode --task \"$ARGUMENTS\"\n")

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
