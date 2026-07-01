package aitools

func runtimeFor(index SurfaceIndex) string {
	switch index.Stack.Kind {
	case "multi-surface":
		return "Multi-surface repository, " + index.Stack.PackageManager
	case "frontend-app":
		return "Frontend application, " + index.Stack.PackageManager
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
	case "multi-surface":
		return []AgentProfile{
			{Name: "repo-surface-engineer", Description: "Works on task-selected source surfaces without crossing forbidden boundaries"},
			{Name: "ui-surface-reviewer", Description: "Reviews user-facing pages, components, state, and visible behavior when selected by task scope"},
			{Name: "api-boundary-reviewer", Description: "Reviews route/API contracts, validation, and persistence boundaries when selected by task scope"},
		}
	case "frontend-app":
		return []AgentProfile{
			{Name: "frontend-surface-engineer", Description: "Works on frontend components, pages, routing, and user-facing state"},
			{Name: "frontend-api-contract-reviewer", Description: "Checks frontend API clients against server/API contracts"},
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
				"Record metadata drift after route moves or additions; run full metadata refresh only when explicitly requested.",
			},
			Verification: taskScopedVerificationGuidance(),
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
				"Update tests and record metadata drift after page/component moves; run full metadata refresh only when explicitly requested.",
			},
			Verification: verificationForSurface(index.BuildCommands, surfaceScope{Frontend: true}, index.Stack.Kind),
		})
	}
	skills = append(skills, SkillProfile{
		Name:        "repo-quality-gates",
		Description: "Select correct verification commands for this repository",
		FocusFiles:  index.ConfigFiles,
		Workflow: []string{
			"Use detected package/build commands only.",
			"Use runweaver start/context query to select task-scoped checks first and broaden only for shared contracts.",
			"Record any command that cannot run with the exact blocker.",
		},
		Verification: taskScopedVerificationGuidance(),
	})
	return skills
}
