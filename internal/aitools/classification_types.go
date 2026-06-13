package aitools

// RepoClassification is the semantic map of a repository produced by rules or AI.
type RepoClassification struct {
	Source           string                 `json:"source"`
	ModelReady       bool                   `json:"modelReady"`
	ValidationStatus string                 `json:"validationStatus"`
	Summary          string                 `json:"summary,omitempty"`
	Entrypoints      []ClassifiedSurface    `json:"entrypoints,omitempty"`
	Domains          []DomainClassification `json:"domains,omitempty"`
	ExternalSystems  []ClassifiedSurface    `json:"externalSystems,omitempty"`
	Persistence      []ClassifiedSurface    `json:"persistence,omitempty"`
	Tests            []ClassifiedSurface    `json:"tests,omitempty"`
	Configs          []ClassifiedSurface    `json:"configs,omitempty"`
	Agents           []AgentProfile         `json:"agents,omitempty"`
	Skills           []SkillProfile         `json:"skills,omitempty"`
	Verification     []string               `json:"verification,omitempty"`
	Warnings         []string               `json:"warnings,omitempty"`
}

// ClassifiedSurface describes one non-domain surface with supporting evidence.
type ClassifiedSurface struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind,omitempty"`
	Files      []string `json:"files,omitempty"`
	Evidence   []string `json:"evidence,omitempty"`
	Confidence string   `json:"confidence,omitempty"`
}

// DomainClassification describes a business or technical domain in the repo.
type DomainClassification struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Kind        string   `json:"kind,omitempty"`
	Files       []string `json:"files,omitempty"`
	Evidence    []string `json:"evidence,omitempty"`
	Confidence  string   `json:"confidence,omitempty"`
}
