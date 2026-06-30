package aitools

const (
	// RunWeaverPrimaryAgentName is the repo-local coordinator entrypoint.
	// It is intentionally not "swarm" because runtime plugins or global
	// configs may define that generic agent name and shadow project metadata.
	RunWeaverPrimaryAgentName = "runweaver-swarm"

	// OpenCodePrimaryAgentName is kept as the OpenCode-facing alias for callers
	// and tests that refer to the OpenCode runtime explicitly.
	OpenCodePrimaryAgentName = RunWeaverPrimaryAgentName

	openCodeLegacyPrimaryAgentName = "swarm"
)
