package aitools

import (
	"fmt"
	"path/filepath"
	"strings"
)

func autoCompleteWorkflowFromVerificationEvidence(root, runDir string, workflow WorkflowSpec, checkpoint *WorkflowCheckpoint) (bool, error) {
	if checkpoint == nil || checkpoint.Status == "complete" || len(checkpoint.Blockers) > 0 {
		return false, nil
	}
	phaseEvidence := workflowPhaseVerificationEvidence(runDir, workflow)
	evidence := append([]string{}, checkpoint.VerificationResults...)
	evidence = append(evidence, phaseEvidence...)
	if !hasPositiveVerificationEvidence(evidence) || hasBlockingVerificationFailure(evidence) {
		return false, nil
	}
	if workflowHasWritePhase(workflow) && len(changedWorkflowFiles(checkpoint.FilesChanged)) == 0 {
		return false, nil
	}
	checkpoint.CompletedPhases = workflowPhaseIDs(workflow)
	checkpoint.Status = "complete"
	checkpoint.CurrentPhase = ""
	checkpoint.NextPhase = ""
	normalizeCompletedCheckpointState(checkpoint)
	normalizeWorkflowVerificationEvidence(checkpoint, phaseEvidence...)
	applyWorkflowIndexFreshness(root, checkpoint)
	checkpoint.UpdatedAt = Now()
	checkpointPath := filepath.Join(runDir, "checkpoint.json")
	if err := WriteJSON(checkpointPath, checkpoint); err != nil {
		return false, fmt.Errorf("write auto-completed workflow checkpoint %s: %w", rel(root, checkpointPath), err)
	}
	if err := syncWorkflowTodo(root, runDir, *checkpoint); err != nil {
		return false, err
	}
	if err := writeWorkflowCurrent(root, runDir, *checkpoint); err != nil {
		return false, err
	}
	if err := appendWorkflowEvent(root, runDir, WorkflowEvent{
		Type:                "auto-completed",
		RunID:               checkpoint.RunID,
		Workflow:            checkpoint.Workflow,
		Task:                checkpoint.Task,
		Status:              checkpoint.Status,
		FilesChanged:        checkpoint.FilesChanged,
		Verification:        checkpoint.Verification,
		VerificationResults: checkpoint.VerificationResults,
		At:                  Now(),
	}); err != nil {
		return false, err
	}
	return true, nil
}

func addVerificationQualityChecks(workflow WorkflowSpec, checkpoint WorkflowCheckpoint, result *WorkflowVerificationResult) bool {
	if hasBlockingVerificationFailure(checkpoint.VerificationResults) {
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:        "verification-failures",
			Status:      "warning",
			Summary:     "Recorded verification output contains failure-like diagnostics",
			Evidence:    Limit(matchingVerificationFailures(checkpoint.VerificationResults), 6),
			NextActions: []string{"Fix the failing verification command before completing the workflow, or record a blocker if it cannot be fixed safely."},
		})
		return false
	}
	if workflowHasWritePhase(workflow) && len(changedWorkflowFiles(checkpoint.FilesChanged)) > 0 && !hasFocusedStaticEvidence(checkpoint) {
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:     "focused-static-verification",
			Status:   "warning",
			Summary:  "Changed files require a recorded focused lint/static/type/build check",
			Evidence: Limit(changedWorkflowFiles(checkpoint.FilesChanged), 8),
			NextActions: []string{
				"Run the narrowest available lint/static/type/build check for changed files.",
				"Record both --verification and --verification-result. If broad lint has unrelated failures, record the focused changed-file check separately.",
			},
		})
		return false
	}
	if len(changedTestFiles(checkpoint.FilesChanged)) > 0 && !hasAssertionQualityEvidence(checkpoint) {
		addWorkflowVerificationCheck(result, WorkflowVerificationCheck{
			Name:     "assertion-quality",
			Status:   "warning",
			Summary:  "Changed tests require recorded assertion-quality review evidence",
			Evidence: Limit(changedTestFiles(checkpoint.FilesChanged), 8),
			NextActions: []string{
				"Compare changed assertions against the original acceptance criteria.",
				"Record a finding or verification result that assertions prove the required values/behavior and were not weakened to existence-only checks.",
			},
		})
		return false
	}
	return true
}

func hasFocusedStaticEvidence(checkpoint WorkflowCheckpoint) bool {
	text := strings.ToLower(strings.Join(append(append([]string{}, checkpoint.Verification...), checkpoint.VerificationResults...), "\n"))
	for _, marker := range []string{
		"lint",
		"static",
		"typecheck",
		"type check",
		"compile",
		"build",
		"format",
		"vet",
		"analysis",
		"analyze",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func hasAssertionQualityEvidence(checkpoint WorkflowCheckpoint) bool {
	text := strings.ToLower(strings.Join(append(append(append([]string{}, checkpoint.Findings...), checkpoint.Decisions...), append(checkpoint.VerificationResults, checkpoint.ResidualRisks...)...), "\n"))
	for _, marker := range []string{
		"assertion quality",
		"assertions prove",
		"assertions cover",
		"acceptance criteria",
		"acceptance criterion",
		"requirement-by-requirement",
		"behavioral assertion",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func changedWorkflowFiles(paths []string) []string {
	var out []string
	for _, path := range compactStrings(paths) {
		lower := strings.ToLower(filepath.ToSlash(path))
		if lower == "" || strings.HasPrefix(lower, ".runweaver/") || strings.HasPrefix(lower, ".opencode/") || strings.HasPrefix(lower, ".omo/") {
			continue
		}
		if strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".txt") || strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".gif") || strings.HasSuffix(lower, ".svg") {
			continue
		}
		out = append(out, path)
	}
	return Unique(out)
}

func changedTestFiles(paths []string) []string {
	var out []string
	for _, path := range changedWorkflowFiles(paths) {
		lower := strings.ToLower(filepath.ToSlash(path))
		base := filepath.Base(lower)
		if strings.Contains(lower, "/test/") ||
			strings.Contains(lower, "/tests/") ||
			strings.Contains(base, "_test.") ||
			strings.Contains(base, ".test.") ||
			strings.Contains(base, ".spec.") ||
			strings.Contains(base, "test_") {
			out = append(out, path)
		}
	}
	return Unique(out)
}

func hasPositiveVerificationEvidence(entries []string) bool {
	for _, entry := range compactStrings(entries) {
		lower := strings.ToLower(entry)
		for _, marker := range []string{" passed", "passed ", " pass", "success", "ok", "verified", "complete"} {
			if strings.Contains(lower, marker) {
				return true
			}
		}
	}
	return false
}

func hasBlockingVerificationFailure(entries []string) bool {
	return len(matchingVerificationFailures(entries)) > 0
}

func matchingVerificationFailures(entries []string) []string {
	var failures []string
	for _, entry := range compactStrings(entries) {
		for _, line := range strings.Split(entry, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || !verificationLineLooksFailure(line) {
				continue
			}
			failures = append(failures, truncateResidualRisk(line))
		}
	}
	return Unique(failures)
}

func verificationLineLooksFailure(line string) bool {
	lower := strings.ToLower(line)
	for _, clean := range []string{
		"0 error",
		"0 errors",
		"0 fail",
		"0 failed",
		"0 failures",
		"no error",
		"no errors",
		"no fail",
		"no failures",
		"without error",
		"without errors",
	} {
		if strings.Contains(lower, clean) {
			return false
		}
	}
	for _, marker := range []string{
		"failed",
		"failure",
		"error:",
		"fatal:",
		"exit code 1",
		"exit status 1",
		"parse error",
		"syntax error",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
