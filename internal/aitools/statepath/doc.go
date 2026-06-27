// Package statepath centralizes RunWeaver's repo-local path layout.
//
// Canonical generated state lives under .runweaver. Workflow templates are read
// from .runweaver/workflows, workflow runs and checkpoints are written under
// .runweaver/tmp/swarm-runs, and index/cache artifacts live under
// .runweaver/tmp/index and .runweaver/tmp/cache. Keeping these paths in one
// package prevents runtime adapters from drifting apart.
package statepath
