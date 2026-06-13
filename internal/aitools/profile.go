package aitools

import (
	"path/filepath"
	"strings"
)

// GenerateProfile builds runtime-neutral metadata from a lightweight surface scan.
func GenerateProfile(index SurfaceIndex) Profile {
	repoName := filepath.Base(index.RepoRoot)
	kind := index.Stack.Kind
	if kind == "" {
		kind = "generic"
	}
	keyFiles := []string{}
	keyFiles = append(keyFiles, index.ConfigFiles...)
	keyFiles = append(keyFiles, index.EntryPoints...)
	keyFiles = append(keyFiles, index.SourceDirs...)
	keyFiles = append(keyFiles, index.Routes...)
	keyFiles = append(keyFiles, index.Pages...)
	keyFiles = Unique(keyFiles)
	keyFiles = Limit(keyFiles, 40)

	return Profile{
		Workspace: WorkspaceProfile{
			Name:        repoName,
			Description: "Generated OpenCode swarm profile for " + repoName + ".",
			Repos:       []string{"."},
		},
		GlobalAgents: []AgentProfile{
			{Name: "repo-surface-indexer", Description: "Scans repository surfaces and maintains AI-readable surface indexes", Mode: "subagent"},
			{Name: "agent-skill-drift-reviewer", Description: "Reviews runtime agents and skills for stale anchors and missing surfaces", Mode: "subagent"},
			{Name: "profile-regenerator", Description: "Regenerates local runtime profile proposals from scan results", Mode: "subagent"},
			{Name: "stale-anchor-fixer", Description: "Repairs stale file anchors in generated runtime metadata", Mode: "subagent"},
			{Name: "repo-surface-engineer", Description: "Generic implementer role used by portable workflows when a stack-specific engineer is not selected", Mode: "subagent"},
			{Name: "repo-contract-reviewer", Description: "Generic contract and boundary reviewer used by portable workflows", Mode: "subagent"},
			{Name: "repo-test-quality-reviewer", Description: "Generic test and verification reviewer used by portable workflows", Mode: "subagent"},
		},
		Repos: []RepoProfile{{
			Dir:      ".",
			Kind:     kind,
			Purpose:  "Generated profile for " + repoName + ".",
			Domain:   domainFor(index),
			Runtime:  runtimeFor(index),
			KeyFiles: keyFiles,
			Risks: []string{
				"Generated profile must be reviewed before committing.",
				"Do not store secrets in OpenCode agents, skills, profiles, or run artifacts.",
				"Regenerate metadata after moving routes, pages, services, tests, or build configs.",
			},
			Agents:       suggestedAgents(index),
			CustomSkills: suggestedSkills(index),
		}},
	}
}

// GenerateProfileFromIndex builds runtime-neutral metadata from the full repo index.
func GenerateProfileFromIndex(repoIndex RepoIndex) Profile {
	profile := GenerateProfile(repoIndex.Surface)
	if len(profile.Repos) == 0 {
		return profile
	}
	classification := repoIndex.Classification
	if classification.Source == "" {
		classification = ClassifyRepo(repoIndex)
	}
	repo := &profile.Repos[0]
	repo.Runtime = runtimeFromToolchain(repoIndex)
	repo.Domain = domainFromIndex(repoIndex)
	repo.Semantic = classification
	repo.KeyFiles = keyFilesFromIndex(repoIndex)
	if isModelBackedClassification(classification) {
		repo.Agents = enrichAgentProfiles(repoIndex, mergeAgents(nil, classification.Agents...))
		repo.CustomSkills = mergeSkills(nil, classification.Skills...)
		return profile
	}
	repo.Agents = mergeAgents(repo.Agents, packageAgents(repoIndex)...)
	repo.Agents = mergeAgents(repo.Agents, classification.Agents...)
	repo.Agents = enrichAgentProfiles(repoIndex, repo.Agents)
	repo.CustomSkills = mergeSkills(repo.CustomSkills, packageSkills(repoIndex)...)
	repo.CustomSkills = mergeSkills(repo.CustomSkills, classification.Skills...)
	return profile
}

func isModelBackedClassification(classification RepoClassification) bool {
	return classification.Source == "model-backed" || strings.HasPrefix(classification.Source, "model-backed-")
}

func domainFor(index SurfaceIndex) string {
	switch index.Stack.Kind {
	case "frontend-react":
		return "frontend routes, components, state, API clients, and tests"
	case "node-api":
		return "Node API routes, middleware, contracts, persistence, and tests"
	case "go-service":
		return "Go service packages, command entrypoints, adapters, domain logic, and tests"
	case "jvm-service":
		return "JVM service controllers, services, repositories, config, and tests"
	case "python-service":
		return "Python application modules, API surfaces, config, and tests"
	default:
		return "repository source, build config, tests, and public contracts"
	}
}
