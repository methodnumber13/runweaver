package aitools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// StartWorkflow creates or resumes the closest RunWeaver workflow for a task.
func StartWorkflow(repoPath string, opts StartOptions) (StartResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return StartResult{}, err
	}
	task := strings.TrimSpace(opts.Task)
	if task == "" {
		return StartResult{}, fmt.Errorf("start task is required")
	}
	runtimeID, runtimeResolution, err := ResolveSingleRuntime(root, opts.Runtime)
	if err != nil {
		return StartResult{}, err
	}
	taskTier := ClassifyTaskTier(task)
	indexFreshness := CheckIndexFreshness(root)
	indexRefreshed := false
	if !opts.SkipIndex && !indexFreshness.Fresh {
		if _, err := IndexWithOptions(root, IndexOptions{
			ChangedOnly:    true,
			Prune:          true,
			MaxCacheMB:     256,
			Classification: ClassifyOptions{Mode: ClassificationDeterministic},
		}); err != nil {
			return StartResult{}, fmt.Errorf("refresh index before start: %w", err)
		}
		indexRefreshed = true
		indexFreshness = CheckIndexFreshness(root)
	}
	if !opts.ForceNew {
		resumed, ok, err := tryResumeStart(root, task, runtimeID, runtimeResolution, taskTier, opts)
		if err != nil {
			return StartResult{}, err
		}
		if ok {
			resumed.IndexRefreshed = indexRefreshed
			resumed.IndexFreshness = indexFreshness
			return resumed, nil
		}
	}
	workflowSelection, err := SelectWorkflow(root, WorkflowSelectOptions{Task: task, Workflow: opts.Workflow})
	if err != nil {
		return StartResult{}, err
	}
	workflow, err := PlanWorkflow(root, workflowSelection.WorkflowPath, task)
	if err != nil {
		return StartResult{}, err
	}
	participants, err := SelectParticipants(root, ParticipantSelectOptions{
		Task:        task,
		Workflow:    workflowSelection.WorkflowPath,
		Runtime:     runtimeID,
		ProfilePath: opts.ProfilePath,
		TaskTier:    taskTier.Tier,
	})
	if err != nil {
		return StartResult{}, err
	}
	if _, err := persistStartSelection(root, workflow, participants, "created"); err != nil {
		return StartResult{}, err
	}
	status, err := WorkflowStatus(root, "latest")
	if err != nil {
		return StartResult{}, err
	}
	return StartResult{
		Status:            "success",
		Ready:             true,
		Action:            "created",
		RepoRoot:          root,
		Runtime:           runtimeID,
		RuntimeResolution: runtimeResolution,
		Task:              task,
		TaskTier:          taskTier,
		IndexRefreshed:    indexRefreshed,
		IndexFreshness:    indexFreshness,
		WorkflowSelection: workflowSelection,
		Workflow:          workflow,
		Participants:      participants,
		ExecutionContract: startExecutionContract(status, participants.Participants, taskTier),
		Recommendations:   []string{"continue phase by phase; update checkpoint after each phase", "run runweaver workflow verify --repo . --resume latest before final response"},
	}, nil
}

