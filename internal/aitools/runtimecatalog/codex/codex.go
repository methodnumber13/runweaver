package codex

import catalog "github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog"

// Adapter exposes Codex runtime metadata to the shared catalog.
type Adapter struct{}

// ID returns the canonical Codex runtime ID.
func (Adapter) ID() string {
	return catalog.Codex
}

// Provider returns Codex runtime metadata.
func (Adapter) Provider() catalog.Provider {
	return catalog.Provider{
		ID:          catalog.Codex,
		Name:        "Codex",
		Binary:      "codex",
		Description: "Codex AGENTS.md, .agents/skills, .codex/agents, and codex exec surfaces.",
	}
}

// ProfilePath returns the generated Codex profile path.
func (Adapter) ProfilePath() string {
	return ".codex/runweaver/profile.json"
}

// GeneratedPaths returns Codex files and directories managed by RunWeaver.
func (a Adapter) GeneratedPaths() []string {
	return []string{"AGENTS.md", ".agents/skills", ".codex/agents", a.ProfilePath()}
}

// Capabilities returns Codex runtime support flags.
func (Adapter) Capabilities() map[string]catalog.Flag {
	return map[string]catalog.Flag{
		"render":   {Supported: true, Summary: "writes AGENTS.md, .agents/skills, and .codex/agents metadata"},
		"doctor":   {Supported: true, Summary: "detects Codex binary, project metadata, global config, and auth files"},
		"execute":  {Supported: true, Summary: "runs codex exec through workflow executor"},
		"classify": {Supported: true, Summary: "runs model-backed repo classification through codex exec"},
	}
}

// DelegationGuidance returns orchestration guidance for Codex prompts.
func (Adapter) DelegationGuidance() string {
	return "Use Codex subagents when explicitly available, otherwise emulate selected roles explicitly and record that in participants."
}
