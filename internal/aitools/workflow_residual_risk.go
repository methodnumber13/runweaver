package aitools

import "strings"

const completeWorkflowNextAction = "workflow complete"

func residualRisksFromVerificationResults(results []string) []string {
	var risks []string
	for _, result := range results {
		for _, line := range strings.Split(result, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || !verificationLineLooksWarning(line) {
				continue
			}
			risks = append(risks, "verification warning retained as residual risk: "+truncateResidualRisk(line))
		}
	}
	return Unique(risks)
}

func verificationLineLooksWarning(line string) bool {
	lower := strings.ToLower(line)
	for _, clean := range []string{
		"0 warning",
		"0 warnings",
		"no warning",
		"no warnings",
		"without warning",
		"without warnings",
		"warnings: 0",
		"warning count: 0",
	} {
		if strings.Contains(lower, clean) {
			return false
		}
	}
	for _, marker := range []string{
		"warning:",
		"warn:",
		"warning ",
		" warnings",
		"deprecationwarning",
		"deprecated",
		"stderr:",
		"non-fatal",
		"flaky",
		"skipped",
		"pending",
		"todo",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func truncateResidualRisk(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= 220 {
		return value
	}
	return value[:217] + "..."
}

func normalizeCompletedCheckpointState(checkpoint *WorkflowCheckpoint) bool {
	if checkpoint == nil || checkpoint.Status != "complete" {
		return false
	}
	changed := false
	if strings.TrimSpace(checkpoint.NextAction) != completeWorkflowNextAction {
		checkpoint.NextAction = completeWorkflowNextAction
		changed = true
	}
	if strings.TrimSpace(checkpoint.NextVerification) != "" {
		checkpoint.NextVerification = ""
		changed = true
	}
	if strings.TrimSpace(checkpoint.CurrentPhase) != "" {
		checkpoint.CurrentPhase = ""
		changed = true
	}
	if strings.TrimSpace(checkpoint.NextPhase) != "" {
		checkpoint.NextPhase = ""
		changed = true
	}
	return changed
}

func normalizeWorkflowVerificationEvidence(checkpoint *WorkflowCheckpoint, extraEvidence ...string) bool {
	if checkpoint == nil || checkpoint.Status != "complete" || len(checkpoint.VerificationResults) > 0 {
		return false
	}
	var results []string
	for _, entry := range compactStrings(extraEvidence) {
		if !verificationEntryLooksLikeResult(entry) {
			continue
		}
		results = append(results, "recorded verification evidence: "+entry)
	}
	if len(results) == 0 {
		return false
	}
	checkpoint.VerificationResults = Unique(results)
	checkpoint.ResidualRisks = Unique(append(checkpoint.ResidualRisks, residualRisksFromVerificationResults(results)...))
	return true
}

func verificationEntryLooksLikeResult(entry string) bool {
	lower := strings.ToLower(strings.Join(strings.Fields(entry), " "))
	if lower == "" {
		return false
	}
	for _, phrase := range []string{
		"exit ",
		"exit:",
		"exit code",
		"status:",
		"ok:",
		"not run",
	} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	resultWords := map[string]bool{
		"pass":      true,
		"passed":    true,
		"fail":      true,
		"failed":    true,
		"error":     true,
		"warning":   true,
		"blocked":   true,
		"skipped":   true,
		"success":   true,
		"complete":  true,
		"completed": true,
		"verified":  true,
	}
	for _, word := range strings.FieldsFunc(lower, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	}) {
		if resultWords[word] {
			return true
		}
	}
	return false
}
