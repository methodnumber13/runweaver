package aitools

const featureDeliveryWorkflow = `{
  "id": "feature-delivery-swarm",
  "name": "Feature Delivery Swarm",
  "description": "Plan, implement, review, and verify a feature with repo-specific agents.",
  "maxConcurrent": 4,
  "maxParticipants": 4,
  "phases": [
    {
      "id": "plan",
      "name": "Plan And Ownership",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-surface-indexer", "repo-contract-reviewer"],
      "prompt": "Create a durable task plan under .runweaver/tmp. Apply the context-discipline skill. Read runtime profile semantic.domains, semantic.agents, repos[0].agents, and customSkills; select repo-specific participants by choosing one domain owner first, then at most two reviewers/skills unless maxParticipants explicitly allows more. Record selected participant names, participant rationale, files read, decisions, artifacts, and verification commands. Use portable agents only when no specific owner matches; prefer a domain owner when one matches."
    },
    {
      "id": "implement",
      "name": "Implementation",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "write",
      "concurrency": 1,
      "agents": ["repo-surface-engineer"],
      "prompt": "Implement through the selected domain participant from the durable plan. Keep edits scoped to that participant's focusFiles/domain. Record files read, files changed, decisions, artifacts, next action, and blockers with runweaver workflow update. Use layer reviewers only for contracts, tests, persistence, config, or metadata checks."
    },
    {
      "id": "review",
      "name": "Review And Verification",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-contract-reviewer", "repo-test-quality-reviewer", "agent-skill-drift-reviewer"],
      "prompt": "Review with selected domain participants plus contract/test/metadata reviewers from runtime profile. Check behavioral risk, tests, verification output, and metadata drift. Persist verification results and run runweaver workflow verify --repo . --resume latest before final response."
    }
  ]
}
`

const bugfixWorkflow = `{
  "id": "bugfix-swarm",
  "name": "Bugfix Investigation Swarm",
  "description": "Reproduce, isolate, fix, and verify a defect.",
  "maxConcurrent": 4,
  "maxParticipants": 4,
  "phases": [
    {
      "id": "reproduce",
      "name": "Reproduce And Trace",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-surface-indexer", "repo-test-quality-reviewer"],
      "prompt": "Find the failing path, existing tests, logs, fixtures, and likely owner files before editing. Apply the context-discipline skill. For read-only planning, do not suggest temporary source edits or console logging; use runtime-native read/glob and rg instead of cat/grep. Select one domain owner from runtime profile by matching focusFiles, semantic.domains, and customSkills, then add at most two reviewers/skills for test/contract/config risk. Treat matching customSkills as participants too; for guard/decorator/filter/auth middleware surfaces include the matching security skill such as security-middleware when present. Persist agents and skills together in participants/findings with participant rationale, files read, decisions, and artifacts. If the user did not request planning-only/read-only mode, mark reproduce complete with --complete-phase after diagnosis and continue to the fix phase."
    },
    {
      "id": "fix",
      "name": "Fix",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "write",
      "concurrency": 1,
      "agents": ["repo-surface-engineer"],
      "prompt": "Apply the smallest fix through the selected domain participant and add or update focused tests. Use generic repo-surface-engineer only when no AI-classified domain owner matches. Persist changed files, files read, findings, decisions, artifacts, blockers, and next action with runweaver workflow update --resume latest --phase fix --status in_progress. After the edit is complete, mark fix complete with --complete-phase and continue to verification."
    },
    {
      "id": "verify",
      "name": "Regression Verification",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-test-quality-reviewer", "repo-contract-reviewer"],
      "prompt": "Run focused regression checks chosen from the selected participant verification commands and summarize residual risk. Persist verification commands and outcomes with runweaver workflow update --resume latest --phase verify --status in_progress using --verification-result for exact outcomes. Mark verify complete with --complete-phase after checks finish so the workflow status becomes complete, then run runweaver workflow verify --repo . --resume latest and resolve warnings or record blockers."
    }
  ]
}
`

const refactorWorkflow = `{
  "id": "refactor-swarm",
  "name": "Refactor Safety Swarm",
  "description": "Change structure while preserving behavior and refreshing metadata.",
  "maxConcurrent": 4,
  "maxParticipants": 4,
  "phases": [
    {
      "id": "map-contracts",
      "name": "Map Contracts",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-surface-indexer", "repo-contract-reviewer"],
      "prompt": "Map public contracts, imports, tests, and generated metadata anchors affected by the refactor. Select affected domain participants from runtime profile and record their focusFiles before moving code; add layer reviewers for shared contracts."
    },
    {
      "id": "refactor",
      "name": "Refactor",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "write",
      "concurrency": 1,
      "agents": ["repo-surface-engineer"],
      "prompt": "Move or reshape code without behavior changes through the selected domain participant. Keep imports and tests coherent."
    },
    {
      "id": "metadata-refresh",
      "name": "Metadata Refresh",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "write",
      "concurrency": 1,
      "agents": ["agent-skill-drift-reviewer", "stale-anchor-fixer", "profile-regenerator"],
      "prompt": "Run runweaver refresh --repo . and apply approved runtime metadata updates if anchors changed. Preserve the current classification source; do not replace AI-classified agents/skills with deterministic/package roles unless the user requested deterministic mode."
    }
  ]
}
`

const testHardeningWorkflow = `{
  "id": "test-hardening-swarm",
  "name": "Test Hardening Swarm",
  "description": "Improve coverage, fixtures, and verification reliability around a target surface.",
  "maxConcurrent": 3,
  "maxParticipants": 3,
  "phases": [
    {
      "id": "test-map",
      "name": "Test Map",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-test-quality-reviewer", "repo-surface-indexer"],
      "prompt": "Find current tests, gaps, fixtures, mocks, and reliable commands for the target surface. Select domain participants first, then repo-specific test reviewers from runtime profile, and record the selected verification commands."
    },
    {
      "id": "test-update",
      "name": "Test Update",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "write",
      "concurrency": 1,
      "agents": ["repo-test-quality-reviewer"],
      "prompt": "Add or update focused tests and fixtures through the selected repo-specific participant without broad unrelated rewrites."
    },
    {
      "id": "verify",
      "name": "Verify",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-contract-reviewer"],
      "prompt": "Run selected verification commands from the repo-specific participants and record exact blockers if any command cannot run."
    }
  ]
}
`
