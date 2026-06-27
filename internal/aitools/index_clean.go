package aitools

import (
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
	"os"
)

// IndexCleanResult reports whether repo-local generated state was removed.
type IndexCleanResult struct {
	Status  string `json:"status"`
	Removed bool   `json:"removed"`
	Path    string `json:"path"`
}

// CleanIndex removes repo-local RunWeaver temporary state.
func CleanIndex(repoPath string) (IndexCleanResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return IndexCleanResult{}, err
	}
	path := statepath.TmpPath(root)
	result := IndexCleanResult{
		Status:  "ok",
		Removed: Exists(path),
		Path:    rel(root, path),
	}
	if err := os.RemoveAll(path); err != nil {
		return IndexCleanResult{}, fmt.Errorf("clean index directory %s: %w", rel(root, path), err)
	}
	return result, nil
}
