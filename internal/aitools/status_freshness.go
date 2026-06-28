package aitools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
)

const indexFreshnessSkew = time.Second

// IndexFreshnessResult summarizes whether the saved compact index predates repo files.
type IndexFreshnessResult struct {
	Status            string   `json:"status"`
	Fresh             bool     `json:"fresh"`
	ManifestPath      string   `json:"manifestPath"`
	GeneratedAt       string   `json:"generatedAt,omitempty"`
	CheckedFiles      int      `json:"checkedFiles"`
	NewestFile        string   `json:"newestFile,omitempty"`
	NewestFileModTime string   `json:"newestFileModTime,omitempty"`
	StaleFiles        []string `json:"staleFiles,omitempty"`
	Warnings          []string `json:"warnings,omitempty"`
}

// CheckIndexFreshness compares the latest index manifest with user-controlled repo files.
func CheckIndexFreshness(repoPath string) IndexFreshnessResult {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return IndexFreshnessResult{
			Status:   "error",
			Fresh:    false,
			Warnings: []string{err.Error()},
		}
	}
	manifestPath := filepath.Join(statepath.IndexRootPath(root), "manifest.json")
	result := IndexFreshnessResult{
		Status:       "missing",
		Fresh:        false,
		ManifestPath: rel(root, manifestPath),
	}
	manifestInfo, err := os.Stat(manifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result
		}
		result.Status = "error"
		result.Warnings = []string{err.Error()}
		return result
	}
	var manifest IndexManifest
	if err := ReadJSON(manifestPath, &manifest); err != nil {
		result.Status = "error"
		result.Warnings = []string{err.Error()}
		return result
	}
	result.GeneratedAt = manifest.GeneratedAt
	result.Status = "ok"
	result.Fresh = true

	checked, newestFile, newestMod, stale, warnings := scanFilesNewerThan(root, manifestInfo.ModTime().Add(indexFreshnessSkew))
	result.CheckedFiles = checked
	result.NewestFile = newestFile
	if !newestMod.IsZero() {
		result.NewestFileModTime = newestMod.UTC().Format(time.RFC3339)
	}
	result.Warnings = warnings
	if len(stale) > 0 {
		result.Status = "stale"
		result.Fresh = false
		result.StaleFiles = Limit(stale, 20)
	}
	return result
}

func scanFilesNewerThan(root string, threshold time.Time) (checked int, newestFile string, newestMod time.Time, staleFiles []string, warnings []string) {
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			warnings = append(warnings, walkErr.Error())
			return nil
		}
		if entry == nil {
			return nil
		}
		if entry.IsDir() {
			if path != root && shouldSkipFreshnessDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			warnings = append(warnings, err.Error())
			return nil
		}
		if info.Mode()&os.ModeType != 0 {
			return nil
		}
		checked++
		modTime := info.ModTime()
		relPath := rel(root, path)
		if modTime.After(newestMod) {
			newestFile = relPath
			newestMod = modTime
		}
		if modTime.After(threshold) {
			staleFiles = append(staleFiles, relPath)
		}
		return nil
	})
	if err != nil {
		warnings = append(warnings, err.Error())
	}
	return checked, newestFile, newestMod, staleFiles, warnings
}

func shouldSkipFreshnessDir(name string) bool {
	if name == statepath.RootDir {
		return true
	}
	if shouldSkipDir(name) {
		return true
	}
	return strings.HasPrefix(name, ".cache")
}
