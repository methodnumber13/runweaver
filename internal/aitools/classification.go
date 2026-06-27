package aitools

// ClassifyRepo derives semantic domains, agents, skills, and verification hints.
func ClassifyRepo(index RepoIndex) RepoClassification {
	classification := RepoClassification{
		Source:           "deterministic-semantic-fallback",
		ModelReady:       true,
		ValidationStatus: "valid",
		Summary:          classificationSummary(index),
		Verification:     index.Tools.RecommendedCommands,
	}
	classification.Entrypoints = surfacesByCategory(index, "entrypoint", "application-entrypoint")
	classification.Configs = surfacesByCategory(index, "config", "configuration")
	classification.Persistence = surfacesByCategory(index, "persistence", "persistence")
	classification.Tests = surfacesByCategory(index, "test", "test")
	classification.Domains = classifyDomains(index)
	classification.ExternalSystems = classifyExternalSystems(index, classification.Domains)
	classification.Agents = classificationAgents(index, classification)
	classification.Skills = classificationSkills(index, classification)
	classification = validateClassification(index, classification)
	return classification
}

func classificationSummary(index RepoIndex) string {
	switch index.Surface.Stack.Kind {
	case "node-api":
		if hasPackage(index, "@nestjs/core") {
			return "NestJS BFF/API service classified from deterministic package, file, decorator, and domain evidence."
		}
		return "Node API service classified from deterministic package, file, and domain evidence."
	case "frontend-react":
		return "React frontend classified from deterministic package, file, and route/page evidence."
	default:
		return "Repository classified from deterministic package, file, and symbol evidence."
	}
}
