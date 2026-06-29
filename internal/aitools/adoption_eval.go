package aitools

import (
	"strings"
	"time"
)

// AdoptionEvalOptions configures local adoption smoke checks.
type AdoptionEvalOptions struct {
	Runtime          string
	Task             string
	SkipIndex        bool
	Live             bool
	OpencodeBin      string
	CodexBin         string
	ClaudeBin        string
	Model            string
	SkipGitRepoCheck bool
	Timeout          time.Duration
}

// AdoptionEvalResult combines adoption doctor output with a start smoke run.
type AdoptionEvalResult struct {
	Status          string                   `json:"status"`
	Ready           bool                     `json:"ready"`
	Live            bool                     `json:"live,omitempty"`
	Doctor          AdoptionDoctorResult     `json:"doctor"`
	Start           StartResult              `json:"start,omitempty"`
	ExecutionDryRun *WorkflowExecutionResult `json:"executionDryRun,omitempty"`
	Execution       *WorkflowExecutionResult `json:"execution,omitempty"`
	Checks          []AdoptionEvalCheck      `json:"checks,omitempty"`
}

// AdoptionEvalCheck is one local proof that a runtime can adopt RunWeaver.
type AdoptionEvalCheck struct {
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Summary     string   `json:"summary"`
	Evidence    []string `json:"evidence,omitempty"`
	NextActions []string `json:"nextActions,omitempty"`
}

// EvaluateAdoption verifies that generated metadata can route a task through start.
func EvaluateAdoption(repoPath string, opts AdoptionEvalOptions) (AdoptionEvalResult, error) {
	runtimeID := normalizeRuntimeID(opts.Runtime)
	if runtimeID == "" {
		runtimeID = RuntimeOpenCode
	}
	doctor, err := DoctorAdoption(repoPath, AdoptionDoctorOptions{Runtime: runtimeID})
	if err != nil {
		return AdoptionEvalResult{}, err
	}
	result := AdoptionEvalResult{
		Status: "ok",
		Ready:  doctor.Ready,
		Live:   opts.Live,
		Doctor: doctor,
	}
	if !doctor.Ready {
		result.Status = "warning"
		return result, nil
	}
	startRuntime := runtimeID
	if startRuntime == RuntimeAll {
		startRuntime = RuntimeOpenCode
	}
	task := strings.TrimSpace(opts.Task)
	if task == "" {
		task = "RunWeaver adoption smoke test"
	}
	start, err := StartWorkflow(repoPath, StartOptions{
		Task:      task,
		Runtime:   startRuntime,
		SkipIndex: opts.SkipIndex,
		ForceNew:  true,
	})
	if err != nil {
		return AdoptionEvalResult{}, err
	}
	result.Start = start
	execution, err := ExecuteWorkflow(repoPath, WorkflowExecuteOptions{
		Resume:           "latest",
		Runtime:          start.Runtime,
		DryRun:           !opts.Live,
		SkipModelCheck:   true,
		SkipGitRepoCheck: opts.SkipGitRepoCheck,
		OpencodeBin:      opts.OpencodeBin,
		CodexBin:         opts.CodexBin,
		ClaudeBin:        opts.ClaudeBin,
		Model:            opts.Model,
		Timeout:          opts.Timeout,
	})
	if err != nil {
		if opts.Live {
			attachExecutionPostCheck(start.RepoRoot, &execution)
			result.Execution = &execution
		} else {
			result.ExecutionDryRun = &execution
		}
		result.Checks = adoptionEvalChecks(doctor, start, execution, err)
		result.Ready = doctor.Ready && start.Ready && adoptionEvalChecksReady(result.Checks)
		result.Status = "warning"
		return result, nil
	}
	if opts.Live {
		result.Execution = &execution
	} else {
		result.ExecutionDryRun = &execution
	}
	result.Checks = adoptionEvalChecks(doctor, start, execution, nil)
	result.Ready = doctor.Ready && start.Ready && adoptionEvalChecksReady(result.Checks)
	if !result.Ready {
		result.Status = "warning"
	}
	return result, nil
}

func adoptionEvalChecks(doctor AdoptionDoctorResult, start StartResult, execution WorkflowExecutionResult, executionErr error) []AdoptionEvalCheck {
	return []AdoptionEvalCheck{
		firstActionContractEvalCheck(doctor),
		startSmokeEvalCheck(start),
		workflowStateEvalCheck(start),
		participantsEvalCheck(start),
		contextReturnedEvalCheck(start),
		runtimeDryRunEvalCheck(execution, executionErr),
	}
}

func attachExecutionPostCheck(root string, execution *WorkflowExecutionResult) {
	if execution == nil || execution.Plan.CheckpointPath == "" {
		return
	}
	postCheck := workflowExecutionPostCheck(root, execution.Plan)
	execution.PostCheck = &postCheck
}

