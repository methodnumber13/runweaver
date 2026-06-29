package foundation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// WriteJSON writes indented JSON and creates the parent directory.
func WriteJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %s: %w", path, err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON for %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write JSON %s: %w", path, err)
	}
	return nil
}

// ReadJSON reads JSON into value and wraps path-aware parse errors.
func ReadJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read JSON %s: %w", path, err)
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("parse JSON %s: %w", path, err)
	}
	return nil
}

// SafeJSON returns indented JSON without panicking on marshal errors.
func SafeJSON(value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal JSON: %w", err)
	}
	return string(append(data, '\n')), nil
}

// Now returns the current UTC timestamp in RFC3339 format.
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// Unique trims, slash-normalizes, sorts, and de-duplicates strings.
func Unique(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = filepath.ToSlash(strings.TrimSpace(item))
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

// Limit returns at most n items while preserving order.
func Limit(items []string, n int) []string {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

// RepoRoot resolves and validates a repository directory path.
func RepoRoot(path string) (string, error) {
	if path == "" {
		path = "."
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve repository path %q: %w", path, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("repository path %q does not exist", abs)
		}
		return "", fmt.Errorf("inspect repository path %q: %w", abs, err)
	}
	if !info.IsDir() {
		return "", errors.New("repo path must be a directory")
	}
	return abs, nil
}

// Exists reports whether path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Rel returns a slash-normalized path relative to root when possible.
func Rel(root, path string) string {
	value, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(value)
}

// ShouldSkipDir reports whether RunWeaver should ignore a generated/vendor directory.
func ShouldSkipDir(name string) bool {
	switch name {
	case ".git", ".opencode", ".codex", ".claude", ".agents", "node_modules", "build", "dist", "coverage", "reports", "target", ".gradle", ".idea", ".vscode", "vendor":
		return true
	default:
		return strings.HasPrefix(name, ".cache")
	}
}

// ShouldSkipWalkDir reports whether a directory should be excluded while walking a repository.
// It keeps stable RunWeaver metadata visible while excluding volatile runtime state.
func ShouldSkipWalkDir(root, path string, name string) bool {
	relPath := Rel(root, path)
	if relPath == ".runweaver/tmp" || strings.HasPrefix(relPath, ".runweaver/tmp/") {
		return true
	}
	return ShouldSkipDir(name)
}
