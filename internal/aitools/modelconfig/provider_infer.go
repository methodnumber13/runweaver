package modelconfig

import (
	"os"
	"strings"
)

const defaultProviderID = "openai-compatible"

func resolveProviderID(root string, opts ModelConfigCheckOptions) string {
	if providerID := strings.TrimSpace(opts.ProviderID); providerID != "" {
		return providerID
	}
	if providerID := inferProviderID(root); providerID != "" {
		return providerID
	}
	return defaultProviderID
}

func inferProviderID(root string) string {
	var providerID string
	inferredFromModel := false
	apply := func(parsed map[string]any) {
		if parsed == nil {
			return
		}
		if model, ok := parsed["model"].(string); ok {
			if prefix := providerIDFromModel(model); prefix != "" {
				providerID = prefix
				inferredFromModel = true
				return
			}
		}
		if !inferredFromModel {
			if single := singleProviderID(parsed); single != "" {
				providerID = single
			}
		}
	}

	for _, candidate := range ConfigPaths(root) {
		_, parsed := inspectConfigFile(candidate.Path, candidate.Source, ModelConfigCheckOptions{})
		apply(parsed)
	}
	if content := strings.TrimSpace(os.Getenv("OPENCODE_CONFIG_CONTENT")); content != "" {
		_, parsed := inspectConfigContent(content, "env:OPENCODE_CONFIG_CONTENT", ModelConfigCheckOptions{})
		apply(parsed)
	}
	for _, candidate := range ManagedConfigPaths() {
		_, parsed := inspectConfigFile(candidate.Path, candidate.Source, ModelConfigCheckOptions{})
		apply(parsed)
	}
	return providerID
}

func providerIDFromModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}
	providerID, _, ok := strings.Cut(model, "/")
	if !ok {
		return ""
	}
	return strings.TrimSpace(providerID)
}

func singleProviderID(parsed map[string]any) string {
	providers, ok := parsed["provider"].(map[string]any)
	if !ok || len(providers) != 1 {
		return ""
	}
	for providerID := range providers {
		return strings.TrimSpace(providerID)
	}
	return ""
}
