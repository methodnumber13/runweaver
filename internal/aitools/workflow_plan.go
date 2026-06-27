package aitools

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// PlanWorkflow creates a durable plan, checkpoint, todo list, and latest pointer.
func PlanWorkflow(repoPath, workflowPath, task string) (WorkflowRunSummary, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return WorkflowRunSummary{}, err
	}
	if task == "" {
		return WorkflowRunSummary{}, fmt.Errorf("workflow task is required")
	}
	if workflowPath == "" {
		workflowPath = statepath.WorkflowPath(root, "feature-delivery-swarm.json")
		if !Exists(workflowPath) && Exists(statepath.LegacyOpenCodeWorkflowPath(root, "feature-delivery-swarm.json")) {
			workflowPath = statepath.LegacyOpenCodeWorkflowPath(root, "feature-delivery-swarm.json")
		}
		if !Exists(workflowPath) && Exists(statepath.LegacyOpenCodeWorkflowPath(root, "feature-swarm.json")) {
			workflowPath = statepath.LegacyOpenCodeWorkflowPath(root, "feature-swarm.json")
		}
	}
	if !filepath.IsAbs(workflowPath) {
		workflowPath = filepath.Join(root, workflowPath)
	}
	var workflow WorkflowSpec
	if err := ReadJSON(workflowPath, &workflow); err != nil {
		return WorkflowRunSummary{}, fmt.Errorf("load workflow %s: %w", workflowPath, err)
	}
	if workflow.ID == "" {
		return WorkflowRunSummary{}, fmt.Errorf("workflow id is required in %s", workflowPath)
	}
	if err := validateWorkflowSpec(workflow); err != nil {
		return WorkflowRunSummary{}, fmt.Errorf("validate workflow %s: %w", workflowPath, err)
	}
	runID, runDir, err := createWorkflowRunDir(root, workflow.ID)
	if err != nil {
		return WorkflowRunSummary{}, err
	}
	if err := createWorkflowRunArtifacts(root, runDir, workflow); err != nil {
		return WorkflowRunSummary{}, err
	}
	phaseIDs := make([]string, 0, len(workflow.Phases))
	for _, phase := range workflow.Phases {
		phaseIDs = append(phaseIDs, phase.ID)
	}
	nextPhase := ""
	if len(phaseIDs) > 0 {
		nextPhase = phaseIDs[0]
	}
	plan := WorkflowPlanFile{
		SchemaVersion: 1,
		RunID:         runID,
		Workflow:      workflow,
		Task:          task,
		Status:        "planned",
		RunDir:        rel(root, runDir),
	}
	checkpoint := WorkflowCheckpoint{
		SchemaVersion:   1,
		RunID:           runID,
		Workflow:        workflow.ID,
		Task:            task,
		Status:          "planned",
		CompletedPhases: []string{},
		NextPhase:       nextPhase,
		UpdatedAt:       Now(),
	}
	if err := WriteJSON(filepath.Join(runDir, "plan.json"), plan); err != nil {
		return WorkflowRunSummary{}, err
	}
	if err := WriteJSON(filepath.Join(runDir, "checkpoint.json"), checkpoint); err != nil {
		return WorkflowRunSummary{}, err
	}
	if err := os.WriteFile(filepath.Join(runDir, "todo.md"), []byte(todoMarkdown(workflow, task)), 0o644); err != nil {
		return WorkflowRunSummary{}, fmt.Errorf("write workflow todo: %w", err)
	}
	event := WorkflowEvent{Type: "planned", RunID: runID, Workflow: workflow.ID, Task: task, At: Now()}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return WorkflowRunSummary{}, fmt.Errorf("marshal workflow event: %w", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "events.ndjson"), append(eventBytes, '\n'), 0o644); err != nil {
		return WorkflowRunSummary{}, fmt.Errorf("write workflow event log: %w", err)
	}
	latest := WorkflowLatest{RunID: runID, RunDir: rel(root, runDir), Workflow: workflow.ID, Task: task, UpdatedAt: Now()}
	if err := WriteJSON(statepath.WorkflowLatestPath(root), latest); err != nil {
		return WorkflowRunSummary{}, err
	}
	return WorkflowRunSummary{
		Workflow:       workflow.ID,
		Task:           task,
		RunDir:         rel(root, runDir),
		CheckpointPath: rel(root, filepath.Join(runDir, "checkpoint.json")),
		TodoPath:       rel(root, filepath.Join(runDir, "todo.md")),
		Phases:         phaseIDs,
		Status:         "planned",
	}, nil
}

