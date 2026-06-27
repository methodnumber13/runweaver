package aitools

import (
	"strings"
)

func workflowPhaseIDs(workflow WorkflowSpec) []string {
	ids := make([]string, 0, len(workflow.Phases))
	for _, phase := range workflow.Phases {
		ids = append(ids, phase.ID)
	}
	return ids
}

func expectedNextPhase(phaseIDs, completed []string) string {
	done := map[string]bool{}
	for _, phase := range completed {
		done[phase] = true
	}
	for _, phase := range phaseIDs {
		if !done[phase] {
			return phase
		}
	}
	return ""
}

func workflowParticipantCap(workflow WorkflowSpec) int {
	cap := workflow.MaxParticipants
	if cap <= 0 {
		cap = workflow.MaxConcurrent
	}
	if cap <= 0 {
		cap = 4
	}
	return cap
}

func workflowHasWritePhase(workflow WorkflowSpec) bool {
	for _, phase := range workflow.Phases {
		if strings.EqualFold(strings.TrimSpace(phase.WriteMode), "write") {
			return true
		}
	}
	return false
}

func addWorkflowVerificationCheck(result *WorkflowVerificationResult, check WorkflowVerificationCheck) {
	result.Checks = append(result.Checks, check)
}

func finalizeWorkflowVerification(result WorkflowVerificationResult) WorkflowVerificationResult {
	status := "ok"
	recommendations := []string{}
	for _, check := range result.Checks {
		switch check.Status {
		case "error":
			status = "error"
		case "warning":
			if status != "error" {
				status = "warning"
			}
		}
		recommendations = append(recommendations, check.NextActions...)
	}
	result.Status = status
	result.Ready = status == "ok"
	result.Recommendations = Unique(recommendations)
	return result
}
