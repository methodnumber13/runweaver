package aitools

import "testing"

func TestRuntimeAdaptersExposeProviderParity(t *testing.T) {
	adapters := RuntimeAdapters()
	if len(adapters) != 3 {
		t.Fatalf("RuntimeAdapters() len = %d, want 3", len(adapters))
	}

	wantProfiles := map[string]string{
		RuntimeOpenCode: ".opencode/swarm/profile.json",
		RuntimeCodex:    ".codex/runweaver/profile.json",
		RuntimeClaude:   ".claude/runweaver/profile.json",
	}
	for _, adapter := range adapters {
		provider := adapter.Provider()
		if provider.ID != adapter.ID() {
			t.Fatalf("provider ID = %q, adapter ID = %q", provider.ID, adapter.ID())
		}
		if got := adapter.ProfilePath(); got != wantProfiles[adapter.ID()] {
			t.Fatalf("%s ProfilePath() = %q, want %q", adapter.ID(), got, wantProfiles[adapter.ID()])
		}
		if !containsString(adapter.GeneratedPaths(), adapter.ProfilePath()) {
			t.Fatalf("%s generated paths = %#v, want profile path %q", adapter.ID(), adapter.GeneratedPaths(), adapter.ProfilePath())
		}
		capabilities := adapter.Capabilities()
		for _, capability := range []string{"render", "doctor", "execute", "classify"} {
			if !capabilities[capability].Supported {
				t.Fatalf("%s capability %q is not supported: %#v", adapter.ID(), capability, capabilities)
			}
		}
	}
}

func TestRuntimeAdapterLookupNormalizesAliases(t *testing.T) {
	for input, want := range map[string]string{
		"open-code":   RuntimeOpenCode,
		"codex":       RuntimeCodex,
		"claude-code": RuntimeClaude,
	} {
		adapter, ok := RuntimeAdapterByID(input)
		if !ok {
			t.Fatalf("RuntimeAdapterByID(%q) not found", input)
		}
		if adapter.ID() != want {
			t.Fatalf("RuntimeAdapterByID(%q) = %q, want %q", input, adapter.ID(), want)
		}
	}
}
