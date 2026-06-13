package aitools

import (
	"fmt"
	"sort"

	catalog "github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog"
	claudecatalog "github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog/claude"
	codexcatalog "github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog/codex"
	opencodecatalog "github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog/opencode"
)

// RuntimeAdapter renders, discovers, and executes one supported AI runtime.
type RuntimeAdapter interface {
	// ID returns the canonical runtime ID.
	ID() string
	// Provider returns user-facing runtime metadata.
	Provider() RuntimeProvider
	// ProfilePath returns the runtime-specific generated profile path.
	ProfilePath() string
	// GeneratedPaths returns files and directories managed by RunWeaver.
	GeneratedPaths() []string
	// Capabilities returns runtime support flags.
	Capabilities() map[string]RuntimeFlag
	// PathChecks returns config, auth, metadata, and managed-file probes.
	PathChecks(root string) (configs, auth, metadata, managed []RuntimeFileCheck)
	// MaterializeProfile writes runtime-specific generated metadata.
	MaterializeProfile(root string, profile Profile, force bool) error
	// ExecutionSpec builds the command used to execute a workflow.
	ExecutionSpec(root string, opts WorkflowExecuteOptions, plan WorkflowRunSummary, prompt string) (workflowRuntimeExecutionSpec, error)
	// ClassifierSpec builds the command used for AI-backed classification.
	ClassifierSpec(root, prompt string, opts ClassifyOptions, finalOutputPath string) (classifierCommandSpec, error)
	// DelegationGuidance returns runtime-specific orchestration instructions.
	DelegationGuidance() string
}

// RuntimeAdapters returns all concrete runtime adapters in stable execution order.
func RuntimeAdapters() []RuntimeAdapter {
	adapters := []RuntimeAdapter{
		openCodeRuntimeAdapter{metadata: opencodecatalog.Adapter{}},
		codexRuntimeAdapter{metadata: codexcatalog.Adapter{}},
		claudeRuntimeAdapter{metadata: claudecatalog.Adapter{}},
	}
	sort.Slice(adapters, func(i, j int) bool {
		return runtimeOrder(adapters[i].ID()) < runtimeOrder(adapters[j].ID())
	})
	return adapters
}

// RuntimeAdapterByID returns the concrete adapter for a runtime ID or alias.
func RuntimeAdapterByID(id string) (RuntimeAdapter, bool) {
	id = normalizeRuntimeID(id)
	for _, adapter := range RuntimeAdapters() {
		if adapter.ID() == id {
			return adapter, true
		}
	}
	return nil, false
}

func mustRuntimeAdapter(id string) (RuntimeAdapter, error) {
	adapter, ok := RuntimeAdapterByID(id)
	if !ok {
		return nil, fmt.Errorf("unsupported runtime %q; supported: opencode, codex, claude", id)
	}
	return adapter, nil
}

func runtimeProviderFromCatalog(provider catalog.Provider) RuntimeProvider {
	return RuntimeProvider{
		ID:          provider.ID,
		Name:        provider.Name,
		Binary:      provider.Binary,
		Description: provider.Description,
	}
}

func runtimeFlagsFromCatalog(flags map[string]catalog.Flag) map[string]RuntimeFlag {
	out := make(map[string]RuntimeFlag, len(flags))
	for key, flag := range flags {
		out[key] = RuntimeFlag{Supported: flag.Supported, Summary: flag.Summary}
	}
	return out
}
