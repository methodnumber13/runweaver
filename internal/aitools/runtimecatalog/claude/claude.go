package claude

import catalog "github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog"

// Adapter exposes Claude Code runtime metadata to the shared catalog.
type Adapter struct{}

// ID returns the canonical Claude Code runtime ID.
func (Adapter) ID() string {
	return catalog.Claude
}

// Provider returns Claude Code runtime metadata.
func (Adapter) Provider() catalog.Provider {
	return catalog.Provider{
		ID:          catalog.Claude,
		Name:        "Claude Code",
		Binary:      "claude",
		Description: "Claude Code CLAUDE.md, .claude/agents, .claude/skills, and settings surfaces.",
	}
}

// ProfilePath returns the generated Claude Code profile path.
func (Adapter) ProfilePath() string {
	return ".claude/runweaver/profile.json"
}

// GeneratedPaths returns Claude Code files and directories managed by RunWeaver.
func (a Adapter) GeneratedPaths() []string {
	return []string{"CLAUDE.md", ".claude/agents", ".claude/skills", ".claude/workflows", a.ProfilePath()}
}

// Capabilities returns Claude Code runtime support flags.
func (Adapter) Capabilities() map[string]catalog.Flag {
	return map[string]catalog.Flag{
		"render":   {Supported: true, Summary: "writes CLAUDE.md, .claude/agents, and .claude/skills metadata"},
		"doctor":   {Supported: true, Summary: "detects Claude binary, project metadata, global settings, and managed settings"},
		"execute":  {Supported: true, Summary: "runs claude --print through workflow executor"},
		"classify": {Supported: true, Summary: "runs model-backed repo classification through claude --print"},
	}
}

// DelegationGuidance returns orchestration guidance for Claude Code prompts.
func (Adapter) DelegationGuidance() string {
	return "Delegate through Claude Code's Agent/subagent mechanism when available, otherwise emulate selected roles explicitly and record that in participants."
}
