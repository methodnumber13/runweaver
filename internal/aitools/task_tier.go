package aitools

import "strings"

// ClassifyTaskTier maps a user task to the smallest useful orchestration tier.
func ClassifyTaskTier(task string) TaskTierResult {
	tokens := tokenizeSelectionText(task)
	score := 0
	var rationale []string
	add := func(points int, reason string) {
		score += points
		rationale = append(rationale, reason)
	}
	if len(tokens) <= 5 {
		add(-2, "short task")
	}
	if len(tokens) >= 14 {
		add(2, "multi-token task")
	}
	if len(tokens) >= 28 {
		add(3, "large task description")
	}
	for _, token := range []string{"typo", "rename", "comment", "readme", "format", "lint"} {
		if tokens[token] {
			add(-2, "small-change signal: "+token)
		}
	}
	for _, token := range []string{"fix", "bug", "test", "spec", "add", "implement", "endpoint", "command"} {
		if tokens[token] {
			add(2, "implementation signal: "+token)
		}
	}
	for _, token := range []string{"architecture", "refactor", "migration", "migrate", "provider", "runtime", "integration", "workflow", "security", "e2e", "cross-runtime"} {
		if tokens[token] {
			add(4, "broad-change signal: "+token)
		}
	}
	if strings.Contains(strings.ToLower(task), " and ") || strings.Contains(task, ",") {
		add(1, "compound task text")
	}
	tier := "small"
	cap := 2
	phaseStrategy := "single owner plus focused verification"
	verification := "focused"
	switch {
	case score <= -2:
		tier = "trivial"
		cap = 1
		phaseStrategy = "single participant; skip broad swarm unless evidence expands scope"
		verification = "smallest relevant check"
	case score >= 8:
		tier = "large"
		cap = 4
		phaseStrategy = "full workflow phases with review and verification participants"
		verification = "focused checks plus broad regression gate"
	case score >= 4:
		tier = "medium"
		cap = 3
		phaseStrategy = "workflow phases with domain owner and reviewer"
		verification = "focused regression plus build/type/lint where relevant"
	}
	if len(rationale) == 0 {
		rationale = append(rationale, "default small task")
	}
	return TaskTierResult{
		Tier:           tier,
		Score:          score,
		ParticipantCap: cap,
		PhaseStrategy:  phaseStrategy,
		Verification:   verification,
		Rationale:      Unique(rationale),
	}
}

func applyTaskTierCap(cap int, tier string) int {
	tier = strings.ToLower(strings.TrimSpace(tier))
	if cap <= 0 {
		cap = defaultParticipantCap
	}
	switch tier {
	case "trivial":
		return 1
	case "small":
		if cap > 2 {
			return 2
		}
	case "medium":
		if cap > 3 {
			return 3
		}
	}
	return cap
}
