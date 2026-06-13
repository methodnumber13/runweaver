package aitools

import (
	"encoding/json"
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
	"os"
	"path/filepath"
	"strings"
)

// UpdateWorkflow persists phase progress, evidence, and append-only events.
func UpdateWorkflow(repoPath string, opts WorkflowUpdateOptions) (map[string]any, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return nil, err
	}
	runDir, err := resolveWorkflowRunDir(root, opts.Resume)
	if err != nil {
		return nil, err
	}
	checkpointPath := filepath.Join(runDir, "checkpoint.json")
	var checkpoint WorkflowCheckpoint
	if err := ReadJSON(checkpointPath, &checkpoint); err != nil {
		return nil, fmt.Errorf("load workflow checkpoint %s: %w", rel(root, checkpointPath), err)
	}
	phase := strings.TrimSpace(opts.Phase)
	if phase != "" {
		checkpoint.CurrentPhase = phase
		checkpoint.NextPhase = phase
	}
	if opts.Status != "" {
		checkpoint.Status = strings.TrimSpace(opts.Status)
	} else if phase != "" && checkpoint.Status == "planned" {
		checkpoint.Status = "in_progress"
	}
	if len(opts.Participants) > 0 {
		checkpoint.Participants = Unique(append(checkpoint.Participants, compactStrings(opts.Participants)...))
	}
	if len(opts.ParticipantRationale) > 0 {
		checkpoint.ParticipantRationale = Unique(append(checkpoint.ParticipantRationale, compactStrings(opts.ParticipantRationale)...))
	}
	if len(opts.Findings) > 0 {
		checkpoint.Findings = Unique(append(checkpoint.Findings, compactStrings(opts.Findings)...))
	}
	if len(opts.Decisions) > 0 {
		checkpoint.Decisions = Unique(append(checkpoint.Decisions, compactStrings(opts.Decisions)...))
	}
	if len(opts.FilesRead) > 0 {
		checkpoint.FilesRead = Unique(append(checkpoint.FilesRead, compactStrings(opts.FilesRead)...))
	}
	if len(opts.FilesChanged) > 0 {
		checkpoint.FilesChanged = Unique(append(checkpoint.FilesChanged, compactStrings(opts.FilesChanged)...))
	}
	if len(opts.Artifacts) > 0 {
		checkpoint.Artifacts = Unique(append(checkpoint.Artifacts, compactStrings(opts.Artifacts)...))
	}
	if strings.TrimSpace(opts.NextAction) != "" {
		checkpoint.NextAction = strings.TrimSpace(opts.NextAction)
	}
	if len(opts.Verification) > 0 {
		checkpoint.Verification = Unique(append(checkpoint.Verification, compactStrings(opts.Verification)...))
	}
	if len(opts.VerificationResults) > 0 {
		checkpoint.VerificationResults = Unique(append(checkpoint.VerificationResults, compactStrings(opts.VerificationResults)...))
	}
	if len(opts.Blockers) > 0 {
		checkpoint.Blockers = Unique(append(checkpoint.Blockers, compactStrings(opts.Blockers)...))
	}
	nextPhase := ""
	if opts.CompletePhase && phase != "" {
		var err error
		nextPhase, err = nextWorkflowPhase(root, runDir, phase)
		if err != nil {
			return nil, err
		}
		checkpoint.CompletedPhases = Unique(append(checkpoint.CompletedPhases, phase))
		checkpoint.NextPhase = nextPhase
		if checkpoint.NextPhase == "" {
			checkpoint.Status = "complete"
			checkpoint.CurrentPhase = ""
		} else {
			checkpoint.CurrentPhase = checkpoint.NextPhase
		}
	}
	checkpoint.UpdatedAt = Now()
	if err := WriteJSON(checkpointPath, checkpoint); err != nil {
		return nil, fmt.Errorf("write workflow checkpoint %s: %w", rel(root, checkpointPath), err)
	}
	if err := syncWorkflowTodo(root, runDir, checkpoint); err != nil {
		return nil, err
	}
	eventPhase := phase
	if eventPhase == "" {
		eventPhase = checkpoint.CurrentPhase
	}
	event := WorkflowEvent{
		Type:                 "updated",
		RunID:                checkpoint.RunID,
		Workflow:             checkpoint.Workflow,
		Task:                 checkpoint.Task,
		Phase:                eventPhase,
		Status:               checkpoint.Status,
		Participants:         checkpoint.Participants,
		ParticipantRationale: compactStrings(opts.ParticipantRationale),
		Findings:             compactStrings(opts.Findings),
		Decisions:            compactStrings(opts.Decisions),
		FilesRead:            compactStrings(opts.FilesRead),
		FilesChanged:         compactStrings(opts.FilesChanged),
		Artifacts:            compactStrings(opts.Artifacts),
		NextAction:           strings.TrimSpace(opts.NextAction),
		Verification:         compactStrings(opts.Verification),
		VerificationResults:  compactStrings(opts.VerificationResults),
		Blockers:             compactStrings(opts.Blockers),
		At:                   Now(),
	}
	if err := appendWorkflowEvent(root, runDir, event); err != nil {
		return nil, err
	}
	latest := WorkflowLatest{RunID: checkpoint.RunID, RunDir: rel(root, runDir), Workflow: checkpoint.Workflow, Task: checkpoint.Task, UpdatedAt: checkpoint.UpdatedAt}
	if err := WriteJSON(statepath.WorkflowLatestPath(root), latest); err != nil {
		return nil, err
	}
	return checkpointStatusMap(checkpoint, rel(root, runDir))
}

func syncWorkflowTodo(root, runDir string, checkpoint WorkflowCheckpoint) error {
	var plan WorkflowPlanFile
	planPath := filepath.Join(runDir, "plan.json")
	if err := ReadJSON(planPath, &plan); err != nil {
		return fmt.Errorf("load workflow plan %s for todo sync: %w", rel(root, planPath), err)
	}
	todo := todoMarkdownWithCheckpoint(plan.Workflow, checkpoint)
	todoPath := filepath.Join(runDir, "todo.md")
	if err := os.WriteFile(todoPath, []byte(todo), 0o644); err != nil {
		return fmt.Errorf("write workflow todo %s: %w", rel(root, todoPath), err)
	}
	return nil
}

func appendWorkflowEvent(root, runDir string, event WorkflowEvent) error {
	if event.Type == "" {
		return nil
	}
	if event.At == "" {
		event.At = Now()
	}
	path := runDir
	if !filepath.IsAbs(path) {
		var err error
		path, err = resolveWorkflowRunPath(root, runDir)
		if err != nil {
			return err
		}
	} else if !pathInside(statepath.WorkflowRunsRoot(root), path) {
		return fmt.Errorf("workflow resume path must stay under .runweaver/tmp/swarm-runs")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create workflow run directory: %w", err)
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal workflow event: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(path, "events.ndjson"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open workflow event log: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append workflow event: %w", err)
	}
	return nil
}

func countWorkflowEvents(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event WorkflowEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return count, fmt.Errorf("parse workflow event line %d: %w", count+1, err)
		}
		count++
	}
	if count == 0 {
		return 0, fmt.Errorf("event log is empty")
	}
	return count, nil
}
