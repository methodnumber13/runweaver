package opencode

import catalog "github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog"

// Adapter exposes OpenCode runtime metadata to the shared catalog.
type Adapter struct{}

// ID returns the canonical OpenCode runtime ID.
func (Adapter) ID() string {
	return catalog.OpenCode
}

// Provider returns OpenCode runtime metadata.
func (Adapter) Provider() catalog.Provider {
	return catalog.Provider{
		ID:          catalog.OpenCode,
		Name:        "OpenCode",
		Binary:      "opencode",
		Description: "OpenCode agents, skills, workflows, and opencode.json metadata.",
	}
}

// ProfilePath returns the generated OpenCode profile path.
func (Adapter) ProfilePath() string {
	return ".opencode/swarm/profile.json"
}

// GeneratedPaths returns OpenCode files and directories managed by RunWeaver.
func (a Adapter) GeneratedPaths() []string {
	return []string{"opencode.json", ".opencode/agents", ".opencode/commands", ".opencode/skills", ".runweaver/workflows", a.ProfilePath()}
}

// Capabilities returns OpenCode runtime support flags.
func (Adapter) Capabilities() map[string]catalog.Flag {
	return map[string]catalog.Flag{
		"render":   {Supported: true, Summary: "writes OpenCode agents, skills, workflows, and opencode.json"},
		"doctor":   {Supported: true, Summary: "checks OpenCode config, model, agents, and tools"},
		"execute":  {Supported: true, Summary: "runs opencode run through workflow executor"},
		"classify": {Supported: true, Summary: "runs model-backed repo classification through opencode run"},
	}
}

// DelegationGuidance returns orchestration guidance for OpenCode prompts.
func (Adapter) DelegationGuidance() string {
	return "Delegate through OpenCode's available agent/task mechanism when available; otherwise emulate the roles explicitly and record that in participants."
}
