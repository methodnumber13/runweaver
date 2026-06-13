package aitools

import (
	"encoding/json"
	"fmt"
	"strings"
)

func repoClassifierPrompt(index RepoIndex) string {
	var b strings.Builder
	b.WriteString("# Repository Semantic Classification Prompt\n\n")
	b.WriteString("You are repo-classifier for runweaver. Classify this repository for runtime-neutral swarm agents and skills.\n")
	b.WriteString("Return exactly one minified JSON object matching RepoClassification. Do not pretty-print. Do not wrap in markdown. Do not include prose outside JSON.\n")
	b.WriteString("Use only evidence below. Do not invent files, routes, packages, environment variables, URLs, or secrets.\n")
	b.WriteString("Any files/focusFiles value must be a relative path listed in the Repository Files section. Absolute paths and ../ paths are invalid.\n\n")
	b.WriteString("## Required Compact JSON Shape\n")
	b.WriteString(`{
  "summary": "one specific sentence about what this repo is",
  "domains": [{"name":"...", "description":"specific repo responsibility", "kind":"...", "files":["..."], "confidence":"high"}],
  "agents": [{"name":"kebab-case-name", "description":"detailed task boundary", "focusFiles":["..."], "workflow":["..."], "verification":["..."]}],
  "skills": [{"name":"kebab-case-name", "description":"short reusable procedure", "focusFiles":["..."], "workflow":["..."], "verification":["..."]}],
  "verification": ["..."],
  "warnings": []
}
`)
	b.WriteString("\n\n## Classification Rules\n")
	b.WriteString("- Output must be complete but compact: max 14 domains, max 12 agents, max 8 skills, max 4 files/focusFiles per item, max 2 workflow steps and 2 verification commands per agent/skill.\n")
	b.WriteString("- Keep the whole JSON under 12000 characters. Prefer concise arrays over prose. Descriptions should be under 120 characters.\n")
	b.WriteString("- Prefer repository-specific agents and skills over generic roles.\n")
	b.WriteString("- Agents must be domain-first: create primary agents around semantic ownership domains such as auth, scm, kubernetes, templates, object-storage, checkout, billing, catalog, or pages/features. Use layer roles such as controller, service, DTO, persistence, tests, or config only as secondary reviewers/skills.\n")
	b.WriteString("- Domains should represent actual product/API/code ownership. Include controllers/routes, DTO/contracts, services, persistence, integrations, tests, configs, UI/pages as evidence inside the relevant domain instead of making the layer itself the primary owner.\n")
	b.WriteString("- If the repository has more than 14 domains, group smaller support domains, but keep agents and skills tied to real repo surfaces.\n")
	b.WriteString("- Every Mandatory Coverage File below should appear in at least one domains.files, agents.focusFiles, or skills.focusFiles entry.\n")
	b.WriteString("- Put bootstrap/config files such as src/main.ts, tsconfig*.json, env examples, Docker, Helm, and CI files in a config/bootstrap skill or agent when present.\n")
	b.WriteString("- This is AI-only classification. Do not rely on deterministic fallback filling omitted domains, agents, or skills.\n")
	b.WriteString("- Avoid duplicate semantic domains: prefer scm over scm-integration, devops over devops-integrations, prisma over prisma-persistence when they refer to the same files.\n")
	b.WriteString("- Do not create baseline metadata agents or skills such as repo-surface-indexer, repo-contract-reviewer, repo-test-quality-reviewer, metadata-refresh, or repo-onboarding; those are already provided separately.\n")
	b.WriteString("- Keep each agent focused; do not attach unrelated domain files to one role.\n")
	b.WriteString("- Do not include runtime metadata such as .opencode, .codex, .claude, .agents, workflow run artifacts, or command strings in any files/focusFiles array.\n")
	b.WriteString("- Include verification commands from the evidence. If dependencies are missing, still list the intended project commands.\n")
	b.WriteString("- Never include secret values or contents from .env files.\n\n")

	b.WriteString("## Deterministic Baseline\n")
	b.WriteString(fmt.Sprintf("- source: %s\n- validation: %s\n- summary: %s\n", index.Classification.Source, index.Classification.ValidationStatus, index.Classification.Summary))
	b.WriteString("\n## Stack\n")
	b.WriteString(fmt.Sprintf("- kind: %s\n", index.Surface.Stack.Kind))
	writeListLine(&b, "languages", index.Tools.Languages)
	writeListLine(&b, "frameworks", index.Tools.Frameworks)
	writeListLine(&b, "packageManagers", index.Tools.PackageManagers)
	writeListLine(&b, "testTools", index.Tools.TestTools)
	writeListLine(&b, "linters", index.Tools.Linters)
	writeListLine(&b, "formatters", index.Tools.Formatters)
	writeListLine(&b, "buildTools", index.Tools.BuildTools)

	b.WriteString("\n## Mandatory Coverage Files\n")
	for _, file := range mandatoryCoverageForClassifier(index) {
		b.WriteString("- ")
		b.WriteString(file)
		b.WriteString("\n")
	}

	b.WriteString("\n## Packages\n")
	for _, pkg := range importantPackagesForClassifier(index.Packages, 35) {
		b.WriteString(fmt.Sprintf("- %s:%s role=%s action=%s", pkg.Ecosystem, pkg.Name, pkg.Role, pkg.Action))
		if pkg.Version != "" {
			b.WriteString(" version=")
			b.WriteString(pkg.Version)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n## Repository Files\n")
	for _, file := range importantIndexFiles(index.Files, 110) {
		b.WriteString(fmt.Sprintf("- %s category=%s language=%s\n", file.Path, file.Category, file.Language))
	}

	b.WriteString("\n## Route/Test Edges\n")
	for _, edge := range LimitEdges(edgesByKind(index.Edges, "declares-route", 70), 70) {
		b.WriteString(fmt.Sprintf("- %s declares %s\n", edge.From, edge.To))
	}
	for _, edge := range LimitEdges(edgesByKind(index.Edges, "tests", 45), 45) {
		b.WriteString(fmt.Sprintf("- %s tests %s\n", edge.From, edge.To))
	}

	b.WriteString("\n## Baseline Entrypoints\n")
	for _, surface := range index.Classification.Entrypoints {
		for _, file := range surface.Files {
			b.WriteString("- ")
			b.WriteString(file)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n## Domains\n")
	for _, domain := range index.Classification.Domains {
		b.WriteString("- ")
		b.WriteString(domain.Name)
		b.WriteString(": ")
		b.WriteString(strings.Join(Limit(domain.Files, 5), ", "))
		b.WriteString("\n")
	}
	b.WriteString("\n## Verification Commands\n")
	for _, command := range index.Tools.RecommendedCommands {
		b.WriteString("- ")
		b.WriteString(command)
		b.WriteString("\n")
	}
	b.WriteString("\nReturn only the JSON object now.\n")
	return b.String()
}

func repoClassifierRepairPrompt(current RepoClassification, missing []string) string {
	data, err := json.Marshal(current)
	if err != nil {
		data = []byte("{}")
	}
	var b strings.Builder
	b.WriteString("# Repository Classification Repair\n\n")
	b.WriteString("You are repo-classifier for runweaver. The previous AI classification was valid JSON but missed mandatory coverage files.\n")
	b.WriteString("Return exactly one minified JSON object matching RepoClassification. Do not pretty-print. Do not wrap in markdown. Do not include prose outside JSON.\n")
	b.WriteString("Keep the same AI-based intent, agents, skills, and domains unless a small edit is needed to cover the missing files.\n")
	b.WriteString("Do not add deterministic/package fallback roles. Do not invent files. Add each missing file below to the most relevant existing domain.files, agent.focusFiles, or skill.focusFiles.\n")
	b.WriteString("Preserve domain-first ownership: domain agents remain primary; layer roles remain secondary support only.\n")
	b.WriteString("If a file is bootstrap/config such as src/main.ts or tsconfig*.json, put it in a config/bootstrap skill or agent.\n")
	b.WriteString("Keep the whole JSON under 12000 characters.\n\n")
	b.WriteString("## Missing Mandatory Files\n")
	for _, file := range missing {
		b.WriteString("- ")
		b.WriteString(file)
		b.WriteString("\n")
	}
	b.WriteString("\n## Previous Classification JSON\n")
	b.Write(data)
	b.WriteString("\n\nReturn the full repaired JSON object now.\n")
	return b.String()
}
