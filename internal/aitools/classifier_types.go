package aitools

import (
	"time"
)

// ClassificationMode selects deterministic, AI-backed, or fallback classification.
type ClassificationMode string

const (
	// ClassificationAuto tries AI classification and falls back to deterministic output.
	ClassificationAuto ClassificationMode = "auto"
	// ClassificationAI requires a successful AI classification.
	ClassificationAI ClassificationMode = "ai"
	// ClassificationDeterministic uses only local repository signals.
	ClassificationDeterministic ClassificationMode = "deterministic"
)

// ClassifyOptions configures repository classification and optional metadata rendering.
type ClassifyOptions struct {
	Mode             ClassificationMode
	Runtime          string
	ProviderID       string
	Model            string
	OpencodeBin      string
	CodexBin         string
	ClaudeBin        string
	Agent            string
	Sandbox          string
	ApprovalPolicy   string
	PermissionMode   string
	ClaudeTools      string
	SkipGitRepoCheck bool
	Timeout          time.Duration
	SkipModelCheck   bool
	SkipRuntimeCheck bool
	Apply            bool
	ApplyRuntime     string
}

// ClassifyRunSummary captures how classification was produced and validated.
type ClassifyRunSummary struct {
	Status           string                  `json:"status"`
	Mode             string                  `json:"mode"`
	Runtime          string                  `json:"runtime"`
	Source           string                  `json:"source"`
	ModelAttempted   bool                    `json:"modelAttempted"`
	ModelUsed        bool                    `json:"modelUsed"`
	UsedFallback     bool                    `json:"usedFallback"`
	FallbackReason   string                  `json:"fallbackReason,omitempty"`
	PromptPath       string                  `json:"promptPath,omitempty"`
	RawOutputPath    string                  `json:"rawOutputPath,omitempty"`
	ModelPreflight   *ModelConfigCheck       `json:"modelPreflight,omitempty"`
	RuntimePreflight *RuntimeDiscoveryResult `json:"runtimePreflight,omitempty"`
	Warnings         []string                `json:"warnings,omitempty"`
	ValidationError  string                  `json:"validationError,omitempty"`
}

// ClassifyResult is the CLI-facing result returned by classify commands.
type ClassifyResult struct {
	Status         string                 `json:"status"`
	RepoRoot       string                 `json:"repoRoot"`
	IndexPath      string                 `json:"indexPath"`
	ProfilePath    string                 `json:"profilePath,omitempty"`
	ProfilePaths   []string               `json:"profilePaths,omitempty"`
	Applied        bool                   `json:"applied"`
	Classifier     ClassifyRunSummary     `json:"classifier"`
	Artifacts      IndexArtifacts         `json:"artifacts"`
	Classification RepoClassification     `json:"classification"`
	Stats          IndexStats             `json:"stats"`
	Warnings       []string               `json:"warnings,omitempty"`
	Stack          StackInfo              `json:"stack"`
	Tools          ToolchainInfo          `json:"tools"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}
