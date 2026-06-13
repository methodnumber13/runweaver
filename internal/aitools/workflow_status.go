package aitools

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
	"os"
	"path/filepath"
	"strings"
)

// WorkflowStatus loads checkpoint state for the latest run or an explicit run path.
func WorkflowStatus(repoPath, resume string) (map[string]any, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return nil, err
	}
	runDir, err := resolveWorkflowRunDir(root, resume)
	if err != nil {
		return nil, err
	}
	var checkpoint WorkflowCheckpoint
	checkpointPath := filepath.Join(runDir, "checkpoint.json")
	if err := ReadJSON(checkpointPath, &checkpoint); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("workflow checkpoint not found at %s", rel(root, checkpointPath))
		}
		return nil, fmt.Errorf("load workflow checkpoint %s: %w", rel(root, checkpointPath), err)
	}
	return checkpointStatusMap(checkpoint, rel(root, runDir))
}

func resolveWorkflowRunDir(root, resume string) (string, error) {
	if resume == "" || resume == "latest" {
		var latest WorkflowLatest
		latestPath := statepath.WorkflowLatestPath(root)
		if err := ReadJSON(latestPath, &latest); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("no latest workflow run found; create one with runweaver workflow run --workflow <file> --task <text>")
			} else {
				return "", fmt.Errorf("cannot read latest workflow run %s: %w", rel(root, latestPath), err)
			}
		}
		resume = latest.RunDir
	}
	return resolveWorkflowRunPath(root, resume)
}

func resolveWorkflowRunPath(root, resume string) (string, error) {
	resume = strings.TrimSpace(resume)
	if resume == "" {
		return "", fmt.Errorf("resume run is required")
	}
	if filepath.IsAbs(resume) {
		return "", fmt.Errorf("workflow resume path must stay under .runweaver/tmp/swarm-runs: absolute paths are not allowed")
	}
	cleanResume := filepath.Clean(resume)
	if cleanResume == "." || strings.HasPrefix(cleanResume, "..") {
		return "", fmt.Errorf("workflow resume path must stay under .runweaver/tmp/swarm-runs")
	}
	return resolveWorkflowRunPathUnderRoots(root, cleanResume)
}

func resolveWorkflowRunPathUnderRoots(root, cleanResume string) (string, error) {
	canonicalRootRel := filepath.Clean(filepath.FromSlash(statepath.TmpRel("swarm-runs")))
	if cleanResume == canonicalRootRel {
		return "", fmt.Errorf("workflow resume path must point to a run directory under .runweaver/tmp/swarm-runs")
	}
	if strings.HasPrefix(cleanResume, canonicalRootRel+string(os.PathSeparator)) {
		runDir := filepath.Clean(filepath.Join(root, cleanResume))
		if err := validateWorkflowRunPath(statepath.WorkflowRunsRoot(root), runDir); err != nil {
			return "", err
		}
		return runDir, nil
	}
	if !strings.Contains(cleanResume, string(os.PathSeparator)) {
		return filepath.Join(statepath.WorkflowRunsRoot(root), cleanResume), nil
	}
	return "", fmt.Errorf("workflow resume path must stay under .runweaver/tmp/swarm-runs")
}

func validateWorkflowRunPath(parent, runDir string) error {
	runDir = filepath.Clean(runDir)
	if !pathInside(parent, runDir) {
		return fmt.Errorf("workflow resume path must stay under .runweaver/tmp/swarm-runs")
	}
	if realParent, parentErr := filepath.EvalSymlinks(parent); parentErr == nil {
		if realRun, runErr := filepath.EvalSymlinks(runDir); runErr == nil && !pathInside(realParent, realRun) {
			return fmt.Errorf("workflow resume path must stay under .runweaver/tmp/swarm-runs")
		}
	}
	return nil
}

func checkpointStatusMap(checkpoint WorkflowCheckpoint, runDir string) (map[string]any, error) {
	data, err := json.Marshal(checkpoint)
	if err != nil {
		return nil, fmt.Errorf("marshal workflow checkpoint status: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("unmarshal workflow checkpoint status: %w", err)
	}
	out["runDir"] = runDir
	return out, nil
}

func nextWorkflowPhase(root, runDir, current string) (string, error) {
	var plan WorkflowPlanFile
	planPath := filepath.Join(runDir, "plan.json")
	if err := ReadJSON(planPath, &plan); err != nil {
		return "", fmt.Errorf("load workflow plan %s for phase transition: %w", rel(root, planPath), err)
	}
	for i, phase := range plan.Workflow.Phases {
		if phase.ID == current && i+1 < len(plan.Workflow.Phases) {
			return plan.Workflow.Phases[i+1].ID, nil
		}
		if phase.ID == current {
			return "", nil
		}
	}
	return "", fmt.Errorf("workflow phase %s was not found in plan %s", current, rel(root, planPath))
}

func pathInside(parent, child string) bool {
	relPath, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	if err != nil || relPath == "." || filepath.IsAbs(relPath) {
		return false
	}
	return relPath != ".." && !strings.HasPrefix(relPath, ".."+string(os.PathSeparator))
}
