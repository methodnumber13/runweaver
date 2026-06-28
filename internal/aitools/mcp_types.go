package aitools

// MCPServerOptions configures the local RunWeaver MCP stdio server.
type MCPServerOptions struct {
	RepoPath string
	Version  string
}

// RunWeaverCurrentResult returns the markdown resume surface exposed to LLM clients.
type RunWeaverCurrentResult struct {
	Status          string                `json:"status"`
	Ready           bool                  `json:"ready"`
	RepoRoot        string                `json:"repoRoot"`
	CurrentPath     string                `json:"currentPath,omitempty"`
	CurrentMarkdown string                `json:"currentMarkdown,omitempty"`
	State           RunWeaverStatusResult `json:"state"`
	Recommendations []string              `json:"recommendations,omitempty"`
}

// WorkflowTemplateSummary is a compact listing of one generated workflow template.
type WorkflowTemplateSummary struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Path        string   `json:"path"`
	Phases      []string `json:"phases"`
}

// WorkflowTemplateListResult lists repo-local workflow templates exposed through MCP.
type WorkflowTemplateListResult struct {
	Status    string                    `json:"status"`
	RepoRoot  string                    `json:"repoRoot"`
	Root      string                    `json:"root"`
	Workflows []WorkflowTemplateSummary `json:"workflows"`
}
