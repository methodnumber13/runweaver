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
			Classification: ClassifyOptions{Mode: ClassificationDeterministic, Runtime: runtimeID},
		}); err != nil {
			return StartResult{}, fmt.Errorf("refresh index before start: %w", err)
		}
		indexRefreshed = true
		indexFreshness = CheckIndexFreshness(root)
	}
	context := contextForStart(root, task)
	if !opts.ForceNew {
		resumed, ok, err := tryResumeStart(root, task, runtimeID, runtimeResolution, taskTier, context, opts)
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
		Context:           context,
		WorkflowSelection: workflowSelection,
		Workflow:          workflow,
		Participants:      participants,
		ExecutionContract: startExecutionContract(status, participants.Participants, participants.Assignments, taskTier, context),
		Recommendations:   []string{"continue phase by phase; update checkpoint after each phase", "run $RUNWEAVER_BIN workflow verify --repo . --resume latest before final response"},
	}, nil
}

func tryResumeStart(root, task, runtimeID string, runtimeResolution RuntimeResolutionResult, taskTier TaskTierResult, context ContextQueryResult, opts StartOptions) (StartResult, bool, error) {
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
	if !isContinueTask(task) {
		selectedWorkflowID, err := selectedWorkflowIDForTask(root, task, opts.Workflow)
		if err != nil {
			return StartResult{}, false, err
		}
		if selectedWorkflowID != "" && status.Workflow != "" && selectedWorkflowID != status.Workflow {
			return StartResult{}, false, nil
		}
	}
	workflowPath := opts.Workflow
	if workflowPath == "" {
		workflowPath = workflowPathForID(root, status.Workflow)
	}
	workflow := WorkflowRunSummary{
		Workflow:       status.Workflow,
		Task:           status.Task,
		RunDir:         status.RunDir,
		CheckpointPath: status.CheckpointPath,
		TodoPath:       status.TodoPath,
		Status:         status.WorkflowStatus,
	}
	participants, err := resumeParticipantsFromCheckpoint(root, task, runtimeID, taskTier.Tier, workflow, workflowPath, status)
	if err != nil {
		return StartResult{}, false, err
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
		Context:           context,
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
		}, participants.Participants, participants.Assignments, taskTier, context),
		Recommendations: []string{"resume automatically from checkpoint; do not ask the user to run resume manually"},
	}, true, nil
}

func selectedWorkflowIDForTask(root, task, explicitWorkflow string) (string, error) {
	if strings.TrimSpace(explicitWorkflow) != "" {
		candidate, err := explicitWorkflowCandidate(root, explicitWorkflow)
		if err != nil {
			return "", err
		}
		return candidate.ID, nil
	}
	selection, err := SelectWorkflow(root, WorkflowSelectOptions{Task: task})
	if err != nil {
		return "", err
	}
	return selection.Selected.ID, nil
}

func resumeParticipantsFromCheckpoint(root, task, runtimeID, taskTier string, workflow WorkflowRunSummary, workflowPath string, status RunWeaverStatusResult) (ParticipantSelectResult, error) {
	if len(status.Participants) > 0 {
		return ParticipantSelectResult{
			Status:       "success",
			RepoRoot:     root,
			Task:         task,
			Runtime:      runtimeID,
			TaskTier:     taskTier,
			Workflow:     workflow.Workflow,
			WorkflowPath: workflowPath,
			Participants: status.Participants,
			Assignments:  checkpointParticipantAssignments(status.Participants),
			Rationale:    []string{"preserved participants from active workflow checkpoint"},
			Warnings:     []string{"participant selection was preserved from checkpoint during resume"},
		}, nil
	}
	return SelectParticipants(root, ParticipantSelectOptions{
		Task:     task,
		Workflow: workflowPath,
		Runtime:  runtimeID,
		TaskTier: taskTier,
	})
}

func checkpointParticipantAssignments(participants []string) []ParticipantAssignment {
	assignments := make([]ParticipantAssignment, 0, len(participants))
	for index, name := range participants {
		role := "reviewer"
		if index == 0 {
			role = "owner"
		}
		assignments = append(assignments, ParticipantAssignment{
			Name:      name,
			Kind:      "agent",
			Role:      role,
			Source:    "checkpoint",
			Rationale: []string{"preserved from active workflow checkpoint"},
		})
	}
	return assignments
}

func persistStartSelection(root string, workflow WorkflowRunSummary, participants ParticipantSelectResult, action string) (map[string]any, error) {
	phase := firstPhaseID(workflow.Phases)
	if phase == "" {
		phase = "start"
	}
	completePhase := phase == "plan"
	return UpdateWorkflow(root, WorkflowUpdateOptions{
		Resume:               "latest",
		Phase:                phase,
		Status:               "in_progress",
		CompletePhase:        completePhase,
		Participants:         participants.Participants,
		ParticipantRationale: participants.Rationale,
		LastResult:           startSelectionLastResult(action, workflow.Workflow, phase, completePhase),
		NextAction:           startSelectionNextAction(phase, completePhase, workflow.Phases),
		NextVerification:     "run $RUNWEAVER_BIN workflow verify --repo . --resume latest before final response",
	})
}

func startSelectionLastResult(action, workflowID, phase string, completed bool) string {
	result := "runweaver start " + action + " workflow " + workflowID + " and selected participants"
	if completed {
		return result + "; completed initial " + phase + " phase"
	}
	return result
}

