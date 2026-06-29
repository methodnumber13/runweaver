package aitools

// StartOptions configures the single RunWeaver task intake entrypoint.
type StartOptions struct {
	Task        string
	Runtime     string
	Workflow    string
	ProfilePath string
	SkipIndex   bool
	ForceNew    bool
}

// StartResult is the execution contract returned to an LLM runtime.
type StartResult struct {
	Status            string                  `json:"status"`
	Ready             bool                    `json:"ready"`
	Action            string                  `json:"action"`
	RepoRoot          string                  `json:"repoRoot"`
	Runtime           string                  `json:"runtime"`
	RuntimeResolution RuntimeResolutionResult `json:"runtimeResolution"`
	Task              string                  `json:"task"`
	TaskTier          TaskTierResult          `json:"taskTier"`
	IndexRefreshed    bool                    `json:"indexRefreshed"`
	IndexFreshness    IndexFreshnessResult    `json:"indexFreshness"`
	Context           ContextQueryResult      `json:"context,omitempty"`
	WorkflowSelection WorkflowSelectResult    `json:"workflowSelection,omitempty"`
	Workflow          WorkflowRunSummary      `json:"workflow"`
	Participants      ParticipantSelectResult `json:"participants"`
	ExecutionContract StartExecutionContract  `json:"executionContract"`
	Recommendations   []string                `json:"recommendations,omitempty"`
}

// StartExecutionContract is the short instruction block an LLM should follow next.
type StartExecutionContract struct {
	RunDir           string             `json:"runDir"`
	CheckpointPath   string             `json:"checkpointPath"`
	TodoPath         string             `json:"todoPath"`
	CurrentPhase     string             `json:"currentPhase,omitempty"`
	NextPhase        string             `json:"nextPhase,omitempty"`
	TaskTier         TaskTierResult     `json:"taskTier"`
	Context          ContextQueryResult `json:"context,omitempty"`
	Participants     []string           `json:"participants"`
	NextAction       string             `json:"nextAction"`
	NextVerification string             `json:"nextVerification"`
	ResumeStrategy   string             `json:"resumeStrategy"`
}

// RuntimeResolutionResult explains how a single runtime was selected.
type RuntimeResolutionResult struct {
	Requested  string                       `json:"requested"`
	Selected   string                       `json:"selected"`
	Source     string                       `json:"source"`
	Candidates []RuntimeResolutionCandidate `json:"candidates,omitempty"`
}

// RuntimeResolutionCandidate is one runtime considered for auto selection.
type RuntimeResolutionCandidate struct {
	ID        string `json:"id"`
	Score     int    `json:"score"`
	Source    string `json:"source"`
	Profile   string `json:"profile,omitempty"`
	Generated bool   `json:"generated,omitempty"`
}
