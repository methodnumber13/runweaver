package aitools

const repoIntelligenceWorkflow = `{
  "id": "repo-intelligence-swarm",
  "name": "Repository Intelligence Swarm",
  "description": "Smart initialization workflow: index repository technologies, packages, symbols, surfaces, and then shape runtime agents and skills.",
  "maxConcurrent": 5,
  "maxParticipants": 4,
  "phases": [
    {
      "id": "deterministic-index",
      "name": "Deterministic Repository Index",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-surface-indexer"],
      "prompt": "Run runweaver index --repo . --changed-only --prune. Read .runweaver/tmp/index/repo-context.md and repo-index.compact.json; open repo-index.json only if compact context is insufficient. Summarize languages, package managers, frameworks, test tools, linters, formatters, source categories, and ignored/generated surfaces."
    },
    {
      "id": "technology-review",
      "name": "Technology And Package Review",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-contract-reviewer", "repo-test-quality-reviewer"],
      "prompt": "Use the repo index and runtime profile semantic section to decide which repository-specific AI-classified agents and skills are required. Prefer domain ownership agents first, then layer reviewers/skills for contracts, persistence, tests, lint, and formatting. Do not merge AI and deterministic classifications; use deterministic evidence only as index context. Do not edit code."
    },
    {
      "id": "agent-skill-design",
      "name": "Agent And Skill Design",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["profile-regenerator", "repo-surface-indexer"],
      "prompt": "Propose repository-specific agents, skills, workflows, focus files, and verification commands from the current classification source. If semantic.source is model-backed-*, make domain agents the primary participants and layer roles secondary; do not add deterministic/package roles. Keep artifacts under .runweaver/tmp."
    },
    {
      "id": "initialization-apply",
      "name": "Initialization Apply",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "write",
      "concurrency": 1,
      "agents": ["profile-regenerator"],
      "prompt": "Apply only approved runtime metadata. runweaver init already creates deterministic baseline files; this phase is for reviewed refinements."
    }
  ]
}
`

const metadataRefreshWorkflow = `{
  "id": "metadata-refresh-swarm",
  "name": "Runtime Metadata Refresh Swarm",
  "description": "Refresh repository surface indexes and detect stale agent/skill metadata.",
  "maxConcurrent": 4,
  "maxParticipants": 4,
  "phases": [
    {
      "id": "surface-index",
      "name": "Repository Surface Index",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-surface-indexer"],
      "prompt": "Scan repository source, configs, routes, tests, packages, symbols, and generated surfaces. Prefer runweaver index --repo . --changed-only --prune and read .runweaver/tmp/index/repo-context.md plus repo-index.compact.json. Do not edit business code."
    },
    {
      "id": "drift-review",
      "name": "Agent Skill Drift Review",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["agent-skill-drift-reviewer", "stale-anchor-fixer"],
      "prompt": "Check runtime agents and skills for stale anchors and missing surfaces. Prefer runweaver refresh --repo . Do not edit business code."
    },
    {
      "id": "profile-update",
      "name": "Profile Update",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "write",
      "concurrency": 1,
      "agents": ["profile-regenerator"],
      "prompt": "Apply only approved runtime metadata/profile updates. Keep run artifacts under .runweaver/tmp."
    }
  ]
}
`

const repoOnboardingWorkflow = `{
  "id": "repo-onboarding-swarm",
  "name": "Repository Onboarding Swarm",
  "description": "Build a current map of repository rules, source layout, tests, and generated runtime metadata.",
  "maxConcurrent": 3,
  "maxParticipants": 3,
  "phases": [
    {
      "id": "read-rules",
      "name": "Read Repository Rules",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["swarm", "repo-surface-indexer"],
      "prompt": "Read runtime instructions, runtime config, runtime profile, and current source layout. Produce the task map and verification commands."
    },
    {
      "id": "surface-index",
      "name": "Surface Index",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-surface-indexer"],
      "prompt": "Run runweaver index --repo . --changed-only --prune and summarize important packages, tools, source surfaces, tests, symbols, and verification commands from repo-context.md and repo-index.compact.json."
    }
  ]
}
`
