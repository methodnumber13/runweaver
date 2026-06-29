package aitools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolveSingleRuntime selects one runtime for task intake.
func ResolveSingleRuntime(root, selection string) (string, RuntimeResolutionResult, error) {
	requested := normalizeRuntimeID(selection)
	if requested == "" {
		requested = RuntimeAuto
	}
	if requested != RuntimeAuto && requested != RuntimeAll {
		provider, ok := RuntimeProviderByID(requested)
		if !ok {
			return "", RuntimeResolutionResult{}, fmt.Errorf("unsupported runtime %q; supported: auto, opencode, codex, claude", requested)
		}
		return provider.ID, RuntimeResolutionResult{
			Requested: requested,
			Selected:  provider.ID,
			Source:    "explicit",
			Candidates: []RuntimeResolutionCandidate{{
				ID:     provider.ID,
				Score:  100,
				Source: "explicit",
			}},
		}, nil
	}
	candidates := runtimeResolutionCandidates(root)
	selected := RuntimeOpenCode
	source := "default"
	bestScore := -1
	for _, candidate := range candidates {
		if candidate.Score > bestScore {
			selected = candidate.ID
			source = candidate.Source
			bestScore = candidate.Score
		}
	}
	return selected, RuntimeResolutionResult{
		Requested:  requested,
		Selected:   selected,
		Source:     source,
		Candidates: candidates,
	}, nil
}

func runtimeResolutionCandidates(root string) []RuntimeResolutionCandidate {
	var candidates []RuntimeResolutionCandidate
	for _, provider := range RuntimeProviderRegistry() {
		adapter, ok := RuntimeAdapterByID(provider.ID)
		if !ok {
			continue
		}
		profile := adapter.ProfilePath()
		score := 1
		source := "default"
		generated := runtimeGeneratedMetadataExists(root, adapter.GeneratedPaths())
		if generated {
			score = 40
			source = "generated-metadata"
		}
		if Exists(filepath.Join(root, profile)) {
			score = 100
			source = "profile"
		}
		candidates = append(candidates, RuntimeResolutionCandidate{
			ID:        provider.ID,
			Score:     score,
			Source:    source,
			Profile:   profile,
			Generated: generated,
		})
	}
	return candidates
}

func runtimeGeneratedMetadataExists(root string, paths []string) bool {
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if Exists(filepath.Join(root, path)) {
			return true
		}
	}
	return false
}
