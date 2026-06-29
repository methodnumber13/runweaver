package aitools

// ContextQueryOptions configures task-scoped context retrieval.
type ContextQueryOptions struct {
	Task             string
	Limit            int
	IncludeGenerated bool
}

// ContextQueryResult is the compact task context returned to an LLM runtime.
type ContextQueryResult struct {
	Status   string           `json:"status"`
	RepoRoot string           `json:"repoRoot,omitempty"`
	Task     string           `json:"task,omitempty"`
	Index    string           `json:"index,omitempty"`
	Limit    int              `json:"limit,omitempty"`
	Files    []ContextFileHit `json:"files,omitempty"`
	Symbols  []SymbolInfo     `json:"symbols,omitempty"`
	Routes   []IndexEdge      `json:"routes,omitempty"`
	Tests    []IndexEdge      `json:"tests,omitempty"`
	Commands []string         `json:"commands,omitempty"`
	Warnings []string         `json:"warnings,omitempty"`
}

// ContextFileHit is one ranked file selected for a task.
type ContextFileHit struct {
	Path      string   `json:"path"`
	Category  string   `json:"category,omitempty"`
	Language  string   `json:"language,omitempty"`
	Score     int      `json:"score"`
	Generated bool     `json:"generated,omitempty"`
	Rationale []string `json:"rationale,omitempty"`
}
