package aitools

import (
	"strings"
	"testing"
)

func TestResolveSingleRuntimeAutoIgnoresUnavailableProfile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	writeTestFile(t, root, ".codex/runweaver/profile.json", `{
  "workspace": {"name": "api", "repos": ["."]},
  "repos": [{"dir": ".", "agents": [{"name": "codex-agent", "description": "Codex owner"}]}]
}`)

	selected, resolution, err := ResolveSingleRuntime(root, RuntimeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if selected != RuntimeOpenCode {
		t.Fatalf("selected = %q, want opencode fallback when codex binary is unavailable; resolution=%#v", selected, resolution)
	}
	codex := runtimeResolutionCandidateByID(resolution.Candidates, RuntimeCodex)
	if codex == nil {
		t.Fatalf("candidates = %#v, want codex candidate", resolution.Candidates)
	}
	if codex.BinaryFound {
		t.Fatalf("codex candidate = %#v, want binaryFound false", codex)
	}
	if !strings.Contains(codex.Source, "unavailable") {
		t.Fatalf("codex candidate = %#v, want unavailable source", codex)
	}
}

func TestResolveSingleRuntimeAutoUsesReadyProfile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	writeRuntimeShimOnPath(t, "codex")
	writeTestFile(t, root, ".codex/runweaver/profile.json", `{
  "workspace": {"name": "api", "repos": ["."]},
  "repos": [{"dir": ".", "agents": [{"name": "codex-agent", "description": "Codex owner"}]}]
}`)

	selected, resolution, err := ResolveSingleRuntime(root, RuntimeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if selected != RuntimeCodex {
		t.Fatalf("selected = %q, want ready codex profile; resolution=%#v", selected, resolution)
	}
	codex := runtimeResolutionCandidateByID(resolution.Candidates, RuntimeCodex)
	if codex == nil || !codex.Ready || !codex.BinaryFound {
		t.Fatalf("codex candidate = %#v, want ready with binary", codex)
	}
	if !strings.Contains(codex.Source, "ready") {
		t.Fatalf("codex candidate = %#v, want ready source", codex)
	}
}

func runtimeResolutionCandidateByID(candidates []RuntimeResolutionCandidate, id string) *RuntimeResolutionCandidate {
	for index := range candidates {
		if candidates[index].ID == id {
			return &candidates[index]
		}
	}
	return nil
}
