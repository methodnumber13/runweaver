package aitools

import (
	"fmt"
	catalog "github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// RuntimeProviderRegistry returns supported runtime providers in stable order.
func RuntimeProviderRegistry() []RuntimeProvider {
	adapters := RuntimeAdapters()
	providers := make([]RuntimeProvider, 0, len(adapters))
	for _, adapter := range adapters {
		providers = append(providers, adapter.Provider())
	}
	return providers
}

// RuntimeProviderByID finds a provider by canonical ID or supported alias.
func RuntimeProviderByID(id string) (RuntimeProvider, bool) {
	id = normalizeRuntimeID(id)
	for _, provider := range RuntimeProviderRegistry() {
		if provider.ID == id {
			return provider, true
		}
	}
	return RuntimeProvider{}, false
}

// ResolveRuntimeSelection converts a CLI selection into ordered runtime providers.
func ResolveRuntimeSelection(selection string) ([]RuntimeProvider, error) {
	selection = strings.TrimSpace(selection)
	if selection == "" {
		selection = RuntimeOpenCode
	}
	var ids []string
	for _, part := range strings.Split(selection, ",") {
		id := normalizeRuntimeID(part)
		if id == "" {
			continue
		}
		if id == RuntimeAll {
			for _, provider := range RuntimeProviderRegistry() {
				ids = append(ids, provider.ID)
			}
			continue
		}
		ids = append(ids, id)
	}
	ids = Unique(ids)
	if len(ids) == 0 {
		return nil, fmt.Errorf("runtime is required")
	}
	providers := make([]RuntimeProvider, 0, len(ids))
	for _, id := range ids {
		provider, ok := RuntimeProviderByID(id)
		if !ok {
			return nil, fmt.Errorf("unsupported runtime %q; supported: opencode, codex, claude, all", id)
		}
		providers = append(providers, provider)
	}
	sort.Slice(providers, func(i, j int) bool {
		return runtimeOrder(providers[i].ID) < runtimeOrder(providers[j].ID)
	})
	return providers, nil
}

// DiscoverRuntimes checks binaries, configs, auth files, and generated metadata.
func DiscoverRuntimes(repoPath string, opts RuntimeDiscoveryOptions) ([]RuntimeDiscoveryResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return nil, err
	}
	providers, err := ResolveRuntimeSelection(opts.Runtime)
	if err != nil {
		return nil, err
	}
	results := make([]RuntimeDiscoveryResult, 0, len(providers))
	for _, provider := range providers {
		results = append(results, discoverRuntime(root, provider))
	}
	return results, nil
}

func discoverRuntime(root string, provider RuntimeProvider) RuntimeDiscoveryResult {
	result := RuntimeDiscoveryResult{
		ID:           provider.ID,
		Name:         provider.Name,
		Binary:       provider.Binary,
		Status:       "warning",
		Env:          runtimeEnv(provider.ID),
		Capabilities: runtimeCapabilities(provider.ID),
	}
	if path, err := exec.LookPath(provider.Binary); err == nil {
		result.BinaryFound = true
		result.BinaryPath = path
	} else {
		result.Issues = append(result.Issues, provider.Binary+" is not available on PATH")
	}
	adapter, ok := RuntimeAdapterByID(provider.ID)
	if !ok {
		result.Issues = append(result.Issues, "unsupported runtime")
		return result
	}
	configs, auth, metadata, managed := adapter.PathChecks(root)
	result.ConfigFiles = configs
	result.AuthFiles = auth
	result.MetadataFiles = metadata
	result.ManagedFiles = managed
	result.GeneratedPaths = adapter.GeneratedPaths()
	if hasAnyReadable(configs) || hasAnyReadable(metadata) || provider.ID == RuntimeOpenCode && hasAnyReadable(auth) {
		result.Ready = result.BinaryFound
	} else if result.BinaryFound {
		result.Warnings = append(result.Warnings, "runtime binary is available but no project/global runtime metadata was found")
	}
	if result.Ready {
		result.Status = "ok"
	} else if len(result.Issues) > 0 {
		result.Status = "warning"
	}
	return result
}

func normalizeRuntimeID(value string) string {
	return catalog.NormalizeID(value)
}

func runtimeOrder(id string) int {
	return catalog.Order(id)
}

func runtimeEnv(id string) map[string]bool {
	keys := []string{}
	switch id {
	case RuntimeOpenCode:
		keys = []string{"OPENCODE_CONFIG", "OPENCODE_CONFIG_DIR", "OPENCODE_CONFIG_CONTENT", "RUNWEAVER_MODEL_API_KEY", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "APPDATA", "ProgramData"}
	case RuntimeCodex:
		keys = []string{"CODEX_HOME", "CODEX_API_KEY", "OPENAI_API_KEY", "XDG_CONFIG_HOME", "APPDATA", "ProgramData"}
	case RuntimeClaude:
		keys = []string{"CLAUDE_CONFIG_DIR", "ANTHROPIC_API_KEY", "CLAUDE_CODE_USE_BEDROCK", "CLAUDE_CODE_USE_VERTEX", "APPDATA", "ProgramData"}
	}
	out := map[string]bool{}
	for _, key := range keys {
		out[key] = os.Getenv(key) != ""
	}
	return out
}
