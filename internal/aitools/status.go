package aitools

import (
	"errors"
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
	"os"
	"path/filepath"
)

// RunWeaverStatusResult summarizes repository-local RunWeaver readiness.
type RunWeaverStatusResult struct {
	Status          string   `json:"status"`
	Ready           bool     `json:"ready"`
	RepoRoot        string   `json:"repoRoot"`
	Initialized     bool     `json:"initialized"`
	IndexPath       string   `json:"indexPath,omitempty"`
	ContextPath     string   `json:"contextPath,omitempty"`
	LatestWorkflow  bool     `json:"latestWorkflow"`
	RunID           string   `json:"runId,omitempty"`
	Workflow        string   `json:"workflow,omitempty"`
	Task            string   `json:"task,omitempty"`
	WorkflowStatus  string   `json:"workflowStatus,omitempty"`
	CurrentPhase    string   `json:"currentPhase,omitempty"`
	NextPhase       string   `json:"nextPhase,omitempty"`
	RunDir          string   `json:"runDir,omitempty"`
	CheckpointPath  string   `json:"checkpointPath,omitempty"`
	TodoPath        string   `json:"todoPath,omitempty"`
	CurrentPath     string   `json:"currentPath,omitempty"`
	Participants    []string `json:"participants,omitempty"`
	NextAction      string   `json:"nextAction,omitempty"`
	Blockers        []string `json:"blockers,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
}

// RunWeaverStatus reads local RunWeaver metadata without mutating the repository.
func RunWeaverStatus(repoPath string) (RunWeaverStatusResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return RunWeaverStatusResult{}, err
	}
	result := RunWeaverStatusResult{
		Status:      "warning",
		Ready:       false,
		RepoRoot:    root,
		Initialized: Exists(filepath.Join(root, statepath.WorkflowDir)),
		IndexPath:   relIfExists(root, filepath.Join(statepath.IndexRootPath(root), "repo-index.json")),
		ContextPath: relIfExists(root, filepath.Join(statepath.IndexRootPath(root), "repo-context.md")),
		Recommendations: []string{
			"run runweaver index --repo . --changed-only --prune",
			"create a workflow with runweaver workflow run --workflow .runweaver/workflows/feature-delivery-swarm.json --task \"<task>\"",
		},
	}

	latestPath := statepath.WorkflowLatestPath(root)
	var latest WorkflowLatest
	if err := ReadJSON(latestPath, &latest); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		return RunWeaverStatusResult{}, fmt.Errorf("read workflow latest %s: %w", rel(root, latestPath), err)
	}
	runDir, err := resolveWorkflowRunDir(root, latest.RunDir)
	if err != nil {
		return RunWeaverStatusResult{}, err
	}
	var checkpoint WorkflowCheckpoint
	checkpointPath := filepath.Join(runDir, "checkpoint.json")
	if err := ReadJSON(checkpointPath, &checkpoint); err != nil {
		return RunWeaverStatusResult{}, fmt.Errorf("read workflow checkpoint %s: %w", rel(root, checkpointPath), err)
	}

	result.Status = "ok"
	result.Ready = true
	result.LatestWorkflow = true
	result.RunID = checkpoint.RunID
	result.Workflow = checkpoint.Workflow
	result.Task = checkpoint.Task
	result.WorkflowStatus = checkpoint.Status
	result.CurrentPhase = checkpoint.CurrentPhase
	result.NextPhase = checkpoint.NextPhase
	result.RunDir = rel(root, runDir)
	result.CheckpointPath = rel(root, checkpointPath)
	result.TodoPath = rel(root, filepath.Join(runDir, "todo.md"))
	result.CurrentPath = rel(root, statepath.TmpPath(root, workflowCurrentFile))
	result.Participants = checkpoint.Participants
	result.NextAction = checkpoint.NextAction
	result.Blockers = checkpoint.Blockers
	result.Recommendations = statusRecommendations(checkpoint)
	return result, nil
}

func relIfExists(root, path string) string {
	if !Exists(path) {
		return ""
	}
	return rel(root, path)
}

func statusRecommendations(checkpoint WorkflowCheckpoint) []string {
	if checkpoint.Status == "complete" {
		return []string{"workflow is complete; start a new workflow for unrelated work"}
	}
	if checkpoint.NextPhase != "" {
		return []string{"resume automatically from " + checkpoint.NextPhase, "keep checkpoint.json and current.md updated"}
	}
	if checkpoint.CurrentPhase != "" {
		return []string{"continue current phase " + checkpoint.CurrentPhase, "keep checkpoint.json and current.md updated"}
	}
	return []string{"inspect checkpoint.json and continue the workflow"}
}