func startSelectionNextAction(phase string, completed bool, phases []string) string {
	if completed {
		next := ""
		for index, candidate := range phases {
			if candidate == phase && index+1 < len(phases) {
				next = phases[index+1]
				break
			}
		}
		if next != "" {
			return "execute workflow phase " + next
		}
		return completeWorkflowNextAction
	}
	return "execute workflow phase " + phase
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

func contextForStart(root, task string) ContextQueryResult {
	context, err := QueryContext(root, ContextQueryOptions{Task: task, Limit: 12})
	if err == nil {
		return context
	}
	return ContextQueryResult{
		Status:   "warning",
		RepoRoot: root,
		Task:     task,
		Warnings: []string{err.Error()},
	}
}

func startExecutionContract(status map[string]any, participants []string, assignments []ParticipantAssignment, taskTier TaskTierResult, context ContextQueryResult) StartExecutionContract {
	return StartExecutionContract{
		RunDir:           stringValue(status["runDir"]),
		CheckpointPath:   stringValue(status["checkpointPath"]),
		TodoPath:         stringValue(status["todoPath"]),
		CurrentPhase:     stringValue(status["currentPhase"]),
		NextPhase:        stringValue(status["nextPhase"]),
		TaskTier:         taskTier,
		Context:          context,
		Participants:     participants,
		Assignments:      assignments,
		NextAction:       fallbackString(stringValue(status["nextAction"]), "execute the next workflow phase and update checkpoint.json"),
		NextVerification: fallbackString(stringValue(status["nextVerification"]), "run $RUNWEAVER_BIN workflow verify --repo . --resume latest before final response"),
		QualityGates:     qualityGatesForStart(context),
		PostReview:       postImplementationReviewChecklist(context),
		PhaseCompletion:  "after each finished phase run: $RUNWEAVER_BIN workflow update --repo . --resume latest --phase <phase> --complete-phase --verification \"<command/result>\"",
		TerminalRule:     "do not send a final response while checkpoint status is in_progress after successful verification; complete all workflow phases with --complete-phase or record a blocker with --blocker and nextVerification",
		ResumeStrategy:   "automatic via $RUNWEAVER_BIN start; use $RUNWEAVER_BIN workflow run --resume latest --status only as a diagnostic",
	}
}

func qualityGatesForStart(context ContextQueryResult) []string {
	gates := []string{
		"write or update the narrowest focused test before production edits when the task changes behavior",
		"record exact verification commands and results in the workflow checkpoint before final response",
	}
	if contextLooksFrontend(context) {
		gates = append(gates,
			"when extracting a UI component, pure helper, summary, panel, card, or dialog section, add a separate focused test for the extracted component/helper instead of relying only on parent-surface coverage",
			"run a focused lint/static check for changed component/helper files when the repository has local lint or static-analysis tooling",
			"if shared frontend types, props, or page/container wiring are touched, run page-level verification for the affected pages/containers named by context query",
			"for frontend-only tasks, review changed files and verify server/API/database/runtime worker files were not modified",
		)
	}
	if contextLooksBackend(context) {
		gates = append(gates,
			"for backend/API changes, verify request validation, auth/permission boundaries, status codes, response contracts, and persistence side effects",
		)
	}
	return gates
}

func postImplementationReviewChecklist(context ContextQueryResult) []string {
	checklist := []string{
		"compare original task, selected workflow, selected participants, changed files, and checkpoint state before final response",
		"review behavioral correctness plus rejected/paused paths; record findings with workflow update when relevant",
		"prefer existing focused tests and verification commands; avoid broad new test scaffolding unless task evidence requires it",
	}
	if contextLooksFrontend(context) {
		checklist = append(checklist,
			"for UI changes check loading, disabled, empty, error, and success states",
			"check stale async/race outcomes so late results cannot overwrite newer user state",
			"when adding a compact summary, panel, card, dialog section, or repeated UI block, prefer a cohesive UI component/helper instead of inlining new structure into an already large component",
			"split pure helpers into separate files when local lint/static rules disallow mixed helper and component exports",
			"if the task is frontend-only, verify server/API files were not changed unless evidence required it",
		)
	}
	if contextLooksBackend(context) {
		checklist = append(checklist, "for API/backend changes check request validation, auth/permission boundaries, status codes, and response contracts")
	}
	return checklist
}

func contextLooksFrontend(context ContextQueryResult) bool {
	for _, file := range context.Files {
		path := strings.ToLower(filepath.ToSlash(file.Path))
		if file.Category == "ui" || strings.Contains(path, "/components/") || strings.Contains(path, "/pages/") || strings.Contains(path, "/app/") {
			return true
		}
	}
	return hasAnySelectionToken(tokenizeSelectionText(context.Task), []string{"card", "component", "dialog", "frontend", "page", "ui"})
}

func contextLooksBackend(context ContextQueryResult) bool {
	if taskForbidsBackendChanges(context.Task) {
		for _, file := range context.Files {
			if contextHitLooksBackend(file) {
				return true
			}
		}
		return false
	}
	for _, file := range context.Files {
		path := strings.ToLower(filepath.ToSlash(file.Path))
		if file.Category == "route" || file.Category == "service" || strings.Contains(path, "/api/") || strings.Contains(path, "/controllers/") {
			return true
		}
	}
	return hasAnySelectionToken(tokenizeSelectionText(context.Task), []string{"api", "backend", "controller", "endpoint", "route", "server"})
}

func contextHitLooksBackend(file ContextFileHit) bool {
	return contextFileLooksBackend(FileInventoryItem{
		Path:     file.Path,
		Category: file.Category,
		Language: file.Language,
	})
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
