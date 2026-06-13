package aitools

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type cacheEntry struct {
	Path    string
	Hash    string
	Size    int64
	ModUnix int64
}

func pruneIndexCache(cacheRoot string, live map[string]bool, maxCacheMB int) (int, []string) {
	entries, warning := listCacheEntries(cacheRoot)
	var warnings []string
	if warning != "" {
		warnings = appendIndexWarning(warnings, warning)
	}
	pruned := 0
	for _, entry := range entries {
		if live[entry.Hash] {
			continue
		}
		if err := os.Remove(entry.Path); err != nil {
			warnings = appendIndexWarning(warnings, fmt.Sprintf("cache prune failed for %s: %v", filepath.Base(entry.Path), err))
			continue
		}
		pruned++
	}
	if maxCacheMB <= 0 {
		return pruned, warnings
	}
	entries, warning = listCacheEntries(cacheRoot)
	if warning != "" {
		warnings = appendIndexWarning(warnings, warning)
	}
	maxBytes := int64(maxCacheMB) * 1024 * 1024
	total := int64(0)
	for _, entry := range entries {
		total += entry.Size
	}
	if total <= maxBytes {
		return pruned, warnings
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].ModUnix == entries[j].ModUnix {
			return entries[i].Hash < entries[j].Hash
		}
		return entries[i].ModUnix < entries[j].ModUnix
	})
	for _, entry := range entries {
		if total <= maxBytes {
			break
		}
		if err := os.Remove(entry.Path); err != nil {
			warnings = appendIndexWarning(warnings, fmt.Sprintf("cache size prune failed for %s: %v", filepath.Base(entry.Path), err))
			continue
		}
		total -= entry.Size
		pruned++
	}
	return pruned, warnings
}

func cacheStats(cacheRoot string) (int, int64, string) {
	entries, warning := listCacheEntries(cacheRoot)
	total := int64(0)
	for _, entry := range entries {
		total += entry.Size
	}
	return len(entries), total, warning
}

func listCacheEntries(cacheRoot string) ([]cacheEntry, string) {
	items, err := os.ReadDir(cacheRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ""
		}
		return nil, fmt.Sprintf("cache directory read failed: %v", err)
	}
	var entries []cacheEntry
	for _, item := range items {
		if item.IsDir() || filepath.Ext(item.Name()) != ".json" {
			continue
		}
		info, err := item.Info()
		if err != nil {
			return entries, fmt.Sprintf("cache file stat failed for %s: %v", item.Name(), err)
		}
		hash := strings.TrimSuffix(item.Name(), ".json")
		entries = append(entries, cacheEntry{
			Path:    filepath.Join(cacheRoot, item.Name()),
			Hash:    hash,
			Size:    info.Size(),
			ModUnix: info.ModTime().Unix(),
		})
	}
	return entries, ""
}

func reanchorAnalysis(analysis FileAnalysis, path string) FileAnalysis {
	for i := range analysis.Symbols {
		analysis.Symbols[i].Path = path
	}
	for i := range analysis.Routes {
		analysis.Routes[i].File = path
	}
	return analysis
}

func readFileAnalysis(cacheRoot, hash string) (FileAnalysis, bool) {
	var analysis FileAnalysis
	if err := ReadJSON(filepath.Join(cacheRoot, hash+".json"), &analysis); err != nil {
		return FileAnalysis{}, false
	}
	if analysis.SchemaVersion != fileAnalysisSchemaVersion {
		return FileAnalysis{}, false
	}
	return analysis, true
}

func writeFileAnalysis(cacheRoot string, analysis FileAnalysis) error {
	return WriteJSON(filepath.Join(cacheRoot, analysis.Hash+".json"), analysis)
}
