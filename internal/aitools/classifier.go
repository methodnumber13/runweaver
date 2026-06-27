package aitools

import (
	"fmt"
	"path/filepath"
)

// ClassifyRepoDeterministic classifies a repository index without model calls.
func ClassifyRepoDeterministic(index RepoIndex) RepoClassification {
	return ClassifyRepo(index)
}

// ClassifyRepository indexes and classifies a repository, optionally applying metadata.
func ClassifyRepository(repoPath string, opts ClassifyOptions) (ClassifyResult, error) {
	return classifyRepository(repoPath, opts, runCommandOutputWithEnv)
}

func classifyRepository(repoPath string, opts ClassifyOptions, runner outputRunner) (ClassifyResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return ClassifyResult{}, err
	}
	opts, err = normalizeClassifyOptions(opts, ClassificationDeterministic)
	if err != nil {
		return ClassifyResult{}, err
	}
	index, err := indexWithOptions(root, IndexOptions{
		ChangedOnly:    true,
		Prune:          true,
		MaxCacheMB:     256,
		Classification: opts,
	}, runner)
	if err != nil {
		return ClassifyResult{}, err
	}
	run := ClassifyRunSummary{}
	if index.ClassifierRun != nil {
		run = *index.ClassifierRun
	}
	result := ClassifyResult{
		Status:         "success",
		RepoRoot:       root,
		IndexPath:      index.Artifacts.Index,
		Applied:        opts.Apply,
		Classifier:     run,
		Artifacts:      index.Artifacts,
		Classification: index.Classification,
		Stats:          index.Stats,
		Warnings:       index.Warnings,
		Stack:          index.Surface.Stack,
		Tools:          index.Tools,
	}
	if opts.Apply {
		providers, err := ResolveRuntimeSelection(opts.ApplyRuntime)
		if err != nil {
			return ClassifyResult{}, err
		}
		runtimeIDs := runtimeProviderIDs(providers)
		profile := GenerateProfileFromIndex(index)
		profileJSON, err := safeJSON(profile)
		if err != nil {
			return ClassifyResult{}, err
		}
		for _, path := range runtimeProfilePaths(runtimeIDs) {
			if err := writeIfAllowed(filepath.Join(root, path), profileJSON, true); err != nil {
				return ClassifyResult{}, fmt.Errorf("write profile: %w", err)
			}
			result.ProfilePaths = append(result.ProfilePaths, path)
		}
		if err := MaterializeProfileForRuntimes(root, profile, true, runtimeIDs); err != nil {
			return ClassifyResult{}, err
		}
		if len(result.ProfilePaths) > 0 {
			result.ProfilePath = result.ProfilePaths[0]
		}
	}
	return result, nil
}
