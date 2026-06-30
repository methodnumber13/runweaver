package processdiag

import (
	"strings"
	"testing"
)

func TestDoctorProcessesFromPSOutputDetectsVSCodeAndDuplicateMCP(t *testing.T) {
	result := DoctorProcessesFromPSOutput(`
100 1 01:00:00 codex --yolo
101 100 59:59 npm exec @upstash/context7-mcp -- --api-key redacted
102 100 59:58 node /tmp/node_modules/.bin/context7-mcp --api-key redacted
200 1 00:05:00 /Applications/Visual Studio Code.app/Contents/Frameworks/Code Helper.app/Contents/MacOS/Code Helper --type=utility
201 200 00:04:59 /Applications/Visual Studio Code.app/Contents/Frameworks/Code Helper.app/Contents/MacOS/Code Helper /Users/me/.vscode/extensions/ms-vscode.js-debug/src/bootloader.js
300 1 00:03:00 opencode run --agent runweaver-swarm
`)

	if result.Status != "warning" {
		t.Fatalf("status = %q, want warning", result.Status)
	}
	if len(result.Supervisors) != 2 {
		t.Fatalf("supervisors = %d, want 2", len(result.Supervisors))
	}
	if len(result.Duplicates) != 1 || result.Duplicates[0].Command != "@upstash/context7-mcp" {
		t.Fatalf("duplicates = %#v, want one context7 group", result.Duplicates)
	}
	if !result.VSCode.Detected || result.VSCode.HelperProcessCount != 1 || result.VSCode.DebuggerProcessCount != 1 {
		t.Fatalf("vscode = %#v, want helper/debugger detection", result.VSCode)
	}
	if result.VSCode.AutoAttachRecommendation == "" {
		t.Fatalf("vscode recommendation is empty")
	}
	if !strings.Contains(strings.Join(result.Recommendations, "\n"), "Auto Attach") {
		t.Fatalf("recommendations = %#v, want Auto Attach guidance", result.Recommendations)
	}
}

func TestParseVSCodeDebugSettingsReadsJSONC(t *testing.T) {
	settings, err := parseVSCodeDebugSettings([]byte(`{
  // VS Code accepts JSONC comments here.
  "debug.javascript.autoAttachFilter": "disabled",
  "debug.javascript.autoAttachSmartPattern": ["!**/node_modules/**"]
}`))
	if err != nil {
		t.Fatalf("parse settings: %v", err)
	}
	if !settings.HasAutoAttachFilter || settings.AutoAttachFilter != "disabled" {
		t.Fatalf("settings = %#v, want disabled auto attach filter", settings)
	}
	if settings.SmartPatternCount != 1 {
		t.Fatalf("smart pattern count = %d, want 1", settings.SmartPatternCount)
	}
}
