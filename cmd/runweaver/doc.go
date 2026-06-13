// Command runweaver prepares repository-local AI workflow metadata for coding
// assistants.
//
// RunWeaver scans a repository, builds a compact index, classifies important
// code surfaces, and renders runtime-specific agent, skill, and workflow files
// for OpenCode, Codex, and Claude Code. It keeps generated execution state
// under .runweaver/tmp so workflow plans and checkpoints can survive context
// resets without becoming part of the source repository.
//
// The main commands are:
//
//	runweaver init --repo . --runtime all
//	runweaver index --repo . --changed-only --prune
//	runweaver classify --repo . --classification auto --apply
//	runweaver workflow run --workflow .runweaver/workflows/feature-delivery-swarm.json --task "..."
//	runweaver doctor runtime --repo . --runtime all
//
// Use runweaver help or a subcommand's --help flag for the full CLI reference.
package main
