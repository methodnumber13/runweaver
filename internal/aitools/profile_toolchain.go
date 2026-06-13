package aitools

func runtimeFromToolchain(index RepoIndex) string {
	parts := []string{}
	parts = append(parts, index.Tools.Languages...)
	parts = append(parts, index.Tools.Frameworks...)
	parts = append(parts, index.Tools.PackageManagers...)
	parts = Unique(parts)
	if len(parts) == 0 {
		return runtimeFor(index.Surface)
	}
	return joinHuman(parts)
}

func domainFromIndex(index RepoIndex) string {
	var categories []string
	counts := map[string]int{}
	for _, file := range index.Files {
		if file.Category != "" {
			counts[file.Category]++
		}
	}
	for _, category := range []string{"entrypoint", "module", "route", "contract", "ui", "service", "persistence", "test", "config", "source"} {
		if counts[category] > 0 {
			categories = append(categories, category)
		}
	}
	if len(categories) == 0 {
		return domainFor(index.Surface)
	}
	return "repository " + joinHuman(categories) + " surfaces inferred from package and file index"
}

func keyFilesFromIndex(index RepoIndex) []string {
	keys := []string{}
	keys = append(keys, index.Surface.ConfigFiles...)
	keys = append(keys, index.Surface.EntryPoints...)
	keys = append(keys, index.Surface.Routes...)
	keys = append(keys, index.Surface.Pages...)
	for _, file := range index.Files {
		switch file.Category {
		case "entrypoint", "module", "route", "contract", "ui", "service", "persistence", "test", "config":
			keys = append(keys, file.Path)
		}
	}
	return Limit(Unique(keys), 100)
}

func packageAgents(index RepoIndex) []AgentProfile {
	roles := map[string]bool{}
	for _, pkg := range index.Packages {
		roles[pkg.Role] = true
	}
	var agents []AgentProfile
	if roles["state-management"] {
		agents = append(agents, AgentProfile{Name: "state-flow-reviewer", Description: "Reviews state management, async flows, cache invalidation, selectors, and UI state consistency"})
	}
	if roles["api-client"] {
		agents = append(agents, AgentProfile{Name: "api-client-contract-reviewer", Description: "Reviews API client calls, generated clients, request/response contracts, and frontend/backend compatibility"})
	}
	if roles["orm"] {
		agents = append(agents, AgentProfile{Name: "persistence-boundary-reviewer", Description: "Reviews ORM entities, migrations, query boundaries, and persistence-side contract risk"})
	}
	if roles["validation"] {
		agents = append(agents, AgentProfile{Name: "validation-contract-reviewer", Description: "Reviews schema validation, DTOs, form validation, and request/response constraints"})
	}
	if roles["rpc-framework"] {
		agents = append(agents, AgentProfile{Name: "rpc-contract-reviewer", Description: "Reviews RPC schemas, generated contracts, compatibility, and integration tests"})
	}
	return agents
}

func packageSkills(index RepoIndex) []SkillProfile {
	var skills []SkillProfile
	if len(index.Tools.TestTools) > 0 {
		skills = append(skills, SkillProfile{
			Name:        "repo-test-tooling",
			Description: "Use detected repository test tooling and cache-aware index evidence",
			FocusFiles:  focusByCategory(index, "test"),
			Workflow: []string{
				"Read .runweaver/tmp/index/repo-context.md and repo-index.compact.json before selecting tests.",
				"Use detected test tools: " + joinHuman(index.Tools.TestTools) + ".",
				"Run the smallest relevant command from recommended verification before broad checks.",
			},
			Verification: index.Tools.RecommendedCommands,
		})
	}
	if len(index.Tools.Linters) > 0 || len(index.Tools.Formatters) > 0 {
		skills = append(skills, SkillProfile{
			Name:        "repo-style-tooling",
			Description: "Respect detected lint and formatting tools",
			FocusFiles:  index.Surface.ConfigFiles,
			Workflow: []string{
				"Treat detected linters as repository rules: " + joinHuman(index.Tools.Linters) + ".",
				"Treat detected formatters as repository rules: " + joinHuman(index.Tools.Formatters) + ".",
				"Do not introduce code style that conflicts with local config files.",
			},
			Verification: index.Tools.RecommendedCommands,
		})
	}
	if hasPackageRole(index, "state-management") {
		skills = append(skills, SkillProfile{
			Name:        "state-management-surface",
			Description: "Work on detected state management surfaces",
			FocusFiles:  focusByCategory(index, "ui"),
			Workflow: []string{
				"Trace store/actions/selectors/hooks before editing UI behavior.",
				"Check loading, error, empty, optimistic, and cache invalidation states.",
				"Update tests around state transitions when behavior changes.",
			},
			Verification: index.Tools.RecommendedCommands,
		})
	}
	return skills
}

func focusByCategory(index RepoIndex, category string) []string {
	var out []string
	for _, file := range index.Files {
		if file.Category == category {
			out = append(out, file.Path)
		}
	}
	return Limit(Unique(out), 30)
}

func hasPackageRole(index RepoIndex, role string) bool {
	for _, pkg := range index.Packages {
		if pkg.Role == role {
			return true
		}
	}
	return false
}
