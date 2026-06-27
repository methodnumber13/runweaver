package aitools

import (
	"fmt"
)

// LimitSymbols returns at most n symbols while preserving order.
func LimitSymbols(items []SymbolInfo, n int) []SymbolInfo {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

// LimitEdges returns at most n index edges while preserving order.
func LimitEdges(items []IndexEdge, n int) []IndexEdge {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

// LimitPackages returns at most n package insights while preserving order.
func LimitPackages(items []PackageInsight, n int) []PackageInsight {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func limitRoutes(items []RouteInfo, n int) []RouteInfo {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func indexSummary(index RepoIndex) map[string]any {
	return map[string]any{
		"status":        "success",
		"summary":       fmt.Sprintf("indexed %d files, %d packages, %d symbols", index.Stats.Files, index.Stats.Packages, index.Stats.Symbols),
		"next_actions":  []string{"review .runweaver/tmp/index/repo-context.md", "open repo-index.json only when compact context is insufficient", "run runweaver refresh --repo . --apply after approving generated metadata"},
		"artifacts":     index.Artifacts,
		"stack":         index.Surface.Stack,
		"tools":         index.Tools,
		"cacheHits":     index.Stats.CacheHits,
		"cacheMisses":   index.Stats.CacheMisses,
		"schemaVersion": index.SchemaVersion,
	}
}
