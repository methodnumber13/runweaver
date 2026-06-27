package main

import (
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools"
)

func (c cli) indexCmd(args []string) error {
	if len(args) > 0 && args[0] == "clean" {
		return c.indexCleanCmd(args[1:])
	}
	fs := newFlagSet("index")
	repo := fs.String("repo", ".", "repository path")
	changedOnly := fs.Bool("changed-only", true, "reuse content-hash cache for unchanged files")
	prune := fs.Bool("prune", false, "remove stale cache entries after indexing")
	maxCacheMB := fs.Int("max-cache-mb", 256, "maximum cache size to keep when --prune is set; 0 disables size pruning")
	classification := addClassificationFlags(fs, "deterministic")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "index", err: err}
	}
	if err := rejectExtraArgs(fs, "index"); err != nil {
		return err
	}
	classifyOptions, err := classification.options()
	if err != nil {
		return usageError{command: "index", err: err}
	}
	index, err := aitools.IndexWithOptions(*repo, aitools.IndexOptions{ChangedOnly: *changedOnly, Prune: *prune, MaxCacheMB: *maxCacheMB, Classification: classifyOptions})
	if err != nil {
		return fmt.Errorf("index repository: %w", err)
	}
	payload := map[string]any{
		"status":       "success",
		"summary":      fmt.Sprintf("indexed %d files, %d packages, %d symbols", index.Stats.Files, index.Stats.Packages, index.Stats.Symbols),
		"artifacts":    index.Artifacts,
		"stack":        index.Surface.Stack,
		"tools":        index.Tools,
		"cacheHits":    index.Stats.CacheHits,
		"cacheMisses":  index.Stats.CacheMisses,
		"cachePruned":  index.Stats.CachePruned,
		"cacheEntries": index.Stats.CacheEntries,
		"cacheBytes":   index.Stats.CacheBytes,
		"skipped":      index.Stats.Skipped,
		"classifier":   index.ClassifierRun,
		"warnings":     index.Warnings,
	}
	if err := c.printJSON(payload); err != nil {
		return err
	}
	c.printStatus("success", fmt.Sprintf("indexed %d files; cache hits %d, misses %d", index.Stats.Files, index.Stats.CacheHits, index.Stats.CacheMisses))
	if len(index.Warnings) > 0 {
		c.printStatus("warning", fmt.Sprintf("%d index warning(s); see JSON warnings", len(index.Warnings)))
	}
	return nil
}

func (c cli) indexCleanCmd(args []string) error {
	fs := newFlagSet("index clean")
	repo := fs.String("repo", ".", "repository path")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "index clean", err: err}
	}
	if err := rejectExtraArgs(fs, "index clean"); err != nil {
		return err
	}
	result, err := aitools.CleanIndex(*repo)
	if err != nil {
		return fmt.Errorf("clean index: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	if result.Removed {
		c.printStatus("success", "removed "+result.Path)
	} else {
		c.printStatus("success", "index directory already clean")
	}
	return nil
}

func (c cli) scanCmd(args []string) error {
	fs := newFlagSet("scan")
	repo := fs.String("repo", ".", "repository path")
	out := fs.String("out", "", "optional output JSON path")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "scan", err: err}
	}
	if err := rejectExtraArgs(fs, "scan"); err != nil {
		return err
	}
	index, err := aitools.Scan(*repo)
	if err != nil {
		return fmt.Errorf("scan repository: %w", err)
	}
	if *out != "" {
		if err := aitools.WriteJSON(*out, index); err != nil {
			return err
		}
		c.printStatus("success", "surface index written to "+*out)
		return nil
	}
	if err := c.printJSON(index); err != nil {
		return err
	}
	c.printStatus("success", "repository surface scan complete")
	return nil
}
