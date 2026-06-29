package aitools

// WorkflowSpec is the portable workflow template stored under .runweaver/workflows.
type WorkflowSpec struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	MaxConcurrent   int             `json:"maxConcurrent,omitempty"`
	MaxParticipants int             `json:"maxParticipants,omitempty"`
	Phases          []WorkflowPhase `json:"phases"`
}

// WorkflowPhase defines one ordered phase and the agents expected to handle it.
type WorkflowPhase struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Scope       string   `json:"scope"`
	Mode        string   `json:"mode"`
	WriteMode   string   `json:"writeMode"`
	Concurrency int      `json:"concurrency,omitempty"`
	Agents      []string `json:"agents"`
	Prompt      string   `json:"prompt"`
}

// WorkflowRunSummary is returned when a workflow run is planned.
type WorkflowRunSummary struct {
	Workflow       string   `json:"workflow"`
	Task           string   `json:"task"`
	RunDir         string   `json:"runDir"`
	CheckpointPath string   `json:"checkpointPath"`
	TodoPath       string   `json:"todoPath"`
	Phases         []string `json:"phases"`
	Status         string   `json:"status"`
}

// WorkflowSelectOptions configures deterministic workflow routing for a task.
type WorkflowSelectOptions struct {
	Task     string
	Workflow string
}

// WorkflowSelectResult reports the selected workflow plus ranked alternatives.
type WorkflowSelectResult struct {
	Status       string                       `json:"status"`
	RepoRoot     string                       `json:"repoRoot"`
	Task         string                       `json:"task"`
	TaskTier     TaskTierResult               `json:"taskTier"`
	WorkflowPath string                       `json:"workflowPath"`
	Selected     WorkflowSelectionCandidate   `json:"selected"`
	Candidates   []WorkflowSelectionCandidate `json:"candidates"`
}

// TaskTierResult describes how much orchestration one task should receive.
type TaskTierResult struct {
	Tier           string   `json:"tier"`
	Score          int      `json:"score"`
	ParticipantCap int      `json:"participantCap"`
	PhaseStrategy  string   `json:"phaseStrategy"`
	Verification   string   `json:"verification"`
	Rationale      []string `json:"rationale,omitempty"`
}

