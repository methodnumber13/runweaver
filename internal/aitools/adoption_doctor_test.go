package aitools

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDoctorAdoptionPassesWhenRuntimeMetadataHasStartContract(t *testing.T) {
	root := t.TempDir()
	writeAdoptionFixtures(t, root)

	result, err := DoctorAdoption(root, AdoptionDoctorOptions{Runtime: RuntimeAll})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ok" {
		t.Fatalf("adoption = %#v, want ready ok", result)
	}
	if len(result.Runtimes) != 3 {
		t.Fatalf("runtimes = %d, want 3", len(result.Runtimes))
	}
	for _, runtime := range result.Runtimes {
		if !runtime.Ready {
			t.Fatalf("runtime %s not ready: %#v", runtime.ID, runtime.Checks)
		}
		if !adoptionCheckNamed(runtime.Checks, "command-shortcut") {
			t.Fatalf("runtime %s checks = %#v, want command-shortcut", runtime.ID, runtime.Checks)
		}
		if !adoptionCheckNamed(runtime.Checks, "runtime-permissions") {
			t.Fatalf("runtime %s checks = %#v, want runtime-permissions", runtime.ID, runtime.Checks)
		}
		if !adoptionCheckNamed(runtime.Checks, "mcp-tooling") {
			t.Fatalf("runtime %s checks = %#v, want mcp-tooling", runtime.ID, runtime.Checks)
		}
	}
}

func TestDoctorAdoptionWarnsWhenStartContractMissing(t *testing.T) {
	root := t.TempDir()
	writeAdoptionFixtures(t, root)
	writeTestFile(t, root, "AGENTS.md", "Run tests before final response.\n")
	writeTestFile(t, root, ".codex/agents/runweaver-swarm.toml", "name = \"runweaver-swarm\"\ndescription = \"Generic agent\"\ndeveloper_instructions = \"Read AGENTS.md\"\n")

	result, err := DoctorAdoption(root, AdoptionDoctorOptions{Runtime: RuntimeCodex})
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status != "warning" {
		t.Fatalf("adoption = %#v, want warning not ready", result)
	}
	if len(result.Runtimes) != 1 || result.Runtimes[0].ID != RuntimeCodex {
		t.Fatalf("runtimes = %#v, want codex only", result.Runtimes)
	}
}

func writeAdoptionFixtures(t *testing.T, root string) {
	t.Helper()
	writeRunWeaverCommandFixture(t, root, true)
	writeTestFile(t, root, ".runweaver/workflows/feature-delivery-swarm.json", `{
  "id": "feature-delivery-swarm",
  "name": "Feature Delivery Swarm",
  "phases": []
}`)
	profile := `{"workspace":{"name":"repo","repos":["."]},"repos":[{"dir":".","agents":[{"name":"repo-agent","description":"Repo agent"}]}]}`
	writeTestFile(t, root, ".opencode/swarm/profile.json", profile)
	writeTestFile(t, root, ".codex/runweaver/profile.json", profile)
	writeTestFile(t, root, ".claude/runweaver/profile.json", profile)
	writeTestFile(t, root, ".opencode/agents/runweaver-swarm.md", "Run `RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --task \"<task>\"` before coding.\n")
	writeTestFile(t, root, "AGENTS.md", "Run `RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --task \"<task>\"` before coding.\n")
	writeTestFile(t, root, ".codex/agents/runweaver-swarm.toml", "name = \"runweaver-swarm\"\ndescription = \"Run repo-local runweaver start before coding.\"\ndeveloper_instructions = \"RUNWEAVER_BIN=\\\".runweaver/tmp/bin/runweaver\\\"; \\\"$RUNWEAVER_BIN\\\" start --repo .\"\n")
	writeTestFile(t, root, "CLAUDE.md", "Run `RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --task \"<task>\"` before coding.\n")
	writeTestFile(t, root, ".claude/agents/runweaver-swarm.md", "---\nname: runweaver-swarm\n---\nRun `RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo .` before coding.\n")
	writeTestFile(t, root, ".opencode/commands/runweaver-start.md", "---\ndescription: Start RunWeaver\n---\nRUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --runtime opencode --task \"$ARGUMENTS\"\n")
	writeTestFile(t, root, ".agents/skills/runweaver-start/SKILL.md", "---\nname: runweaver-start\ndescription: Start RunWeaver task intake\n---\nRun `RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --runtime codex --task \"<full raw user task>\"`.\n")
	writeTestFile(t, root, ".claude/skills/runweaver-start/SKILL.md", "---\nname: runweaver-start\ndescription: Start RunWeaver task intake\n---\nRun `RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --runtime claude --task \"<full raw user task>\"`.\n")
	writeTestFile(t, root, ".opencode/agents/runweaver-swarm.md", "permission:\n  bash:\n    \".runweaver/tmp/bin/runweaver *\": allow\nRun `RUNWEAVER_BIN=\".runweaver/tmp/bin/runweaver\"; \"$RUNWEAVER_BIN\" start --repo . --task \"<task>\"` before coding.\n")
	writeTestFile(t, root, ".codex/config.toml", "sandbox_mode = \"workspace-write\"\n")
	writeTestFile(t, root, ".claude/settings.json", `{"permissions":{"allow":["Bash(.runweaver/tmp/bin/runweaver:*)"]}}`)
}

