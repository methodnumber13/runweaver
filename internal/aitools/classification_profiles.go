package aitools

func classificationAgents(index RepoIndex, classification RepoClassification) []AgentProfile {
	var agents []AgentProfile
	if hasPackage(index, "@nestjs/core") {
		agents = append(agents,
			AgentProfile{
				Name:        "nestjs-route-contract-engineer",
				Description: "Works on NestJS controllers, route decorators, DTO contracts, global prefix/versioning, and response shapes",
				FocusFiles:  mergeFileLists(filesByCategory(index, "entrypoint"), filesByCategory(index, "module"), filesByCategory(index, "route"), filesByCategory(index, "contract")),
				Workflow: []string{
					"Start at src/main.ts and src/app.module.ts to confirm global prefix, versioning, pipes, guards, and module wiring.",
					"Trace controller decorators and DTO validation before changing request or response contracts.",
					"Update focused unit/e2e tests when route behavior changes.",
				},
				Verification: index.Tools.RecommendedCommands,
			},
			AgentProfile{
				Name:        "nestjs-config-boundary-reviewer",
				Description: "Reviews NestJS bootstrap, ConfigModule/Joi env schema, Swagger setup, middleware, and deployment config",
				FocusFiles:  configBoundaryFiles(index),
				Workflow: []string{
					"Review runtime bootstrap, global middleware, validation pipe, Swagger, and env validation together.",
					"Check that config changes are mirrored in .env.example, tests, Docker, Helm, and CI when relevant.",
					"Do not expose secret values in generated metadata or logs.",
				},
				Verification: index.Tools.RecommendedCommands,
			},
			AgentProfile{
				Name:        "nestjs-test-harness-reviewer",
				Description: "Maintains NestJS Jest unit/e2e tests, mocks, fixtures, and reliable verification commands",
				FocusFiles:  mergeFileLists(filesByCategory(index, "test"), filterFiles(index, "test/jest-")),
				Workflow: []string{
					"Pick the smallest related Jest target before broad test runs.",
					"Keep mocks and e2e setup aligned with module wiring and external service contracts.",
					"Record exact blockers when e2e tests need unavailable environment or services.",
				},
				Verification: index.Tools.RecommendedCommands,
			},
		)
	}
	if hasPackage(index, "prisma") || hasPackage(index, "@prisma/client") {
		agents = append(agents, AgentProfile{
			Name:        "prisma-persistence-reviewer",
			Description: "Reviews Prisma schema, migrations, PrismaService, repositories, and persistence-side contract risk",
			FocusFiles:  filesByCategory(index, "persistence"),
			Workflow: []string{
				"Start with prisma/schema.prisma, migrations, PrismaService, and repositories before touching persistence behavior.",
				"Check data contract compatibility and migration side effects.",
				"Avoid running destructive database operations without explicit user approval.",
			},
			Verification: index.Tools.RecommendedCommands,
		})
	}
	if len(classification.ExternalSystems) > 0 {
		agents = append(agents, AgentProfile{
			Name:        "external-integration-contract-reviewer",
			Description: "Reviews API integrations with identity, source control, orchestration, catalog, deployment, messaging, observability, object storage, and platform services",
			FocusFiles:  externalSystemFiles(classification),
			Workflow: []string{
				"Trace request URL construction, auth headers, DTO mapping, error handling, and test mocks for each external system.",
				"Keep controller/service tests and e2e mocks aligned with external contract changes.",
				"Never print tokens, API keys, private keys, or env secret values.",
			},
			Verification: index.Tools.RecommendedCommands,
		})
	}
	for _, domain := range classification.Domains {
		if !isBFFDomain(domain.Name) {
			continue
		}
		agents = append(agents, AgentProfile{
			Name:        domainAgentName(domain.Name),
			Description: domain.Description,
			FocusFiles:  domain.Files,
			Workflow: []string{
				"Read all files in this domain before changing behavior.",
				"Trace controller, service, DTO, module, mocks, and e2e coverage for the domain.",
				"Escalate cross-domain contract changes to external-integration-contract-reviewer or nestjs-route-contract-engineer.",
			},
			Verification: index.Tools.RecommendedCommands,
		})
	}
	return dedupeAgents(agents)
}

func classificationSkills(index RepoIndex, classification RepoClassification) []SkillProfile {
	var skills []SkillProfile
	if hasPackage(index, "@nestjs/core") {
		skills = append(skills,
			SkillProfile{
				Name:        "nestjs-bootstrap-config",
				Description: "Work on NestJS bootstrap, app module wiring, global validation, Swagger, and runtime config",
				FocusFiles:  configBoundaryFiles(index),
				Workflow: []string{
					"Start with src/main.ts and src/app.module.ts.",
					"Check ConfigModule/Joi schema, global prefix, versioning, ValidationPipe, Swagger, and middleware together.",
					"Reflect config changes into tests and deployment files when they affect runtime.",
				},
				Verification: index.Tools.RecommendedCommands,
			},
			SkillProfile{
				Name:        "nestjs-test-harness",
				Description: "Select and maintain NestJS Jest unit/e2e tests, mocks, and setup files",
				FocusFiles:  mergeFileLists(filesByCategory(index, "test"), filterFiles(index, "test/jest-")),
				Workflow: []string{
					"Prefer focused Jest targets when possible.",
					"Use test/jest-unit.json for unit tests and test/jest-e2e.json for e2e tests.",
					"Treat e2e environment blockers as explicit risks instead of silently skipping them.",
				},
				Verification: index.Tools.RecommendedCommands,
			},
		)
	}
	if hasPackage(index, "prisma") || hasPackage(index, "@prisma/client") {
		skills = append(skills, SkillProfile{
			Name:        "templates-prisma-surface",
			Description: "Work on template orchestration and Prisma persistence surfaces",
			FocusFiles:  mergeFileLists(domainFiles(classification, "templates"), filesByCategory(index, "persistence")),
			Workflow: []string{
				"Trace template service orchestration, repository calls, Prisma schema, and migrations together.",
				"Keep DTO validation, persistence writes, and tests aligned.",
				"Do not run migrations against a real database without explicit approval.",
			},
			Verification: index.Tools.RecommendedCommands,
		})
	}
	for _, spec := range []struct {
		name        string
		description string
		domains     []string
	}{
		{"identity-auth-surface", "Work on auth guards, identity introspection/userinfo, x-api-key behavior, and auth tests", []string{"auth"}},
		{"scm-integration-surface", "Work on source control adapter controllers, services, DTOs, and tests", []string{"scm"}},
		{"kubernetes-integration-surface", "Work on Kubernetes cluster controllers, service contracts, DTOs, and tests", []string{"kubernetes"}},
		{"object-storage-surface", "Work on object storage access flows, private-key handling, DTOs, and tests", []string{"object-storage"}},
		{"platform-catalog-deployment-surface", "Work on platform, catalog, and deployment integration contracts and orchestration", []string{"devops", "catalog-service", "deployment-service"}},
	} {
		files := filesForDomains(classification, spec.domains...)
		if len(files) == 0 {
			continue
		}
		skills = append(skills, SkillProfile{
			Name:        spec.name,
			Description: spec.description,
			FocusFiles:  files,
			Workflow: []string{
				"Read the full domain surface before editing.",
				"Trace controller, service, DTO, mocks, and e2e coverage.",
				"Keep external request/response contracts and error handling consistent.",
			},
			Verification: index.Tools.RecommendedCommands,
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
