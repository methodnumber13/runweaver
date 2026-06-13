package aitools

import (
	"path/filepath"
	"strings"
)

func missingMandatoryCoverage(index RepoIndex, classification RepoClassification) []string {
	covered := map[string]bool{}
	add := func(files []string) {
		for _, file := range files {
			file = filepath.ToSlash(strings.TrimSpace(file))
			if file != "" {
				covered[file] = true
			}
		}
	}
	for _, item := range classification.Entrypoints {
		add(item.Files)
	}
	for _, item := range classification.Configs {
		add(item.Files)
	}
	for _, item := range classification.Persistence {
		add(item.Files)
	}
	for _, item := range classification.Tests {
		add(item.Files)
	}
	for _, item := range classification.ExternalSystems {
		add(item.Files)
	}
	for _, domain := range classification.Domains {
		add(domain.Files)
	}
	for _, agent := range classification.Agents {
		add(agent.FocusFiles)
	}
	for _, skill := range classification.Skills {
		add(skill.FocusFiles)
	}
	var missing []string
	for _, file := range mandatoryCoverageForClassifier(index) {
		if !covered[file] {
			missing = append(missing, file)
		}
	}
	return Limit(Unique(missing), 24)
}

func hasPackage(index RepoIndex, name string) bool {
	for _, pkg := range index.Packages {
		if pkg.Name == name {
			return true
		}
	}
	return false
}

func mandatoryCoverageForClassifier(index RepoIndex) []string {
	var files []string
	files = append(files, filesByCategory(index, "entrypoint")...)
	for _, file := range Limit(filesByCategory(index, "route"), 24) {
		if shouldSkipMandatoryCoverageFile(file) {
			continue
		}
		files = append(files, file)
	}
	for _, file := range index.Surface.ConfigFiles {
		if shouldSkipMandatoryCoverageFile(file) {
			continue
		}
		files = append(files, file)
	}
	files = append(files, filterFiles(index, "test/jest-")...)
	return Limit(Unique(files), 48)
}

func shouldSkipMandatoryCoverageFile(file string) bool {
	file = filepath.ToSlash(file)
	lower := strings.ToLower(file)
	base := strings.ToLower(filepath.Base(file))
	if isGeneratedRuntimeMetadataPath(lower) || strings.HasPrefix(lower, "helm/") {
		return true
	}
	if lower == "src/app.controller.ts" || lower == "src/app.service.ts" {
		return true
	}
	switch base {
	case "readme.md", "license", "license.md":
		return true
	default:
		return false
	}
}

func importantPackagesForClassifier(packages []PackageInsight, limit int) []PackageInsight {
	var important []PackageInsight
	var context []PackageInsight
	for _, pkg := range packages {
		if pkg.Role == "" || pkg.Role == "library" {
			context = append(context, pkg)
			continue
		}
		important = append(important, pkg)
	}
	out := append(important, context...)
	return LimitPackages(out, limit)
}
