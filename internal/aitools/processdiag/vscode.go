package processdiag

import (
	"encoding/json"
	"github.com/methodnumber13/runweaver/internal/aitools/jsonc"
	"os"
	"path/filepath"
	"strings"
)

type vscodeDebugSettings struct {
	Path                 string
	HasAutoAttachFilter  bool
	AutoAttachFilter     string
	LegacyNodeAutoAttach string
	SmartPatternCount    int
}

func detectVSCodeDiagnostics(processes []ProcessInfo, settings vscodeDebugSettings) VSCodeDiagnostics {
	out := VSCodeDiagnostics{
		SettingsPath:             settings.Path,
		HasAutoAttachSetting:     settings.HasAutoAttachFilter,
		AutoAttachFilter:         settings.AutoAttachFilter,
		LegacyNodeAutoAttach:     settings.LegacyNodeAutoAttach,
		SmartPatternCount:        settings.SmartPatternCount,
		AutoAttachRecommendation: "",
	}
	for _, proc := range processes {
		switch proc.Kind {
		case "vscode":
			out.HelperProcessCount++
		case "vscode-debugger":
			out.DebuggerProcessCount++
		case "mcp", "node", "npm":
			out.NodeLikeProcessCount++
		}
	}
	out.Detected = out.HelperProcessCount > 0 || out.DebuggerProcessCount > 0
	if out.Detected {
		out.Notes = append(out.Notes, "VS Code JavaScript debugger process picker lists Node.js processes; this is UI/debugger visibility, not a separate swarm worker model.")
	}
	if out.NodeLikeProcessCount > 0 {
		out.Notes = append(out.Notes, "Codex/OpenCode MCP servers are Node.js processes, so VS Code Auto Attach can make them appear in the debugger process list.")
	}
	if out.Detected && strings.EqualFold(out.AutoAttachFilter, "disabled") {
		return out
	}
	if out.Detected && (out.NodeLikeProcessCount > 10 || out.DebuggerProcessCount > 0) {
		out.AutoAttachRecommendation = "If VS Code debugger is noisy, run 'Debug: Toggle Auto Attach' and choose Off, or set debug.javascript.autoAttachFilter to disabled."
	}
	return out
}

func isVSCodeDebuggerCommand(command string) bool {
	lower := strings.ToLower(command)
	return strings.Contains(lower, "js-debug") ||
		strings.Contains(lower, "vscode-js-debug") ||
		strings.Contains(lower, "debugservermain.js") ||
		strings.Contains(lower, "bootloader.js") && strings.Contains(lower, "debug")
}

func isVSCodeHelperCommand(command string) bool {
	lower := strings.ToLower(command)
	return strings.Contains(lower, "visual studio code.app") ||
		strings.Contains(lower, "code helper") ||
		strings.Contains(lower, "vscode.app") ||
		strings.Contains(lower, "vscodium")
}

func readVSCodeDebugSettings() vscodeDebugSettings {
	for _, candidate := range vscodeSettingsCandidates() {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		settings, err := parseVSCodeDebugSettings(data)
		if err != nil {
			return vscodeDebugSettings{Path: candidate}
		}
		settings.Path = candidate
		return settings
	}
	return vscodeDebugSettings{}
}

func vscodeSettingsCandidates() []string {
	var out []string
	if appData := os.Getenv("APPDATA"); appData != "" {
		out = append(out, filepath.Join(appData, "Code", "User", "settings.json"))
		out = append(out, filepath.Join(appData, "Cursor", "User", "settings.json"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out,
			filepath.Join(home, "Library", "Application Support", "Code", "User", "settings.json"),
			filepath.Join(home, "Library", "Application Support", "Code - Insiders", "User", "settings.json"),
			filepath.Join(home, "Library", "Application Support", "Cursor", "User", "settings.json"),
			filepath.Join(home, ".config", "Code", "User", "settings.json"),
			filepath.Join(home, ".config", "Code - Insiders", "User", "settings.json"),
			filepath.Join(home, ".config", "Cursor", "User", "settings.json"),
		)
	}
	return out
}

func parseVSCodeDebugSettings(data []byte) (vscodeDebugSettings, error) {
	var parsed map[string]any
	if err := json.Unmarshal(jsonc.StripComments(data), &parsed); err != nil {
		return vscodeDebugSettings{}, err
	}
	settings := vscodeDebugSettings{}
	if value, ok := parsed["debug.javascript.autoAttachFilter"].(string); ok {
		settings.HasAutoAttachFilter = true
		settings.AutoAttachFilter = value
	}
	if value, ok := parsed["debug.node.autoAttach"].(string); ok {
		settings.LegacyNodeAutoAttach = value
	}
	if values, ok := parsed["debug.javascript.autoAttachSmartPattern"].([]any); ok {
		settings.SmartPatternCount = len(values)
	}
	return settings, nil
}
