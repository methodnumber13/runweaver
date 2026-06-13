package aitools

// Profile is the runtime-neutral RunWeaver metadata rendered into providers.
type Profile struct {
	Workspace    WorkspaceProfile `json:"workspace"`
	GlobalAgents []AgentProfile   `json:"globalAgents"`
	Repos        []RepoProfile    `json:"repos"`
}

// WorkspaceProfile describes the workspace-level context shared by repos.
type WorkspaceProfile struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Repos       []string `json:"repos"`
}

// RepoProfile contains the agents, skills, and risks for one repository.
type RepoProfile struct {
	Dir          string             `json:"dir"`
	Kind         string             `json:"kind"`
	Purpose      string             `json:"purpose"`
	Domain       string             `json:"domain"`
	Runtime      string             `json:"runtime"`
	Semantic     RepoClassification `json:"semantic,omitempty"`
	KeyFiles     []string           `json:"keyFiles"`
	Risks        []string           `json:"risks"`
	Agents       []AgentProfile     `json:"agents"`
	CustomSkills []SkillProfile     `json:"customSkills"`
}

// AgentProfile defines one generated assistant role and its operating scope.
type AgentProfile struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Mode         string   `json:"mode,omitempty"`
	FocusFiles   []string `json:"focusFiles,omitempty"`
	Workflow     []string `json:"workflow,omitempty"`
	Verification []string `json:"verification,omitempty"`
}

// SkillProfile defines reusable instructions generated for a repo/runtime.
type SkillProfile struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	FocusFiles   []string `json:"focusFiles"`
	Workflow     []string `json:"workflow"`
	Risks        []string `json:"risks,omitempty"`
	Verification []string `json:"verification,omitempty"`
}
