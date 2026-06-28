package aitools

import "testing"

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
	}
}

func TestDoctorAdoptionWarnsWhenStartContractMissing(t *testing.T) {
	root := t.TempDir()
	writeAdoptionFixtures(t, root)
	writeTestFile(t, root, "AGENTS.md", "Run tests before final response.\n")
	writeTestFile(t, root, ".codex/agents/swarm.toml", "name = \"swarm\"\ndescription = \"Generic agent\"\ndeveloper_instructions = \"Read AGENTS.md\"\n")

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
	writeTestFile(t, root, ".runweaver/workflows/feature-delivery-swarm.json", `{
  "id": "feature-delivery-swarm",
  "name": "Feature Delivery Swarm",
  "phases": []
}`)
	profile := `{"workspace":{"name":"repo","repos":["."]},"repos":[{"dir":".","agents":[{"name":"repo-agent","description":"Repo agent"}]}]}`
	writeTestFile(t, root, ".opencode/swarm/profile.json", profile)
	writeTestFile(t, root, ".codex/runweaver/profile.json", profile)
	writeTestFile(t, root, ".claude/runweaver/profile.json", profile)
	writeTestFile(t, root, ".opencode/agents/swarm.md", "Run `runweaver start --repo . --task \"<task>\"` before coding.\n")
	writeTestFile(t, root, "AGENTS.md", "Run `runweaver start --repo . --task \"<task>\"` before coding.\n")
	writeTestFile(t, root, ".codex/agents/swarm.toml", "name = \"swarm\"\ndescription = \"Run runweaver start --repo . before coding.\"\ndeveloper_instructions = \"Run runweaver start --repo .\"\n")
	writeTestFile(t, root, "CLAUDE.md", "Run `runweaver start --repo . --task \"<task>\"` before coding.\n")
	writeTestFile(t, root, ".claude/agents/swarm.md", "---\nname: swarm\n---\nRun `runweaver start --repo .` before coding.\n")
}
