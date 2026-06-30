package aitools

const (
	// OpenCodePrimaryAgentName is the repo-local RunWeaver OpenCode entrypoint.
	// It is intentionally not "swarm" because global plugins may define that
	// generic agent name and shadow project-local RunWeaver metadata.
	OpenCodePrimaryAgentName = "runweaver-swarm"

	openCodeLegacyPrimaryAgentName = "swarm"
)
