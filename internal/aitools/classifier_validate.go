package aitools

import (
	"fmt"
	pathpkg "path"
	"path/filepath"
	"strings"
)

// ValidateModelClassification confines AI output to indexed paths and required fields.
func ValidateModelClassification(index RepoIndex, classification RepoClassification) (RepoClassification, []string, error) {
	var warnings []string
	var unsafe []string
	exists := fileSet(index)
	directoryFiles := indexedFilesByDirectory(index)
	validateFiles := func(files []string) []string {
		var out []string
		for _, raw := range files {
			raw = strings.TrimSpace(filepath.ToSlash(raw))
			if raw == "" {
				continue
			}
			if isGeneratedRuntimeMetadataPath(raw) {
				warnings = append(warnings, "dropped generated metadata path from classification: "+raw)
				continue
			}
			if !isSafeRepoPath(raw) {
				unsafe = append(unsafe, raw)
				continue
			}
			file := pathpkg.Clean(raw)
			if !exists[file] {
				if expanded := directoryFiles[file]; len(expanded) > 0 {
					warnings = append(warnings, "expanded classified directory to indexed files: "+raw)
					out = append(out, expanded...)
					continue
				}
				warnings = append(warnings, "dropped non-existent classified file: "+raw)
				continue
			}
			out = append(out, file)
		}
		return Unique(out)
	}
	classification.Entrypoints = sanitizeClassifiedSurfaces(classification.Entrypoints, validateFiles, &warnings, 24)
	classification.Configs = sanitizeClassifiedSurfaces(classification.Configs, validateFiles, &warnings, 32)
	classification.Persistence = sanitizeClassifiedSurfaces(classification.Persistence, validateFiles, &warnings, 32)
	classification.Tests = sanitizeClassifiedSurfaces(classification.Tests, validateFiles, &warnings, 32)
	classification.ExternalSystems = sanitizeClassifiedSurfaces(classification.ExternalSystems, validateFiles, &warnings, 40)
	classification.Domains = sanitizeDomainClassifications(classification.Domains, validateFiles, &warnings, 14)
	classification.Agents = sanitizeAgentProfiles(classification.Agents, validateFiles, &warnings, 12)
	classification.Skills = sanitizeSkillProfiles(classification.Skills, validateFiles, &warnings, 8)
	classification.Verification = Limit(Unique(compactStrings(classification.Verification)), 30)
	classification.Warnings = Unique(append(classification.Warnings, warnings...))
	if len(unsafe) > 0 {
		return classification, warnings, fmt.Errorf("unsafe classified file paths: %s", strings.Join(Unique(unsafe), ", "))
	}
	if !classificationHasContent(classification) {
		return classification, warnings, fmt.Errorf("classification has no useful domains, agents, skills, entrypoints, or surfaces")
	}
	if classification.Summary == "" {
		classification.Summary = classificationSummary(index)
	}
	if len(classification.Warnings) > 0 {
		classification.ValidationStatus = "warning"
	} else {
		classification.ValidationStatus = "valid"
	}
	return classification, warnings, nil
}

func classificationHasContent(classification RepoClassification) bool {
	return len(classification.Domains) > 0 ||
		len(classification.Agents) > 0 ||
		len(classification.Skills) > 0 ||
		len(classification.Entrypoints) > 0 ||
		len(classification.ExternalSystems) > 0 ||
		len(classification.Persistence) > 0 ||
		len(classification.Tests) > 0 ||
		len(classification.Configs) > 0
}

func indexedFilesByDirectory(index RepoIndex) map[string][]string {
	out := map[string][]string{}
	for _, file := range index.Files {
		if file.Generated || file.Path == "" {
			continue
		}
		dir := pathpkg.Dir(filepath.ToSlash(file.Path))
		for dir != "." && dir != "/" && dir != "" {
			out[dir] = append(out[dir], file.Path)
			next := pathpkg.Dir(dir)
			if next == dir {
				break
			}
			dir = next
		}
	}
	for dir, files := range out {
		out[dir] = Unique(files)
	}
	return out
}

func isGeneratedRuntimeMetadataPath(value string) bool {
	clean := pathpkg.Clean(strings.TrimSpace(value))
	for _, prefix := range []string{".opencode", ".codex", ".claude", ".agents", ".runweaver/tmp"} {
		if clean == prefix || strings.HasPrefix(clean, prefix+"/") {
			return true
		}
	}
	return false
}

func isReservedGeneratedAgentName(name string) bool {
	switch name {
	case OpenCodePrimaryAgentName, "swarm", "repo-classifier", "repo-surface-indexer", "agent-skill-drift-reviewer", "profile-regenerator", "stale-anchor-fixer", "repo-surface-engineer", "repo-contract-reviewer", "repo-test-quality-reviewer":
		return true
	default:
		return false
	}
}

func isReservedGeneratedSkillName(name string) bool {
	switch name {
	case "metadata-refresh", "repo-onboarding":
		return true
	default:
		return false
	}
}
