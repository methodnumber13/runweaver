package aitools

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func runtimeExecutionPreflight(root string, opts WorkflowExecuteOptions) RuntimeDiscoveryResult {
	provider, ok := RuntimeProviderByID(opts.Runtime)
	if !ok {
		return RuntimeDiscoveryResult{
			ID:     opts.Runtime,
			Status: "error",
			Issues: []string{"unsupported runtime"},
		}
	}
	result := discoverRuntime(root, provider)
	result.Binary = selectedRuntimeBinary(opts)
	result.BinaryFound = false
	result.BinaryPath = ""
	if path, err := exec.LookPath(result.Binary); err == nil {
		result.BinaryFound = true
		result.BinaryPath = path
	} else {
		result.Issues = Unique(append(result.Issues, result.Binary+" is not available on PATH"))
	}
	if result.BinaryFound && (hasAnyReadable(result.ConfigFiles) || hasAnyReadable(result.AuthFiles) || hasAnyReadable(result.MetadataFiles)) {
		result.Ready = true
		result.Status = "ok"
	} else if result.BinaryFound {
		result.Status = "warning"
	} else {
		result.Ready = false
		result.Status = "warning"
	}
	return result
}

func selectedRuntimeBinary(opts WorkflowExecuteOptions) string {
	switch opts.Runtime {
	case RuntimeCodex:
		return opts.CodexBin
	case RuntimeClaude:
		return opts.ClaudeBin
	default:
		return opts.OpencodeBin
	}
}

func workflowPlanForExecution(root string, opts WorkflowExecuteOptions) (WorkflowRunSummary, error) {
	if opts.Resume != "" {
		status, err := WorkflowStatus(root, opts.Resume)
		if err != nil {
			return WorkflowRunSummary{}, err
		}
		workflow, _ := status["workflow"].(string)
		task, _ := status["task"].(string)
		runDir, _ := status["runDir"].(string)
		if workflow == "" || runDir == "" {
			return WorkflowRunSummary{}, fmt.Errorf("workflow checkpoint is missing workflow or runDir")
		}
		phases := phaseIDsFromRunPlan(root, runDir)
		return WorkflowRunSummary{
			Workflow:       workflow,
			Task:           task,
			RunDir:         runDir,
			CheckpointPath: filepath.ToSlash(filepath.Join(runDir, "checkpoint.json")),
			TodoPath:       filepath.ToSlash(filepath.Join(runDir, "todo.md")),
			Phases:         phases,
			Status:         "resumed",
		}, nil
	}
	return PlanWorkflow(root, opts.WorkflowPath, opts.Task)
}