func tryResumeStart(root, task, runtimeID string, runtimeResolution RuntimeResolutionResult, taskTier TaskTierResult, opts StartOptions) (StartResult, bool, error) {
	status, err := RunWeaverStatus(root)
	if err != nil {
		return StartResult{}, false, err
	}
	if !status.LatestWorkflow || status.WorkflowStatus == "complete" {
		return StartResult{}, false, nil
	}
	if !tasksLikelyMatch(status.Task, task) && !isContinueTask(task) {
		return StartResult{}, false, nil
	}
	workflowPath := opts.Workflow
	if workflowPath == "" {
		workflowPath = workflowPathForID(root, status.Workflow)
	}
	participants, err := SelectParticipants(root, ParticipantSelectOptions{
		Task:        task,
		Workflow:    workflowPath,
		Runtime:     runtimeID,
		ProfilePath: opts.ProfilePath,
		TaskTier:    taskTier.Tier,
	})
	if err != nil {
		return StartResult{}, false, err
	}
	workflow := WorkflowRunSummary{
		Workflow:       status.Workflow,
		Task:           status.Task,
		RunDir:         status.RunDir,
		CheckpointPath: status.CheckpointPath,
		TodoPath:       status.TodoPath,
		Status:         status.WorkflowStatus,
	}
	if len(status.Participants) == 0 && len(participants.Participants) > 0 {
		if _, err := persistStartSelection(root, workflow, participants, "resumed"); err != nil {
			return StartResult{}, false, err
		}
		status, err = RunWeaverStatus(root)
		if err != nil {
			return StartResult{}, false, err
		}
	}
	return StartResult{
		Status:            "success",
		Ready:             true,
		Action:            "resumed",
		RepoRoot:          root,
		Runtime:           runtimeID,
		RuntimeResolution: runtimeResolution,
		Task:              task,
		TaskTier:          taskTier,
		IndexFreshness:    status.IndexFreshness,
		Workflow:          workflow,
		Participants:      participants,
		ExecutionContract: startExecutionContract(map[string]any{
			"runDir":           status.RunDir,
			"checkpointPath":   status.CheckpointPath,
			"todoPath":         status.TodoPath,
			"currentPhase":     status.CurrentPhase,
			"nextPhase":        status.NextPhase,
			"nextAction":       status.NextAction,
			"nextVerification": status.NextVerification,
		}, participantNamesOrExisting(participants.Participants, status.Participants), taskTier),
		Recommendations: []string{"resume automatically from checkpoint; do not ask the user to run resume manually"},
	}, true, nil
}

func persistStartSelection(root string, workflow WorkflowRunSummary, participants ParticipantSelectResult, action string) (map[string]any, error) {
	phase := firstPhaseID(workflow.Phases)
	if phase == "" {
		phase = "start"
	}
	return UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:               "latest",
		Phase:                phase,
		Status:               "in_progress",
		Participants:         participants.Participants,
		ParticipantRationale: participants.Rationale,
		LastResult:           "runweaver start " + action + " workflow " + workflow.Workflow + " and selected participants",
		NextAction:           "execute workflow phase " + phase,
		NextVerification:     "run runweaver workflow verify --repo . --resume latest before final response",
	})
}

func workflowPathForID(root, workflowID string) string {
	if workflowID == "" {
		return ""
	}
	for _, path := range []string{
		filepath.Join(root, ".runweaver", "workflows", workflowID+".json"),
		filepath.Join(root, ".opencode", "workflows", workflowID+".json"),
	} {
		if Exists(path) {
			return rel(root, path)
		}
	}
	return ""
}

func startExecutionContract(status map[string]any, participants []string, taskTier TaskTierResult) StartExecutionContract {
	return StartExecutionContract{
		RunDir:           stringValue(status["runDir"]),
		CheckpointPath:   stringValue(status["checkpointPath"]),
		TodoPath:         stringValue(status["todoPath"]),
		CurrentPhase:     stringValue(status["currentPhase"]),
		NextPhase:        stringValue(status["nextPhase"]),
		TaskTier:         taskTier,
		Participants:     participants,
		NextAction:       fallbackString(stringValue(status["nextAction"]), "execute the next workflow phase and update checkpoint.json"),
		NextVerification: fallbackString(stringValue(status["nextVerification"]), "run runweaver workflow verify --repo . --resume latest before final response"),
		ResumeStrategy:   "automatic via runweaver start; use runweaver workflow run --resume latest --status only as a diagnostic",
	}
}

func tasksLikelyMatch(existing, next string) bool {
	existingTokens := tokenizeSelectionText(existing)
	nextTokens := tokenizeSelectionText(next)
	if len(existingTokens) == 0 || len(nextTokens) == 0 {
		return false
	}
	overlap := 0
	for token := range nextTokens {
		if existingTokens[token] {
			overlap++
		}
	}
	return overlap >= 2
}

func isContinueTask(task string) bool {
	tokens := tokenizeSelectionText(task)
	return tokens["continue"] || tokens["resume"] || tokens["продолжи"] || tokens["дальше"]
}

func firstPhaseID(values []string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func participantNamesOrExisting(selected, existing []string) []string {
	if len(existing) > 0 {
		return existing
	}
	return selected
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
