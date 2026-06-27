package modelconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Candidate is a possible configuration path and its source label.
type Candidate struct {
	Path   string
	Source string
}

// ConfigPaths returns project and user-level OpenCode config candidates.
func ConfigPaths(root string) []Candidate {
	seen := map[string]bool{}
	add := func(items *[]Candidate, path, source string) {
		if path == "" {
			return
		}
		path = ExpandHome(path)
		if seen[path] {
			return
		}
		seen[path] = true
		*items = append(*items, Candidate{Path: path, Source: source})
	}
	var out []Candidate
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		add(&out, filepath.Join(xdg, "opencode", "opencode.json"), "global:xdg")
		add(&out, filepath.Join(xdg, "opencode", "opencode.jsonc"), "global:xdg")
	}
	add(&out, "~/.config/opencode/opencode.json", "global")
	add(&out, "~/.config/opencode/opencode.jsonc", "global")
	add(&out, "~/.opencode/config.json", "legacy-global")
	add(&out, "~/.opencode/config.jsonc", "legacy-global")
	if appData := os.Getenv("APPDATA"); appData != "" {
		add(&out, filepath.Join(appData, "opencode", "opencode.json"), "global:appdata")
		add(&out, filepath.Join(appData, "opencode", "opencode.jsonc"), "global:appdata")
	}
	if envPath := os.Getenv("OPENCODE_CONFIG"); envPath != "" {
		add(&out, envPath, "env:OPENCODE_CONFIG")
	}
	add(&out, filepath.Join(root, "opencode.json"), "project")
	if customDir := os.Getenv("OPENCODE_CONFIG_DIR"); customDir != "" {
		add(&out, filepath.Join(customDir, "opencode.json"), "env:OPENCODE_CONFIG_DIR")
		add(&out, filepath.Join(customDir, "opencode.jsonc"), "env:OPENCODE_CONFIG_DIR")
	}
	return out
}

// ManagedConfigPaths returns OS-level managed OpenCode config candidates.
func ManagedConfigPaths() []Candidate {
	seen := map[string]bool{}
	add := func(items *[]Candidate, path, source string) {
		if path == "" {
			return
		}
		path = ExpandHome(path)
		if seen[path] {
			return
		}
		seen[path] = true
		*items = append(*items, Candidate{Path: path, Source: source})
	}
	var out []Candidate
	if runtime.GOOS == "darwin" {
		add(&out, "/Library/Application Support/opencode/opencode.json", "managed")
		add(&out, "/Library/Application Support/opencode/opencode.jsonc", "managed")
	}
	if runtime.GOOS == "linux" {
		add(&out, "/etc/opencode/opencode.json", "managed")
		add(&out, "/etc/opencode/opencode.jsonc", "managed")
	}
	if programData := os.Getenv("ProgramData"); programData != "" {
		add(&out, filepath.Join(programData, "opencode", "opencode.json"), "managed")
		add(&out, filepath.Join(programData, "opencode", "opencode.jsonc"), "managed")
	}
	return out
}

// AuthFilePaths returns user-level OpenCode auth file candidates.
func AuthFilePaths() []string {
	seen := map[string]bool{}
	add := func(out *[]string, path string) {
		path = ExpandHome(path)
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		*out = append(*out, path)
	}
	var out []string
	if data := os.Getenv("XDG_DATA_HOME"); data != "" {
		add(&out, filepath.Join(data, "opencode", "auth.json"))
	}
	add(&out, "~/.local/share/opencode/auth.json")
	add(&out, "~/.opencode/auth.json")
	if appData := os.Getenv("APPDATA"); appData != "" {
		add(&out, filepath.Join(appData, "opencode", "auth.json"))
	}
	return out
}

// ExpandHome expands a leading ~ using the current user's home directory.
func ExpandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}
