package aitools

import (
	pathpkg "path"
	"path/filepath"
	"strings"
)

func sanitizeClassifiedSurfaces(items []ClassifiedSurface, validateFiles func([]string) []string, warnings *[]string, limit int) []ClassifiedSurface {
	var out []ClassifiedSurface
	for _, item := range items {
		item.Name = sanitizeReadableName(item.Name)
		if item.Name == "" {
			*warnings = append(*warnings, "dropped classified surface with empty name")
			continue
		}
		item.Kind = sanitizeReadableName(item.Kind)
		item.Files = Limit(validateFiles(item.Files), 80)
		item.Evidence = Limit(Unique(compactStrings(item.Evidence)), 20)
		item.Confidence = sanitizeConfidence(item.Confidence)
		out = append(out, item)
	}
	return limitClassifiedSurfaces(out, limit)
}

func sanitizeDomainClassifications(items []DomainClassification, validateFiles func([]string) []string, warnings *[]string, limit int) []DomainClassification {
	var out []DomainClassification
	for _, item := range items {
		item.Name = sanitizeID(item.Name)
		if item.Name == "" {
			*warnings = append(*warnings, "dropped domain with empty name")
			continue
		}
		item.Kind = sanitizeReadableName(item.Kind)
		item.Files = Limit(validateFiles(item.Files), 80)
		item.Evidence = Limit(Unique(compactStrings(item.Evidence)), 20)
		item.Confidence = sanitizeConfidence(item.Confidence)
		out = append(out, item)
	}
	return LimitDomains(out, limit)
}

func sanitizeAgentProfiles(items []AgentProfile, validateFiles func([]string) []string, warnings *[]string, limit int) []AgentProfile {
	var out []AgentProfile
	seen := map[string]bool{}
	for _, item := range items {
		item.Name = sanitizeID(item.Name)
		if item.Name == "" || seen[item.Name] {
			if item.Name == "" {
				*warnings = append(*warnings, "dropped agent with empty name")
			}
			continue
		}
		if isReservedGeneratedAgentName(item.Name) {
			continue
		}
		seen[item.Name] = true
		item.FocusFiles = Limit(validateFiles(item.FocusFiles), 120)
		item.Workflow = Limit(Unique(compactStrings(item.Workflow)), 12)
		item.Verification = Limit(Unique(compactStrings(item.Verification)), 20)
		out = append(out, item)
	}
	return limitAgentProfiles(out, limit)
}

func sanitizeSkillProfiles(items []SkillProfile, validateFiles func([]string) []string, warnings *[]string, limit int) []SkillProfile {
	var out []SkillProfile
	seen := map[string]bool{}
	for _, item := range items {
		item.Name = sanitizeID(item.Name)
		if item.Name == "" || seen[item.Name] {
			if item.Name == "" {
				*warnings = append(*warnings, "dropped skill with empty name")
			}
			continue
		}
		if isReservedGeneratedSkillName(item.Name) {
			continue
		}
		seen[item.Name] = true
		item.FocusFiles = Limit(validateFiles(item.FocusFiles), 120)
		item.Workflow = Limit(Unique(compactStrings(item.Workflow)), 12)
		item.Risks = Limit(Unique(compactStrings(item.Risks)), 12)
		item.Verification = Limit(Unique(compactStrings(item.Verification)), 20)
		out = append(out, item)
	}
	return limitSkillProfiles(out, limit)
}

func isSafeRepoPath(value string) bool {
	if value == "" || strings.HasPrefix(value, "/") || filepath.IsAbs(value) {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(value), "/") {
		if part == ".." {
			return false
		}
	}
	clean := pathpkg.Clean(value)
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return false
	}
	return true
}

func sanitizeReadableName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.Join(strings.Fields(value), " ")
}

func limitClassifiedSurfaces(items []ClassifiedSurface, n int) []ClassifiedSurface {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func limitAgentProfiles(items []AgentProfile, n int) []AgentProfile {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func limitSkillProfiles(items []SkillProfile, n int) []SkillProfile {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func sanitizeConfidence(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "low", "medium", "high":
		return value
	default:
		return ""
	}
}