// WorkflowSelectionCandidate is one workflow considered for a task.
type WorkflowSelectionCandidate struct {
	ID          string   `json:"id"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Path        string   `json:"path"`
	Score       int      `json:"score"`
	Explicit    bool     `json:"explicit,omitempty"`
	Rationale   []string `json:"rationale,omitempty"`
}

// ParticipantSelectOptions configures repo-specific participant routing.
type ParticipantSelectOptions struct {
	Task        string
	Workflow    string
	Runtime     string
	ProfilePath string
	TaskTier    string
}

// ParticipantSelectResult reports selected agents/skills for one workflow task.
type ParticipantSelectResult struct {
	Status       string                          `json:"status"`
	RepoRoot     string                          `json:"repoRoot"`
	Task         string                          `json:"task"`
	Runtime      string                          `json:"runtime"`
	TaskTier     string                          `json:"taskTier,omitempty"`
	Workflow     string                          `json:"workflow"`
	WorkflowPath string                          `json:"workflowPath"`
	ProfilePath  string                          `json:"profilePath,omitempty"`
	Cap          int                             `json:"cap"`
	Participants []string                        `json:"participants"`
	Rationale    []string                        `json:"rationale"`
	Candidates   []ParticipantSelectionCandidate `json:"candidates"`
}

// ParticipantSelectionCandidate is one agent or skill considered for routing.
type ParticipantSelectionCandidate struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Source    string   `json:"source,omitempty"`
	Score     int      `json:"score"`
	Selected  bool     `json:"selected,omitempty"`
	Rationale []string `json:"rationale,omitempty"`
}

// WorkflowPlanFile is the durable plan written into a run directory.
type WorkflowPlanFile struct {
	SchemaVersion int          `json:"schemaVersion"`
	RunID         string       `json:"runId"`
	Workflow      WorkflowSpec `json:"workflow"`
	Task          string       `json:"task"`
	Status        string       `json:"status"`
	RunDir        string       `json:"runDir"`
}

// WorkflowCheckpoint is the durable resume state updated after every phase.
type WorkflowCheckpoint struct {
	SchemaVersion        int      `json:"schemaVersion"`
	RunID                string   `json:"runId"`
	Workflow             string   `json:"workflow"`
	Task                 string   `json:"task"`
	Status               string   `json:"status"`
	CurrentPhase         string   `json:"currentPhase,omitempty"`
	CompletedPhases      []string `json:"completedPhases"`
	NextPhase            string   `json:"nextPhase"`
	Participants         []string `json:"participants,omitempty"`
	ParticipantRationale []string `json:"participantRationale,omitempty"`
	Findings             []string `json:"findings,omitempty"`
	Decisions            []string `json:"decisions,omitempty"`
	FilesRead            []string `json:"filesRead,omitempty"`
	FilesChanged         []string `json:"filesChanged,omitempty"`
	Artifacts            []string `json:"artifacts,omitempty"`
	LastResult           string   `json:"lastResult,omitempty"`
	RejectedPaths        []string `json:"rejectedPaths,omitempty"`
	NextAction           string   `json:"nextAction,omitempty"`
	NextVerification     string   `json:"nextVerification,omitempty"`
	Verification         []string `json:"verification,omitempty"`
	VerificationResults  []string `json:"verificationResults,omitempty"`
	Blockers             []string `json:"blockers,omitempty"`
	IndexFreshnessStatus string   `json:"indexFreshnessStatus,omitempty"`
	StaleIndex           bool     `json:"staleIndex,omitempty"`
	StaleIndexFiles      []string `json:"staleIndexFiles,omitempty"`
	UpdatedAt            string   `json:"updatedAt"`
}

// WorkflowLatest points to the most recently created canonical workflow run.
type WorkflowLatest struct {
	RunID     string `json:"runId"`
	RunDir    string `json:"runDir"`
	Workflow  string `json:"workflow"`
	Task      string `json:"task"`
	UpdatedAt string `json:"updatedAt"`
}

// WorkflowEvent is one append-only event in a workflow run log.
type WorkflowEvent struct {
	Type                 string   `json:"type"`
	RunID                string   `json:"runId,omitempty"`
	Workflow             string   `json:"workflow,omitempty"`
	Task                 string   `json:"task,omitempty"`
	Phase                string   `json:"phase,omitempty"`
	Status               string   `json:"status,omitempty"`
	Participants         []string `json:"participants,omitempty"`
	ParticipantRationale []string `json:"participantRationale,omitempty"`
	Findings             []string `json:"findings,omitempty"`
	Decisions            []string `json:"decisions,omitempty"`
	FilesRead            []string `json:"filesRead,omitempty"`
	FilesChanged         []string `json:"filesChanged,omitempty"`
	Artifacts            []string `json:"artifacts,omitempty"`
	LastResult           string   `json:"lastResult,omitempty"`
	RejectedPaths        []string `json:"rejectedPaths,omitempty"`
	NextAction           string   `json:"nextAction,omitempty"`
	NextVerification     string   `json:"nextVerification,omitempty"`
	Verification         []string `json:"verification,omitempty"`
	VerificationResults  []string `json:"verificationResults,omitempty"`
	Blockers             []string `json:"blockers,omitempty"`
	IndexFreshnessStatus string   `json:"indexFreshnessStatus,omitempty"`
	StaleIndex           bool     `json:"staleIndex,omitempty"`
	StaleIndexFiles      []string `json:"staleIndexFiles,omitempty"`
	Command              []string `json:"command,omitempty"`
	ExitCode             *int     `json:"exitCode,omitempty"`
	Error                string   `json:"error,omitempty"`
	Warnings             []string `json:"warnings,omitempty"`
	At                   string   `json:"at"`
}

// WorkflowUpdateOptions records agent progress in a workflow checkpoint.
type WorkflowUpdateOptions struct {
	Resume               string
	Phase                string
	Status               string
	Participants         []string
	ParticipantRationale []string
	Findings             []string
	Decisions            []string
	FilesRead            []string
	FilesChanged         []string
	Artifacts            []string
	LastResult           string
	RejectedPaths        []string
	NextAction           string
	NextVerification     string
	Verification         []string
	VerificationResults  []string
	Blockers             []string
	CompletePhase        bool
}

// WorkflowVerificationResult is the structured output of workflow verification.
type WorkflowVerificationResult struct {
	Status          string                      `json:"status"`
	Ready           bool                        `json:"ready"`
	RepoRoot        string                      `json:"repoRoot"`
	RunDir          string                      `json:"runDir"`
	Checks          []WorkflowVerificationCheck `json:"checks"`
	Recommendations []string                    `json:"recommendations,omitempty"`
}

// WorkflowVerificationCheck is one validation item in a workflow verification run.
type WorkflowVerificationCheck struct {
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Summary     string   `json:"summary"`
	Evidence    []string `json:"evidence,omitempty"`
	NextActions []string `json:"nextActions,omitempty"`
}
