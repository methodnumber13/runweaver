package runtimecatalog_test

import (
	"testing"

	catalog "github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog"
	"github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog/claude"
	"github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog/codex"
	"github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog/opencode"
)

func TestRuntimeCatalogAdaptersExposeStableMetadata(t *testing.T) {
	adapters := []catalog.Adapter{
		opencode.Adapter{},
		codex.Adapter{},
		claude.Adapter{},
	}
	wantProfiles := map[string]string{
		catalog.OpenCode: ".opencode/swarm/profile.json",
		catalog.Codex:    ".codex/runweaver/profile.json",
		catalog.Claude:   ".claude/runweaver/profile.json",
	}
	for _, adapter := range adapters {
		provider := adapter.Provider()
		if provider.ID != adapter.ID() {
			t.Fatalf("%s provider ID = %q, want adapter ID", adapter.ID(), provider.ID)
		}
		if provider.Binary == "" {
			t.Fatalf("%s provider binary is empty", adapter.ID())
		}
		if adapter.ProfilePath() != wantProfiles[adapter.ID()] {
			t.Fatalf("%s profile path = %q, want %q", adapter.ID(), adapter.ProfilePath(), wantProfiles[adapter.ID()])
		}
		if !contains(adapter.GeneratedPaths(), adapter.ProfilePath()) {
			t.Fatalf("%s generated paths = %#v, want profile path", adapter.ID(), adapter.GeneratedPaths())
		}
		for _, capability := range []string{"render", "doctor", "execute", "classify"} {
			if !adapter.Capabilities()[capability].Supported {
				t.Fatalf("%s capability %q is not supported", adapter.ID(), capability)
			}
		}
		if adapter.DelegationGuidance() == "" {
			t.Fatalf("%s delegation guidance is empty", adapter.ID())
		}
	}
}

func TestRuntimeCatalogNormalizeAndOrder(t *testing.T) {
	for input, want := range map[string]string{
		"open-code":   catalog.OpenCode,
		"open_code":   catalog.OpenCode,
		"codex":       catalog.Codex,
		"claude-code": catalog.Claude,
		"claudecode":  catalog.Claude,
	} {
		if got := catalog.NormalizeID(input); got != want {
			t.Fatalf("NormalizeID(%q) = %q, want %q", input, got, want)
		}
	}
	if !(catalog.Order(catalog.OpenCode) < catalog.Order(catalog.Codex) && catalog.Order(catalog.Codex) < catalog.Order(catalog.Claude)) {
		t.Fatalf("unexpected runtime order: opencode=%d codex=%d claude=%d", catalog.Order(catalog.OpenCode), catalog.Order(catalog.Codex), catalog.Order(catalog.Claude))
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
