package aitools

func classificationAgents(index RepoIndex, classification RepoClassification) []AgentProfile {
	var agents []AgentProfile
	if len(filesByCategory(index, "route")) > 0 || len(filesByCategory(index, "contract")) > 0 {
		agents = append(agents,
			AgentProfile{
				Name:        "api-route-contract-engineer",
				Description: "Works on detected route handlers, request/response contracts, validation boundaries, and response shapes",
				FocusFiles:  mergeFileLists(filesByCategory(index, "entrypoint"), filesByCategory(index, "module"), filesByCategory(index, "route"), filesByCategory(index, "contract")),
				Workflow: []string{
					"Start from detected entrypoints/modules, then trace routes and contract files before changing request or response behavior.",
					"Keep validation, response shape, status codes, and route tests aligned.",
					"Update focused tests when route behavior changes.",
				},
				Verification: taskScopedVerificationGuidance(),
			},
		)
	}
	if len(configBoundaryFiles(index)) > 0 {
		agents = append(agents,
			AgentProfile{
				Name:        "config-bootstrap-boundary-reviewer",
				Description: "Reviews bootstrap, runtime configuration, middleware wiring, environment schema, and deployment config",
				FocusFiles:  configBoundaryFiles(index),
				Workflow: []string{
					"Review runtime bootstrap, middleware, validation, API docs, and env validation together when present.",
					"Check that config changes are mirrored in .env.example, tests, Docker, Helm, and CI when relevant.",
					"Do not expose secret values in generated metadata or logs.",
				},
				Verification: taskScopedVerificationGuidance(),
			},
		)
	}
	if len(filesByCategory(index, "test")) > 0 {
		agents = append(agents,
			AgentProfile{
				Name:        "repo-test-harness-reviewer",
				Description: "Maintains repository tests, mocks, fixtures, and reliable verification commands",
				FocusFiles:  filesByCategory(index, "test"),
				Workflow: []string{
					"Pick the smallest related test target before broad test runs.",
					"Keep mocks and integration setup aligned with module wiring and external service contracts.",
					"Record exact blockers when tests need unavailable environment or services.",
				},
				Verification: taskScopedVerificationGuidance(),
			},
		)
	}
	if len(filesByCategory(index, "persistence")) > 0 {
		focusFiles := filesByCategory(index, "persistence")
		agents = append(agents, AgentProfile{
			Name:        "persistence-boundary-reviewer",
			Description: "Reviews persistence schema, migrations, repository calls, and persistence-side contract risk",
			FocusFiles:  focusFiles,
			Workflow: []string{
				"Start with persistence schemas, migrations, repository calls, and storage adapters before touching persistence behavior.",
				"Check data contract compatibility and migration side effects.",
				"Avoid running destructive database operations without explicit user approval.",
			},
			Verification: taskScopedVerificationGuidance(),
		})
	}
	if len(classification.ExternalSystems) > 0 {
		focusFiles := externalSystemFiles(classification)
		agents = append(agents, AgentProfile{
			Name:        "external-integration-contract-reviewer",
			Description: "Reviews external integration request construction, authentication, DTO mapping, error handling, and test mocks",
			FocusFiles:  focusFiles,
			Workflow: []string{
				"Trace request URL construction, auth headers, DTO mapping, error handling, and test mocks for each external system.",
				"Keep controller/service tests and e2e mocks aligned with external contract changes.",
				"Never print tokens, API keys, private keys, or env secret values.",
			},
			Verification: taskScopedVerificationGuidance(),
		})
	}
	for _, domain := range classification.Domains {
		if len(domain.Files) == 0 {
			continue
		}
		agents = append(agents, AgentProfile{
			Name:        domainAgentName(domain.Name),
			Description: domain.Description,
			FocusFiles:  domain.Files,
			Workflow: []string{
				"Read all files in this domain before changing behavior.",
				"Trace controller, service, DTO, module, mocks, and e2e coverage for the domain.",
				"Escalate cross-domain contract changes to external-integration-contract-reviewer or api-route-contract-engineer.",
			},
			Verification: taskScopedVerificationGuidance(),
		})
	}
	return dedupeAgents(agents)
}

