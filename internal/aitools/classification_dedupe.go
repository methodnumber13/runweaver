package aitools

func dedupeAgents(items []AgentProfile) []AgentProfile {
	seen := map[string]bool{}
	var out []AgentProfile
	for _, item := range items {
		if item.Name == "" || seen[item.Name] {
			continue
		}
		seen[item.Name] = true
		out = append(out, item)
	}
	return out
}

func dedupeSkills(items []SkillProfile) []SkillProfile {
	seen := map[string]bool{}
	var out []SkillProfile
	for _, item := range items {
		if item.Name == "" || seen[item.Name] {
			continue
		}
		seen[item.Name] = true
		out = append(out, item)
	}
	return out
}

// LimitDomains returns at most n domain classifications while preserving order.
func LimitDomains(items []DomainClassification, n int) []DomainClassification {
	if len(items) <= n {
		return items
	}
	return items[:n]
}
