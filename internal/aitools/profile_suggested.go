package aitools

func runtimeFor(index SurfaceIndex) string {
	switch index.Stack.Kind {
	case "frontend-react":
		return "React/TypeScript/JavaScript, " + index.Stack.PackageManager
	case "node-api":
		return "Node API, " + index.Stack.PackageManager
	case "go-service":
		return "Go"
	case "jvm-service":
		return "JVM/Gradle/Maven"
	case "python-service":
		return "Python"
	default:
		return "Generic repository"
	}
}

func suggestedAgents(index SurfaceIndex) []AgentProfile {
	switch index.Stack.Kind {
	case "frontend-react":
		return []AgentProfile{
			{Name: "frontend-surface-engineer", Description: "Works on React components, pages, routing, and user-facing state"},
			{Name: "frontend-api-contract-reviewer", Description: "Checks frontend API clients against backend contracts"},
			{Name: "frontend-test-quality-reviewer", Description: "Finds and updates relevant frontend tests, fixtures, and mocks"},
		}
	case "node-api":
		return []AgentProfile{
			{Name: "api-route-engineer", Description: "Works on Node API routes, middleware, handlers, and response contracts"},
			{Name: "api-security-boundary-reviewer", Description: "Reviews auth, validation, secrets, and unsafe IO in API code"},
			{Name: "api-test-harness-reviewer", Description: "Maintains route tests, mocks, and integration harnesses"},
		}
	case "go-service":
		return []AgentProfile{
			{Name: "go-service-engineer", Description: "Works on Go command entrypoints, packages, and application logic"},
			{Name: "go-boundary-reviewer", Description: "Reviews Go package boundaries, adapters, interfaces, and error handling"},
			{Name: "go-test-quality-reviewer", Description: "Maintains Go unit tests and package-level verification"},
		}
	case "jvm-service":
		return []AgentProfile{
			{Name: "jvm-service-engineer", Description: "Works on Kotlin/Java controllers, services, repositories, and config"},
			{Name: "jvm-contract-reviewer", Description: "Reviews JVM API contracts, DTOs, validation, and security boundaries"},
			{Name: "jvm-test-quality-reviewer", Description: "Maintains Gradle/Maven tests and fixtures"},
		}
	default:
		return []AgentProfile{
			{Name: "repo-surface-engineer", Description: "Works on the repository's main source surfaces"},
			{Name: "repo-contract-reviewer", Description: "Reviews public contracts, configuration, and verification gates"},
			{Name: "repo-test-quality-reviewer", Description: "Maintains available tests and manual verification guidance"},
		}
	}
}

func suggestedSkills(index SurfaceIndex) []SkillProfile {
	var skills []SkillProfile
	if len(index.Routes) > 0 {
		skills = append(skills, SkillProfile{
			Name:        "api-route-surface",
			Description: "Work on detected API route/controller surfaces",
			FocusFiles:  Limit(index.Routes, 20),
			Workflow: []string{
				"Trace route registration before editing handlers.",
				"Keep request validation, response shape, status codes, and tests aligned.",
				"Update runtime metadata after route moves or additions.",
			},
			Verification: index.BuildCommands,
		})
	}
	if len(index.Pages) > 0 {
		skills = append(skills, SkillProfile{
			Name:        "ui-page-surface",
			Description: "Work on detected page/component/container surfaces",
			FocusFiles:  Limit(index.Pages, 20),
			Workflow: []string{
				"Trace component ownership, state flow, API calls, and visible states before editing.",
				"Check loading, error, empty, disabled, and success states.",
				"Update tests and metadata after page/component moves.",
			},
			Verification: index.BuildCommands,
		})
	}
	skills = append(skills, SkillProfile{
		Name:        "repo-quality-gates",
		Description: "Select correct verification commands for this repository",
		FocusFiles:  index.ConfigFiles,
		Workflow: []string{
			"Use detected package/build commands only.",
			"Run focused checks first and broaden only for shared contracts.",
			"Record any command that cannot run with the exact blocker.",
		},
		Verification: index.BuildCommands,
	})
	return skills
}
