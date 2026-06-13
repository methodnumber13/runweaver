package aitools

import (
	"path/filepath"
	"strings"
)

func mergeAgents(base []AgentProfile, extra ...AgentProfile) []AgentProfile {
	seen := map[string]bool{}
	var out []AgentProfile
	for _, agent := range append(base, extra...) {
		if agent.Name == "" || seen[agent.Name] {
			continue
		}
		seen[agent.Name] = true
		out = append(out, agent)
	}
	return out
}

func mergeSkills(base []SkillProfile, extra ...SkillProfile) []SkillProfile {
	seen := map[string]bool{}
	var out []SkillProfile
	for _, skill := range append(base, extra...) {
		if skill.Name == "" || seen[skill.Name] {
			continue
		}
		seen[skill.Name] = true
		out = append(out, skill)
	}
	return out
}

func enrichAgentProfiles(index RepoIndex, agents []AgentProfile) []AgentProfile {
	var out []AgentProfile
	for _, agent := range agents {
		switch agent.Name {
		case "api-route-engineer":
			agent.FocusFiles = mergeFileLists(filesByCategory(index, "entrypoint"), filesByCategory(index, "module"), filesByCategory(index, "route"), filesByCategory(index, "contract"))
			agent.Workflow = []string{
				"Start at NestJS bootstrap/module wiring, then trace controller decorators and DTO contracts.",
				"Do not treat spec files as route sources; use tests only for verification or requested updates.",
				"Keep request validation, response shape, status codes, and route tests aligned.",
			}
			agent.Verification = index.Tools.RecommendedCommands
		case "api-security-boundary-reviewer":
			agent.FocusFiles = mergeFileLists(domainFiles(index.Classification, "auth"), configBoundaryFiles(index), filesByCategory(index, "route"), filesByCategory(index, "contract"))
			agent.Workflow = []string{
				"Review guards, decorators, token/API-key handling, env validation, request validation, and unsafe IO together.",
				"Check external auth/service calls for secret handling and error behavior.",
				"Never print or persist secret values from env/config/auth files.",
			}
			agent.Verification = index.Tools.RecommendedCommands
		case "api-test-harness-reviewer", "repo-test-quality-reviewer":
			agent.FocusFiles = mergeFileLists(filesByCategory(index, "test"), filterFiles(index, "test/jest-"))
			agent.Workflow = []string{
				"Select the smallest reliable Jest unit/e2e command for the changed surface.",
				"Keep mocks, fixtures, and create-test-app helpers aligned with module wiring.",
				"Record exact blockers for e2e tests that require unavailable env or services.",
			}
			agent.Verification = index.Tools.RecommendedCommands
		case "persistence-boundary-reviewer":
			agent.FocusFiles = filesByCategory(index, "persistence")
			agent.Workflow = []string{
				"Review Prisma schema, migrations, PrismaService, and repository calls together.",
				"Check data contract and migration side effects before implementation.",
				"Do not run destructive database commands without explicit approval.",
			}
			agent.Verification = index.Tools.RecommendedCommands
		case "validation-contract-reviewer", "repo-contract-reviewer":
			agent.FocusFiles = mergeFileLists(filesByCategory(index, "contract"), filesByCategory(index, "route"), filesByCategory(index, "module"))
			agent.Workflow = []string{
				"Trace DTOs, params, validation decorators, route handlers, and response types together.",
				"Check compatibility of public request/response contracts.",
				"Use tests and e2e cases as contract evidence, not as source surfaces.",
			}
			agent.Verification = index.Tools.RecommendedCommands
		case "repo-surface-engineer":
			agent.FocusFiles = mergeFileLists(filesByCategory(index, "entrypoint"), filesByCategory(index, "module"), filesByCategory(index, "route"), filesByCategory(index, "service"), filesByCategory(index, "contract"), filesByCategory(index, "persistence"))
			agent.Workflow = []string{
				"Use the generated domain agents when the task maps to a specific BFF domain.",
				"Keep edits scoped to the current domain and its contracts.",
				"Refresh metadata after moving routes, services, tests, configs, or schema files.",
			}
			agent.Verification = index.Tools.RecommendedCommands
		}
		out = append(out, agent)
	}
	return out
}

func joinHuman(items []string) string {
	items = Unique(items)
	if len(items) == 0 {
		return "none detected"
	}
	return filepath.ToSlash(strings.Join(items, ", "))
}