func validateWorkflowSpec(workflow WorkflowSpec) error {
	if workflow.ID == "" {
		return fmt.Errorf("workflow id is required")
	}
	if workflow.MaxConcurrent < 0 {
		return fmt.Errorf("maxConcurrent cannot be negative")
	}
	if workflow.MaxParticipants < 0 {
		return fmt.Errorf("maxParticipants cannot be negative")
	}
	seen := map[string]bool{}
	for _, phase := range workflow.Phases {
		if strings.TrimSpace(phase.ID) == "" {
			return fmt.Errorf("phase id is required")
		}
		if seen[phase.ID] {
			return fmt.Errorf("duplicate phase id %s", phase.ID)
		}
		seen[phase.ID] = true
		if phase.Concurrency < 0 {
			return fmt.Errorf("phase %s concurrency cannot be negative", phase.ID)
		}
	}
	return nil
}

func createWorkflowRunDir(root, workflowID string) (string, string, error) {
	parent := statepath.WorkflowRunsRoot(root)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", "", fmt.Errorf("create workflow run parent directory %s: %w", parent, err)
	}
	base := time.Now().UTC().Format("20060102-150405-000000000") + "-" + sanitizeRunID(workflowID)
	for attempt := 0; attempt < 1000; attempt++ {
		runID := base
		if attempt > 0 {
			runID = fmt.Sprintf("%s-%03d", base, attempt)
		}
		runDir := filepath.Join(parent, runID)
		if err := os.Mkdir(runDir, 0o755); err != nil {
			if errors.Is(err, os.ErrExist) {
				continue
			}
			return "", "", fmt.Errorf("create workflow run directory %s: %w", runDir, err)
		}
		return runID, runDir, nil
	}
	return "", "", fmt.Errorf("create workflow run directory: exhausted unique run id attempts for workflow %s", workflowID)
}

func createWorkflowRunArtifacts(root, runDir string, workflow WorkflowSpec) error {
	if err := os.MkdirAll(filepath.Join(runDir, "agents"), 0o755); err != nil {
		return fmt.Errorf("create workflow agent artifact directory %s: %w", rel(root, filepath.Join(runDir, "agents")), err)
	}
	for _, phase := range workflow.Phases {
		phaseDir := filepath.Join(runDir, "phases", sanitizeRunID(phase.ID))
		if err := os.MkdirAll(phaseDir, 0o755); err != nil {
			return fmt.Errorf("create workflow phase artifact directory %s: %w", rel(root, phaseDir), err)
		}
		handoff := "# " + phase.ID + " Handoff\n\n"
		if err := os.WriteFile(filepath.Join(phaseDir, "handoff.md"), []byte(handoff), 0o644); err != nil {
			return fmt.Errorf("write workflow phase handoff %s: %w", rel(root, filepath.Join(phaseDir, "handoff.md")), err)
		}
		notes := "# " + phase.ID + " Notes\n\n"
		if err := os.WriteFile(filepath.Join(phaseDir, "notes.md"), []byte(notes), 0o644); err != nil {
			return fmt.Errorf("write workflow phase notes %s: %w", rel(root, filepath.Join(phaseDir, "notes.md")), err)
		}
		if err := os.WriteFile(filepath.Join(phaseDir, "verification.jsonl"), nil, 0o644); err != nil {
			return fmt.Errorf("write workflow phase verification log %s: %w", rel(root, filepath.Join(phaseDir, "verification.jsonl")), err)
		}
	}
	return nil
}

var runNamePattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func sanitizeRunID(value string) string {
	value = strings.Trim(runNamePattern.ReplaceAllString(value, "-"), "-")
	if value == "" {
		return "workflow"
	}
	return value
}
