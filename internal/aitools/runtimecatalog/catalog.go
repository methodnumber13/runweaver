package runtimecatalog

import "strings"

const (
	// Auto lets RunWeaver select the best runtime for one task from repo metadata.
	Auto = "auto"
	// OpenCode is the canonical ID for the OpenCode runtime.
	OpenCode = "opencode"
	// Codex is the canonical ID for the Codex runtime.
	Codex = "codex"
	// Claude is the canonical ID for the Claude Code runtime.
	Claude = "claude"
	// All selects every supported runtime.
	All = "all"
)

// Provider is the stable metadata for an AI coding runtime.
type Provider struct {
	ID          string
	Name        string
	Binary      string
	Description string
}

// Flag describes whether a runtime capability is supported.
type Flag struct {
	Supported bool
	Summary   string
}

// Adapter provides static metadata for a runtime provider.
type Adapter interface {
	// ID returns the canonical runtime ID.
	ID() string
	// Provider returns user-facing runtime metadata.
	Provider() Provider
	// ProfilePath returns the runtime-specific generated profile path.
	ProfilePath() string
	// GeneratedPaths returns files and directories managed by RunWeaver.
	GeneratedPaths() []string
	// Capabilities returns runtime support flags.
	Capabilities() map[string]Flag
	// DelegationGuidance returns runtime-specific orchestration instructions.
	DelegationGuidance() string
}

// NormalizeID canonicalizes runtime IDs and supported aliases.
func NormalizeID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	switch value {
	case "", Auto, OpenCode, Codex, Claude, All:
		return value
	case "open-code":
		return OpenCode
	case "claude-code", "claudecode":
		return Claude
	default:
		return value
	}
}

// Order returns the stable ordering rank for a runtime ID.
func Order(id string) int {
	switch NormalizeID(id) {
	case OpenCode:
		return 0
	case Codex:
		return 1
	case Claude:
		return 2
	default:
		return 99
	}
}
