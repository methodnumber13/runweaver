package aitools

import (
	"strings"
)

func normalizeDomainFirstClassification(index RepoIndex, classification RepoClassification) RepoClassification {
	if len(classification.Domains) == 0 {
		return classification
	}
	byName := map[string]AgentProfile{}
	for _, agent := range classification.Agents {
		name := sanitizeID(agent.Name)
		if name == "" {
			continue
		}
		byName[name] = agent
	}

	verification := classification.Verification
	if len(verification) == 0 {
		verification = index.Tools.RecommendedCommands
	}

	var primary []AgentProfile
	primaryNames := map[string]bool{}
	for _, domain := range classification.Domains {
		name := domainFirstAgentName(domain)
		if name == "" || len(domain.Files) == 0 {
			continue
		}
		agent := AgentProfile{
			Name:         name,
			Description:  domainAgentDescription(domain),
			FocusFiles:   Limit(Unique(domain.Files), 40),
			Workflow:     domainAgentWorkflow(domain.Name),
			Verification: verification,
		}
		if existing, ok := byName[name]; ok {
			if existing.Description != "" {
				agent.Description = existing.Description
			}
			agent.FocusFiles = mergeFileLists(agent.FocusFiles, existing.FocusFiles)
			agent.Workflow = Unique(append(agent.Workflow, existing.Workflow...))
			agent.Verification = Unique(append(agent.Verification, existing.Verification...))
		}
		primary = append(primary, agent)
		primaryNames[name] = true
	}
	if len(primary) == 0 {
		return classification
	}

	var secondary []AgentProfile
	for _, agent := range classification.Agents {
		name := sanitizeID(agent.Name)
		if name == "" || primaryNames[name] {
			continue
		}
		if duplicatesPrimaryDomainAgent(name, classification.Domains) {
			continue
		}
		secondary = append(secondary, agent)
	}
	classification.Agents = dedupeAgents(append(primary, secondary...))
	return classification
}

func duplicatesPrimaryDomainAgent(agentName string, domains []DomainClassification) bool {
	for _, domain := range domains {
		name := sanitizeID(domain.Name)
		if name == "" {
			continue
		}
		for _, suffix := range []string{"-domain-agent", "-integration-agent", "-persistence-agent", "-orchestration-agent", "-security-agent"} {
			if agentName == name+suffix {
				return true
			}
		}
	}
	return false
}

func domainFirstAgentName(domain DomainClassification) string {
	name := sanitizeID(domain.Name)
	if name == "" {
		return ""
	}
	if isBFFDomain(name) {
		return domainAgentName(name)
	}
	if name == "prisma" {
		return "prisma-persistence-agent"
	}
	switch strings.TrimSpace(domain.Kind) {
	case "external-integration", "integration":
		return name + "-integration-agent"
	case "orchestration":
		return name + "-orchestration-agent"
	case "persistence", "database":
		return name + "-persistence-agent"
	case "auth", "security":
		return name + "-security-agent"
	default:
		return name + "-domain-agent"
	}
}

func domainAgentDescription(domain DomainClassification) string {
	description := strings.TrimSpace(domain.Description)
	if description == "" {
		description = domainDescription(domain.Name)
	}
	return description
}

func domainAgentWorkflow(domainName string) []string {
	name := sanitizeID(domainName)
	if name == "" {
		name = "this"
	}
	return []string{
		"Read the full " + name + " domain surface and related tests before changing behavior.",
		"Trace controllers/routes, services, DTO/contracts, modules, mocks, and external calls for this domain.",
		"Escalate edits outside this domain to the matching domain agent or workflow reviewer before implementation.",
	}
}
