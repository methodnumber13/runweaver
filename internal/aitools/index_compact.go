package aitools

import (
	"fmt"
	"sort"
	"strings"
)

// CompactIndex reduces a full index to the model-facing context payload.
func CompactIndex(index RepoIndex) CompactRepoIndex {
	return CompactRepoIndex{
		SchemaVersion:  1,
		GeneratedAt:    index.GeneratedAt,
		RepoRoot:       index.RepoRoot,
		Stack:          index.Surface.Stack,
		Tools:          index.Tools,
		Packages:       LimitPackages(index.Packages, 120),
		Files:          importantIndexFiles(index.Files, 120),
		Symbols:        LimitSymbols(index.Symbols, 120),
		Routes:         edgesByKind(index.Edges, "declares-route", 120),
		Tests:          edgesByKind(index.Edges, "tests", 120),
		Classification: index.Classification,
		ClassifierRun:  index.ClassifierRun,
		Stats:          index.Stats,
		Artifacts:      index.Artifacts,
		Warnings:       Limit(index.Warnings, 40),
	}
}

func repoContextMarkdown(index RepoIndex, compact CompactRepoIndex) string {
	var b strings.Builder
	b.WriteString("# Repository Context\n\n")
	b.WriteString("Generated: ")
	b.WriteString(index.GeneratedAt)
	b.WriteString("\n\n")
	b.WriteString("## Stack\n")
	writeListLine(&b, "Languages", index.Surface.Stack.Languages)
	writeListLine(&b, "Frameworks", index.Surface.Stack.Frameworks)
	if index.Surface.Stack.PackageManager != "" {
		b.WriteString("- Package manager: ")
		b.WriteString(index.Surface.Stack.PackageManager)
		b.WriteString("\n")
	}
	writeListLine(&b, "Test tools", index.Tools.TestTools)
	writeListLine(&b, "Linters", index.Tools.Linters)
	writeListLine(&b, "Formatters", index.Tools.Formatters)
	writeListLine(&b, "Commands", index.Tools.RecommendedCommands)
	b.WriteString("\n## Counts\n")
	b.WriteString(fmt.Sprintf("- Files: %d\n- Packages: %d\n- Symbols: %d\n- Edges: %d\n- Cache entries: %d\n", index.Stats.Files, index.Stats.Packages, index.Stats.Symbols, index.Stats.Edges, index.Stats.CacheEntries))
	b.WriteString("\n## Package Roles\n")
	if len(compact.Packages) == 0 {
		b.WriteString("- none detected\n")
	} else {
		for _, pkg := range compact.Packages {
			version := ""
			if pkg.Version != "" {
				version = " " + pkg.Version
			}
			b.WriteString(fmt.Sprintf("- %s/%s%s: %s\n", pkg.Ecosystem, pkg.Name, version, pkg.Role))
		}
	}
	b.WriteString("\n## Important Files\n")
	if len(compact.Files) == 0 {
		b.WriteString("- none detected\n")
	} else {
		for _, file := range compact.Files {
			label := file.Category
			if label == "" {
				label = "file"
			}
			b.WriteString(fmt.Sprintf("- %s: %s\n", label, file.Path))
		}
	}
	if len(compact.Routes) > 0 {
		b.WriteString("\n## Routes\n")
		for _, edge := range compact.Routes {
			b.WriteString(fmt.Sprintf("- %s declares %s\n", edge.From, edge.To))
		}
	}
	if len(compact.Tests) > 0 {
		b.WriteString("\n## Test Links\n")
		for _, edge := range compact.Tests {
			b.WriteString(fmt.Sprintf("- %s tests %s\n", edge.From, edge.To))
		}
	}
	if len(index.Classification.Domains) > 0 {
		b.WriteString("\n## Semantic Classification\n")
		b.WriteString("- Source: ")
		b.WriteString(index.Classification.Source)
		b.WriteString("\n")
		b.WriteString("- Validation: ")
		b.WriteString(index.Classification.ValidationStatus)
		b.WriteString("\n")
		for _, domain := range LimitDomains(index.Classification.Domains, 20) {
			b.WriteString(fmt.Sprintf("- domain/%s: %s\n", domain.Name, strings.Join(domain.Files, ", ")))
		}
	}
	if len(index.Warnings) > 0 {
		b.WriteString("\n## Warnings\n")
		for _, warning := range Limit(index.Warnings, 20) {
			b.WriteString("- ")
			b.WriteString(warning)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n## Artifacts\n")
	b.WriteString("- Full index: ")
	b.WriteString(index.Artifacts.Index)
	b.WriteString("\n- Compact index: ")
	b.WriteString(index.Artifacts.Compact)
	b.WriteString("\n- Manifest: ")
	b.WriteString(index.Artifacts.Manifest)
	if index.Artifacts.Classification != "" {
		b.WriteString("\n- Classification: ")
		b.WriteString(index.Artifacts.Classification)
	}
	b.WriteString("\n")
	return b.String()
}

func writeListLine(b *strings.Builder, label string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("- ")
	b.WriteString(label)
	b.WriteString(": ")
	b.WriteString(strings.Join(items, ", "))
	b.WriteString("\n")
}

func importantIndexFiles(files []FileInventoryItem, limit int) []FileInventoryItem {
	priority := map[string]int{
		"config":      0,
		"entrypoint":  1,
		"module":      2,
		"route":       3,
		"contract":    4,
		"service":     5,
		"persistence": 6,
		"ui":          7,
		"test":        8,
		"source":      9,
	}
	candidates := make([]FileInventoryItem, 0, len(files))
	for _, file := range files {
		if file.Generated {
			continue
		}
		if file.Category != "source" || strings.HasPrefix(file.Path, "src/") || strings.HasPrefix(file.Path, "app/") || strings.HasPrefix(file.Path, "cmd/") || strings.HasPrefix(file.Path, "internal/") {
			candidates = append(candidates, file)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := priorityValue(priority, candidates[i].Category)
		right := priorityValue(priority, candidates[j].Category)
		if left == right {
			return candidates[i].Path < candidates[j].Path
		}
		return left < right
	})
	if len(candidates) > limit {
		return candidates[:limit]
	}
	return candidates
}

func priorityValue(priority map[string]int, key string) int {
	if value, ok := priority[key]; ok {
		return value
	}
	return 99
}

func edgesByKind(edges []IndexEdge, kind string, limit int) []IndexEdge {
	var out []IndexEdge
	for _, edge := range edges {
		if edge.Kind == kind {
			out = append(out, edge)
		}
	}
	if len(out) > limit {
		return out[:limit]
	}
	return out
}