func firstActionContractEvalCheck(doctor AdoptionDoctorResult) AdoptionEvalCheck {
	for _, runtime := range doctor.Runtimes {
		for _, check := range runtime.Checks {
			if check.Name == "startup-contract" && check.Status == "ok" {
				return AdoptionEvalCheck{Name: "first-action-contract", Status: "ok", Summary: "Runtime instructions require runweaver start first", Evidence: []string{runtime.ID}}
			}
		}
	}
	return AdoptionEvalCheck{Name: "first-action-contract", Status: "error", Summary: "No runtime startup contract requiring runweaver start was verified"}
}

func startSmokeEvalCheck(start StartResult) AdoptionEvalCheck {
	if start.Ready && start.Action != "" {
		return AdoptionEvalCheck{Name: "start-smoke", Status: "ok", Summary: "runweaver start returned an execution contract", Evidence: []string{start.Action, start.Workflow.Workflow}}
	}
	return AdoptionEvalCheck{Name: "start-smoke", Status: "error", Summary: "runweaver start did not produce a ready execution contract"}
}

func workflowStateEvalCheck(start StartResult) AdoptionEvalCheck {
	if start.Workflow.CheckpointPath != "" && start.Workflow.TodoPath != "" {
		return AdoptionEvalCheck{Name: "workflow-state", Status: "ok", Summary: "Workflow checkpoint and todo paths were created", Evidence: []string{start.Workflow.CheckpointPath, start.Workflow.TodoPath}}
	}
	return AdoptionEvalCheck{Name: "workflow-state", Status: "error", Summary: "Workflow state artifacts were not created"}
}

func participantsEvalCheck(start StartResult) AdoptionEvalCheck {
	if len(start.Participants.Participants) > 0 {
		return AdoptionEvalCheck{Name: "participants-recorded", Status: "ok", Summary: "runweaver start selected participants", Evidence: start.Participants.Participants}
	}
	return AdoptionEvalCheck{Name: "participants-recorded", Status: "error", Summary: "No participants were selected by runweaver start"}
}

func contextReturnedEvalCheck(start StartResult) AdoptionEvalCheck {
	if start.ExecutionContract.Context.Status == "success" {
		return AdoptionEvalCheck{Name: "context-returned", Status: "ok", Summary: "Task-scoped context was returned", Evidence: contextEvidence(start.ExecutionContract.Context)}
	}
	return AdoptionEvalCheck{Name: "context-returned", Status: "warning", Summary: "Task-scoped context was not available", Evidence: start.ExecutionContract.Context.Warnings}
}

func runtimeDryRunEvalCheck(execution WorkflowExecutionResult, err error) AdoptionEvalCheck {
	if err != nil {
		if execution.Executed && execution.PostCheck != nil && execution.PostCheck.CheckpointState == "complete" && execution.PostCheck.CompletedPhases > 0 {
			return AdoptionEvalCheck{
				Name:    "runtime-execution",
				Status:  "warning",
				Summary: "Runtime process ended with an error after durable workflow state completed",
				Evidence: append([]string{
					err.Error(),
					"checkpoint status: " + execution.PostCheck.CheckpointState,
				}, execution.Command...),
				NextActions: []string{"Inspect runtime stdout/stderr if the runtime repeatedly fails to exit after writing a complete checkpoint."},
			}
		}
		return AdoptionEvalCheck{Name: "runtime-execution", Status: "error", Summary: "Runtime execution command could not be prepared", Evidence: []string{err.Error()}}
	}
	if len(execution.Command) > 0 && execution.DryRun && !execution.Executed {
		return AdoptionEvalCheck{Name: "runtime-dry-run", Status: "ok", Summary: "Runtime execution command was prepared without launching the model", Evidence: execution.Command}
	}
	if len(execution.Command) > 0 && execution.Executed {
		check := AdoptionEvalCheck{Name: "runtime-execution", Status: "ok", Summary: "Runtime execution completed", Evidence: execution.Command}
		if execution.PostCheck != nil && execution.PostCheck.Status == "warning" {
			check.Status = "warning"
			check.Summary = "Runtime execution completed but workflow checkpoint needs attention"
			check.NextActions = execution.PostCheck.Warnings
		}
		return check
	}
	return AdoptionEvalCheck{Name: "runtime-execution", Status: "error", Summary: "Runtime execution did not return a command"}
}

func adoptionEvalChecksReady(checks []AdoptionEvalCheck) bool {
	for _, check := range checks {
		if check.Status == "error" {
			return false
		}
	}
	return true
}

func contextEvidence(context ContextQueryResult) []string {
	var evidence []string
	for _, file := range context.Files {
		evidence = append(evidence, file.Path)
		if len(evidence) >= 5 {
			break
		}
	}
	return evidence
}
