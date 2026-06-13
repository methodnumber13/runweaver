package aitools

import (
	"github.com/methodnumber13/runweaver/internal/aitools/modelconfig"
	catalog "github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog"
)

const (
	// RuntimeOpenCode identifies the OpenCode runtime provider.
	RuntimeOpenCode = catalog.OpenCode
	// RuntimeCodex identifies the Codex runtime provider.
	RuntimeCodex = catalog.Codex
	// RuntimeClaude identifies the Claude Code runtime provider.
	RuntimeClaude = catalog.Claude
	// RuntimeAll selects every supported runtime provider.
	RuntimeAll = catalog.All
)

// RuntimeProvider is the public metadata for one AI coding runtime.
type RuntimeProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Binary      string `json:"binary"`
	Description string `json:"description"`
}

// RuntimeDiscoveryOptions configures provider readiness discovery.
type RuntimeDiscoveryOptions struct {
	Runtime string
}

// RuntimeDiscoveryResult reports binary, config, auth, and capability readiness.
type RuntimeDiscoveryResult struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Binary         string                 `json:"binary"`
	BinaryPath     string                 `json:"binaryPath,omitempty"`
	BinaryFound    bool                   `json:"binaryFound"`
	Status         string                 `json:"status"`
	Ready          bool                   `json:"ready"`
	Env            map[string]bool        `json:"env,omitempty"`
	ConfigFiles    []RuntimeFileCheck     `json:"configFiles,omitempty"`
	AuthFiles      []RuntimeFileCheck     `json:"authFiles,omitempty"`
	MetadataFiles  []RuntimeFileCheck     `json:"metadataFiles,omitempty"`
	ManagedFiles   []RuntimeFileCheck     `json:"managedFiles,omitempty"`
	GeneratedPaths []string               `json:"generatedPaths,omitempty"`
	Issues         []string               `json:"issues,omitempty"`
	Warnings       []string               `json:"warnings,omitempty"`
	Capabilities   map[string]RuntimeFlag `json:"capabilities,omitempty"`
}

// RuntimeFileCheck records one config/auth/metadata file probe.
type RuntimeFileCheck struct {
	Path     string   `json:"path"`
	Source   string   `json:"source"`
	Kind     string   `json:"kind"`
	Exists   bool     `json:"exists"`
	Readable bool     `json:"readable"`
	Issues   []string `json:"issues,omitempty"`
}

// RuntimeFlag describes one runtime capability and its support level.
type RuntimeFlag struct {
	Supported bool   `json:"supported"`
	Summary   string `json:"summary,omitempty"`
}

type configCandidate = modelconfig.Candidate
