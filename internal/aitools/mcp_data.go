package aitools

import (
	"errors"
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
	"os"
	"path/filepath"
	"sort"
)

// RunWeaverCurrent reads the latest workflow mirror intended as an LLM resume entrypoint.
func RunWeaverCurrent(repoPath string) (RunWeaverCurrentResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return RunWeaverCurrentResult{}, err
	}
	status, err := RunWeaverStatus(root)
	if err != nil {
		return RunWeaverCurrentResult{}, err
	}
	currentPath := statepath.TmpPath(root, workflowCurrentFile)
	result := RunWeaverCurrentResult{
		Status:          "warning",
		Ready:           false,
		RepoRoot:        root,
		CurrentPath:     rel(root, currentPath),
		State:           status,
		Recommendations: status.Recommendations,
	}
	data, err := os.ReadFile(currentPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		return RunWeaverCurrentResult{}, fmt.Errorf("read current workflow %s: %w", rel(root, currentPath), err)
	}
	result.Status = "ok"
	result.Ready = true
	result.CurrentMarkdown = string(data)
	return result, nil
}

// ListRunWeaverWorkflows returns generated workflow templates for runtime tools.
func ListRunWeaverWorkflows(repoPath string) (WorkflowTemplateListResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return WorkflowTemplateListResult{}, err
	}
	workflowRoot := filepath.Join(root, statepath.WorkflowDir)
	result := WorkflowTemplateListResult{
		Status:    "ok",
		RepoRoot:  root,
		Root:      statepath.WorkflowDir,
		Workflows: []WorkflowTemplateSummary{},
	}
	entries, err := os.ReadDir(workflowRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		return WorkflowTemplateListResult{}, fmt.Errorf("read workflow directory %s: %w", rel(root, workflowRoot), err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(workflowRoot, entry.Name())
		var spec WorkflowSpec
		if err := ReadJSON(path, &spec); err != nil {
			return WorkflowTemplateListResult{}, fmt.Errorf("read workflow template %s: %w", rel(root, path), err)
		}
		result.Workflows = append(result.Workflows, workflowTemplateSummary(root, path, spec))
	}
	sort.Slice(result.Workflows, func(i, j int) bool {
		return result.Workflows[i].ID < result.Workflows[j].ID
	})
	return result, nil
}

func workflowTemplateSummary(root, path string, spec WorkflowSpec) WorkflowTemplateSummary {
	phases := make([]string, 0, len(spec.Phases))
	for _, phase := range spec.Phases {
		phases = append(phases, phase.ID)
	}
	return WorkflowTemplateSummary{
		ID:          spec.ID,
		Name:        spec.Name,
		Description: spec.Description,
		Path:        rel(root, path),
		Phases:      phases,
	}
}