func TestDoctorAdoptionErrorsWhenRunWeaverCommandSurfaceIsStale(t *testing.T) {
	root := t.TempDir()
	t.Setenv("RUNWEAVER_BIN", "")
	writeAdoptionFixtures(t, root)
	writeRunWeaverCommandFixture(t, root, false)

	result, err := DoctorAdoption(root, AdoptionDoctorOptions{Runtime: RuntimeCodex})
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || result.Status != "warning" {
		t.Fatalf("adoption = %#v, want warning not ready", result)
	}
	check, ok := adoptionCheckByName(result.Runtimes[0].Checks, "runweaver-command-surface")
	if !ok {
		t.Fatalf("checks = %#v, want runweaver-command-surface", result.Runtimes[0].Checks)
	}
	if check.Status != "error" {
		t.Fatalf("surface check = %#v, want error", check)
	}
	if !containsString(check.Evidence, runWeaverProjectBinPath(root)+": missing start --repo, status --repo, context query") {
		t.Fatalf("surface evidence = %#v, want missing required command evidence", check.Evidence)
	}
}

func writeRunWeaverCommandFixture(t *testing.T, root string, fullSurface bool) {
	t.Helper()
	path := runWeaverProjectBinPath(root)
	var content string
	if runtime.GOOS == "windows" {
		content = "@echo off\r\nif \"%1\"==\"help\" (\r\n"
		if fullSurface {
			content += "echo start --repo ^<path^>\r\necho status --repo ^<path^>\r\necho context query --task ^<text^>\r\n"
		} else {
			content += "echo scan --repo ^<path^>\r\n"
		}
		content += "exit /b 0\r\n)\r\nexit /b 0\r\n"
	} else {
		content = "#!/bin/sh\nif [ \"$1\" = \"help\" ]; then\n"
		if fullSurface {
			content += "printf '%s\\n' 'start --repo <path>' 'status --repo <path>' 'context query --task <text>'\n"
		} else {
			content += "printf '%s\\n' 'scan --repo <path>'\n"
		}
		content += "exit 0\nfi\nexit 0\n"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func adoptionCheckNamed(checks []AdoptionCheck, name string) bool {
	for _, check := range checks {
		if check.Name == name {
			return true
		}
	}
	return false
}

func adoptionCheckByName(checks []AdoptionCheck, name string) (AdoptionCheck, bool) {
	for _, check := range checks {
		if check.Name == name {
			return check, true
		}
	}
	return AdoptionCheck{}, false
}
