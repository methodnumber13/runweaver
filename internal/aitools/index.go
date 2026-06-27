package aitools

import (
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
	"os"
	"path/filepath"
	"strings"
)

const hashChunkSize = 64 * 1024

const fileAnalysisSchemaVersion = 2

// Index builds the repo-local index and writes canonical artifacts.
func Index(repoPath string, changedOnly bool) (RepoIndex, error) {
	return IndexWithOptions(repoPath, IndexOptions{ChangedOnly: changedOnly})
}

// IndexWithOptions builds the repo index with cache and classifier controls.
func IndexWithOptions(repoPath string, opts IndexOptions) (RepoIndex, error) {
	return indexWithOptions(repoPath, opts, runCommandOutputWithEnv)
}

func indexWithOptions(repoPath string, opts IndexOptions, runner outputRunner) (RepoIndex, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return RepoIndex{}, err
	}
	files, warnings, err := repoFiles(root)
	if err != nil {
		return RepoIndex{}, err
	}
	surface, err := Scan(root)
	if err != nil {
		return RepoIndex{}, err
	}

	indexRoot := statepath.IndexRootPath(root)
	cacheRoot := statepath.CacheRootPath(root)
	if err := os.MkdirAll(indexRoot, 0o755); err != nil {
		return RepoIndex{}, err
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return RepoIndex{}, err
	}

	var inventory []FileInventoryItem
	var symbols []SymbolInfo
	var edges []IndexEdge
	liveCacheHashes := map[string]bool{}
	stats := IndexStats{}

	for _, file := range files {
		abs := filepath.Join(root, file)
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			if err != nil {
				stats.Skipped++
				warnings = appendIndexWarning(warnings, fmt.Sprintf("skipped %s: %v", file, err))
			}
			continue
		}
		hash, err := computeFileHash(abs)
		if err != nil {
			stats.Skipped++
			warnings = appendIndexWarning(warnings, fmt.Sprintf("skipped %s: cannot hash file: %v", file, err))
			continue
		}
		language := languageFor(file)
		category := categoryFor(file)
		inventory = append(inventory, FileInventoryItem{
			Path:      file,
			Size:      info.Size(),
			ModTime:   info.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
			Hash:      hash,
			Language:  language,
			Category:  category,
			Generated: isGeneratedFile(file),
		})
		liveCacheHashes[hash] = true
		analysis, hit := readFileAnalysis(cacheRoot, hash)
		if !hit || !opts.ChangedOnly {
			if !hit {
				stats.CacheMisses++
			}
			analysis = analyzeFile(abs, file, hash, language, category)
			if err := writeFileAnalysis(cacheRoot, analysis); err != nil {
				warnings = appendIndexWarning(warnings, fmt.Sprintf("cache write failed for %s: %v", file, err))
			}
		} else {
			stats.CacheHits++
			analysis.SourcePaths = Unique(append(analysis.SourcePaths, file))
			analysis = reanchorAnalysis(analysis, file)
			if err := writeFileAnalysis(cacheRoot, analysis); err != nil {
				warnings = appendIndexWarning(warnings, fmt.Sprintf("cache reanchor write failed for %s: %v", file, err))
			}
		}
		symbols = append(symbols, analysis.Symbols...)
		for _, imp := range analysis.Imports {
			edges = append(edges, IndexEdge{From: file, To: imp, Kind: "imports"})
		}
		for _, route := range analysis.Routes {
			name := route.Method + " " + route.Path
			if strings.TrimSpace(name) == "" {
				name = "route"
			}
			edges = append(edges, IndexEdge{From: file, To: name, Kind: "declares-route"})
		}
	}
	edges = append(edges, testEdges(inventory)...)
	packages, packageWarnings := DetectPackagesWithWarnings(root, files)
	warnings = append(warnings, packageWarnings...)
	tools := BuildToolchain(surface.Stack, packages, surface.BuildCommands)
	surface.Packages = packages
	surface.Tools = tools
	if opts.Prune {
		pruned, pruneWarnings := pruneIndexCache(cacheRoot, liveCacheHashes, opts.MaxCacheMB)
		stats.CachePruned = pruned
		warnings = append(warnings, pruneWarnings...)
	}
	cacheEntries, cacheBytes, cacheWarning := cacheStats(cacheRoot)
	stats.CacheEntries = cacheEntries
	stats.CacheBytes = cacheBytes
	if cacheWarning != "" {
		warnings = appendIndexWarning(warnings, cacheWarning)
	}
	stats.Files = len(inventory)
	stats.Packages = len(packages)
	stats.Symbols = len(symbols)
	stats.Edges = len(edges)

	result := RepoIndex{
		SchemaVersion: 1,
		GeneratedAt:   Now(),
		RepoRoot:      root,
		Surface:       surface,
		Tools:         tools,
		Packages:      packages,
		Files:         inventory,
		Symbols:       LimitSymbols(symbols, 400),
		Edges:         LimitEdges(edges, 800),
		Stats:         stats,
		Artifacts: IndexArtifacts{
			Root:             rel(root, statepath.TmpPath(root)),
			Files:            rel(root, filepath.Join(indexRoot, "files.json")),
			Packages:         rel(root, filepath.Join(indexRoot, "packages.json")),
			Symbols:          rel(root, filepath.Join(indexRoot, "symbols.json")),
			Edges:            rel(root, filepath.Join(indexRoot, "edges.json")),
			Index:            rel(root, filepath.Join(indexRoot, "repo-index.json")),
			Compact:          rel(root, filepath.Join(indexRoot, "repo-index.compact.json")),
			Context:          rel(root, filepath.Join(indexRoot, "repo-context.md")),
			Manifest:         rel(root, filepath.Join(indexRoot, "manifest.json")),
			Classification:   rel(root, filepath.Join(indexRoot, "repo-classification.json")),
			ClassifierPrompt: rel(root, filepath.Join(indexRoot, "repo-classifier-prompt.md")),
			ClassifierOutput: rel(root, filepath.Join(indexRoot, "repo-classifier-output.json")),
		},
		Warnings: Limit(warnings, 80),
	}
	classification, classifierRun, err := classifyRepoIndex(root, result, opts.Classification, runner)
	if err != nil {
		return RepoIndex{}, err
	}
	result.Classification = classification
	result.ClassifierRun = &classifierRun
	result.Warnings = Limit(Unique(append(result.Warnings, classifierRun.Warnings...)), 100)
	compact := CompactIndex(result)
	manifest := IndexManifest{
		SchemaVersion:    1,
		GeneratedAt:      result.GeneratedAt,
		RepoRoot:         root,
		Artifacts:        result.Artifacts,
		Stats:            result.Stats,
		LiveCacheEntries: len(liveCacheHashes),
		Warnings:         result.Warnings,
	}
	if err := WriteJSON(filepath.Join(indexRoot, "files.json"), inventory); err != nil {
		return RepoIndex{}, err
	}
	if err := WriteJSON(filepath.Join(indexRoot, "packages.json"), packages); err != nil {
		return RepoIndex{}, err
	}
	if err := WriteJSON(filepath.Join(indexRoot, "symbols.json"), result.Symbols); err != nil {
		return RepoIndex{}, err
	}
	if err := WriteJSON(filepath.Join(indexRoot, "edges.json"), result.Edges); err != nil {
		return RepoIndex{}, err
	}
	if err := WriteJSON(filepath.Join(indexRoot, "repo-classification.json"), result.Classification); err != nil {
		return RepoIndex{}, err
	}
	if err := WriteJSON(filepath.Join(indexRoot, "repo-index.json"), result); err != nil {
		return RepoIndex{}, err
	}
	if err := WriteJSON(filepath.Join(indexRoot, "repo-index.compact.json"), compact); err != nil {
		return RepoIndex{}, err
	}
	if err := os.WriteFile(filepath.Join(indexRoot, "repo-context.md"), []byte(repoContextMarkdown(result, compact)), 0o644); err != nil {
		return RepoIndex{}, fmt.Errorf("write repo context: %w", err)
	}
	if err := WriteJSON(filepath.Join(indexRoot, "manifest.json"), manifest); err != nil {
		return RepoIndex{}, err
	}
	return result, nil
}
