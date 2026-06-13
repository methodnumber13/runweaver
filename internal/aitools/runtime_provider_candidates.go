package aitools

import (
	"github.com/methodnumber13/runweaver/internal/aitools/modelconfig"
	"os"
	"path/filepath"
	"runtime"
)

func runtimeOpenCodeConfigCandidates(root string) []configCandidate {
	out := modelconfig.ConfigPaths(root)
	out = append(out,
		configCandidate{Path: filepath.Join(root, "opencode.jsonc"), Source: "project"},
		configCandidate{Path: filepath.Join(root, ".opencode", "opencode.json"), Source: "project"},
		configCandidate{Path: filepath.Join(root, ".opencode", "opencode.jsonc"), Source: "project"},
	)
	if os.Getenv("OPENCODE_CONFIG_CONTENT") != "" {
		out = append(out, configCandidate{Path: "env:OPENCODE_CONFIG_CONTENT", Source: "env"})
	}
	return uniqueConfigCandidates(out)
}

func runtimeOpenCodeAuthCandidates() []configCandidate {
	items := modelconfig.AuthFilePaths()
	out := make([]configCandidate, 0, len(items))
	for _, path := range items {
		out = append(out, configCandidate{Path: path, Source: "auth"})
	}
	return uniqueConfigCandidates(out)
}

func runtimeOpenCodeManagedCandidates() []configCandidate {
	return uniqueConfigCandidates(modelconfig.ManagedConfigPaths())
}

func runtimeCodexConfigCandidates(root string) []configCandidate {
	var out []configCandidate
	addConfigCandidate(&out, filepath.Join(root, ".codex", "config.toml"), "project")
	addConfigCandidate(&out, filepath.Join(root, ".codex", "requirements.toml"), "project")
	if home := os.Getenv("CODEX_HOME"); home != "" {
		addConfigCandidate(&out, filepath.Join(home, "config.toml"), "env:CODEX_HOME")
		addConfigCandidate(&out, filepath.Join(home, "AGENTS.md"), "env:CODEX_HOME")
	}
	addConfigCandidate(&out, "~/.codex/config.toml", "global")
	addConfigCandidate(&out, "~/.codex/AGENTS.md", "global")
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		addConfigCandidate(&out, filepath.Join(xdg, "codex", "config.toml"), "global:xdg")
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		addConfigCandidate(&out, filepath.Join(appData, "codex", "config.toml"), "global:appdata")
	}
	return uniqueConfigCandidates(out)
}

func runtimeCodexAuthCandidates() []configCandidate {
	var out []configCandidate
	if home := os.Getenv("CODEX_HOME"); home != "" {
		addConfigCandidate(&out, filepath.Join(home, "auth.json"), "env:CODEX_HOME")
	}
	addConfigCandidate(&out, "~/.codex/auth.json", "global")
	return uniqueConfigCandidates(out)
}

func runtimeCodexManagedCandidates() []configCandidate {
	var out []configCandidate
	if runtime.GOOS == "darwin" {
		addConfigCandidate(&out, "/Library/Application Support/Codex/config.toml", "managed")
	}
	if runtime.GOOS == "linux" {
		addConfigCandidate(&out, "/etc/codex/config.toml", "managed")
		addConfigCandidate(&out, "/etc/codex/skills", "managed")
	}
	if programData := os.Getenv("ProgramData"); programData != "" {
		addConfigCandidate(&out, filepath.Join(programData, "Codex", "config.toml"), "managed")
	}
	return uniqueConfigCandidates(out)
}

func runtimeClaudeConfigCandidates(root string) []configCandidate {
	var out []configCandidate
	addConfigCandidate(&out, filepath.Join(root, ".claude", "settings.json"), "project")
	addConfigCandidate(&out, filepath.Join(root, ".claude", "settings.local.json"), "local")
	addConfigCandidate(&out, filepath.Join(root, ".mcp.json"), "project")
	claudeHome := "~/.claude"
	if custom := os.Getenv("CLAUDE_CONFIG_DIR"); custom != "" {
		claudeHome = custom
	}
	addConfigCandidate(&out, filepath.Join(claudeHome, "settings.json"), "global")
	addConfigCandidate(&out, filepath.Join(claudeHome, "CLAUDE.md"), "global")
	addConfigCandidate(&out, filepath.Join(claudeHome, ".claude.json"), "legacy-global")
	addConfigCandidate(&out, "~/.claude.json", "global:mcp")
	if appData := os.Getenv("APPDATA"); appData != "" {
		addConfigCandidate(&out, filepath.Join(appData, "ClaudeCode", "settings.json"), "global:appdata")
	}
	return uniqueConfigCandidates(out)
}

func runtimeClaudeAuthCandidates() []configCandidate {
	var out []configCandidate
	claudeHome := "~/.claude"
	if custom := os.Getenv("CLAUDE_CONFIG_DIR"); custom != "" {
		claudeHome = custom
	}
	addConfigCandidate(&out, filepath.Join(claudeHome, "auth.json"), "global")
	addConfigCandidate(&out, filepath.Join(claudeHome, ".credentials.json"), "global")
	return uniqueConfigCandidates(out)
}

func runtimeClaudeManagedCandidates() []configCandidate {
	var out []configCandidate
	if runtime.GOOS == "darwin" {
		addConfigCandidate(&out, "/Library/Application Support/ClaudeCode/managed-settings.json", "managed")
		addConfigCandidate(&out, "/Library/Application Support/ClaudeCode/managed-mcp.json", "managed")
		addConfigCandidate(&out, "/Library/Application Support/ClaudeCode/managed-settings.d", "managed")
	}
	if runtime.GOOS == "linux" {
		addConfigCandidate(&out, "/etc/claude-code/managed-settings.json", "managed")
		addConfigCandidate(&out, "/etc/claude-code/managed-mcp.json", "managed")
		addConfigCandidate(&out, "/etc/claude-code/managed-settings.d", "managed")
	}
	if programData := os.Getenv("ProgramData"); programData != "" {
		addConfigCandidate(&out, filepath.Join(programData, "ClaudeCode", "managed-settings.json"), "managed:legacy")
	}
	return uniqueConfigCandidates(out)
}
