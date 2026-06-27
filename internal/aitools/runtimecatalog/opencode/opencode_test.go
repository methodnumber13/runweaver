package opencode

import (
	"testing"

	catalog "github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog"
)

func TestAdapterMetadata(t *testing.T) {
	adapter := Adapter{}
	if adapter.ID() != catalog.OpenCode {
		t.Fatalf("ID() = %q, want %q", adapter.ID(), catalog.OpenCode)
	}
	if adapter.Provider().Binary != "opencode" {
		t.Fatalf("provider binary = %q, want opencode", adapter.Provider().Binary)
	}
	if adapter.ProfilePath() != ".opencode/swarm/profile.json" {
		t.Fatalf("profile path = %q", adapter.ProfilePath())
	}
	if !adapter.Capabilities()["execute"].Supported {
		t.Fatal("execute capability should be supported")
	}
	if len(adapter.GeneratedPaths()) == 0 {
		t.Fatal("generated paths should not be empty")
	}
	if adapter.DelegationGuidance() == "" {
		t.Fatal("delegation guidance should not be empty")
	}
	for _, name := range []string{"render", "doctor", "execute", "classify"} {
		if !adapter.Capabilities()[name].Supported {
			t.Fatalf("capability %s should be supported", name)
		}
	}
}