func classificationSkills(index RepoIndex, classification RepoClassification) []SkillProfile {
	var skills []SkillProfile
	if len(configBoundaryFiles(index)) > 0 {
		skills = append(skills,
			SkillProfile{
				Name:        "bootstrap-config-surface",
				Description: "Work on bootstrap wiring, validation, API docs, middleware, and runtime config",
				FocusFiles:  configBoundaryFiles(index),
				Workflow: []string{
					"Start with detected entrypoint, module, and configuration files.",
					"Check env schema, routing prefix/versioning, validation, API docs, and middleware together when present.",
					"Reflect config changes into tests and deployment files when they affect runtime.",
				},
				Verification: taskScopedVerificationGuidance(),
			},
		)
	}
	if len(filesByCategory(index, "test")) > 0 {
		skills = append(skills,
			SkillProfile{
				Name:        "repo-test-harness",
				Description: "Select and maintain repository tests, mocks, and setup files",
				FocusFiles:  filesByCategory(index, "test"),
				Workflow: []string{
					"Prefer focused repository-native test targets when possible.",
					"Use detected test configuration and package scripts instead of hardcoded runner names.",
					"Treat e2e environment blockers as explicit risks instead of silently skipping them.",
				},
				Verification: taskScopedVerificationGuidance(),
			},
		)
	}
	if len(filesByCategory(index, "persistence")) > 0 {
		focusFiles := filesByCategory(index, "persistence")
		skills = append(skills, SkillProfile{
			Name:        "persistence-surface",
			Description: "Work on persistence schema, repository, migration, and storage contract surfaces",
			FocusFiles:  focusFiles,
			Workflow: []string{
				"Trace repository calls, schema, migrations, and persistence contracts together.",
				"Keep DTO validation, persistence writes, and tests aligned.",
				"Do not run migrations against a real database without explicit approval.",
			},
			Verification: taskScopedVerificationGuidance(),
		})
	}
	for _, domain := range classification.Domains {
		files := filesForDomains(classification, domain.Name)
		if len(files) == 0 {
			continue
		}
		name := sanitizeID(domain.Name)
		if name == "" {
			continue
		}
		skills = append(skills, SkillProfile{
			Name:        name + "-domain-surface",
			Description: "Work on the " + domain.Name + " domain surface, related contracts, and verification",
			FocusFiles:  files,
			Workflow: []string{
				"Read the full domain surface before editing.",
				"Trace controller, service, DTO, mocks, and e2e coverage.",
				"Keep external request/response contracts and error handling consistent.",
			},
			Verification: taskScopedVerificationGuidance(),
		})
	}
	return dedupeSkills(skills)
}

func validateClassification(index RepoIndex, classification RepoClassification) RepoClassification {
	exists := fileSet(index)
	validateFiles := func(files []string) []string {
		var out []string
		for _, file := range files {
			if exists[file] {
				out = append(out, file)
			} else {
				classification.Warnings = append(classification.Warnings, "dropped non-existent classified file: "+file)
			}
		}
		return Unique(out)
	}
	for i := range classification.Entrypoints {
		classification.Entrypoints[i].Files = validateFiles(classification.Entrypoints[i].Files)
	}
	for i := range classification.Configs {
		classification.Configs[i].Files = validateFiles(classification.Configs[i].Files)
	}
	for i := range classification.Persistence {
		classification.Persistence[i].Files = validateFiles(classification.Persistence[i].Files)
	}
	for i := range classification.Tests {
		classification.Tests[i].Files = validateFiles(classification.Tests[i].Files)
	}
	for i := range classification.ExternalSystems {
		classification.ExternalSystems[i].Files = validateFiles(classification.ExternalSystems[i].Files)
	}
	for i := range classification.Domains {
		classification.Domains[i].Files = validateFiles(classification.Domains[i].Files)
	}
	for i := range classification.Agents {
		classification.Agents[i].FocusFiles = validateFiles(classification.Agents[i].FocusFiles)
	}
	for i := range classification.Skills {
		classification.Skills[i].FocusFiles = validateFiles(classification.Skills[i].FocusFiles)
	}
	if len(classification.Warnings) > 0 {
		classification.ValidationStatus = "warning"
	}
	return classification
}
