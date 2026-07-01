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
      "prompt": "Create a durable task plan under .runweaver/tmp. Apply the context-discipline skill. Read runtime profile semantic.domains, semantic.agents, repos[0].agents, and customSkills; select repo-specific participants by choosing one domain owner first, then at most two reviewers/skills unless maxParticipants explicitly allows more. Record selected participant names, participant rationale, files read, decisions, artifacts, lastResult, nextAction, nextVerification, and verification commands. Use portable agents only when no specific owner matches; prefer a domain owner when one matches."
    },
    {
      "id": "implement",
      "name": "Implementation",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "write",
      "concurrency": 1,
      "agents": ["repo-surface-engineer"],
      "prompt": "Implement through the selected domain participant from the durable plan. Keep edits scoped to that participant's focusFiles/domain. For UI additions, prefer a cohesive small component/helper for a new summary, panel, card, dialog section, or repeated block instead of inlining more structure into an already large component. Record files read with --file-read, files changed with --file-changed, decisions with --decision, artifacts with --artifact, lastResult with --last-result, rejectedPaths with --rejected-path, next action with --next-action, nextVerification with --next-verification, and blockers with --blocker via $RUNWEAVER_BIN workflow update. Do not invent plural changed-file flags. Use layer reviewers only for contracts, tests, persistence, config, or metadata checks."
    },
    {
      "id": "review",
      "name": "Review And Verification",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["post-implementation-reviewer", "repo-contract-reviewer", "repo-test-quality-reviewer"],
      "prompt": "Review with selected domain participants plus post-implementation, contract, and test reviewers from runtime profile. Compare every acceptanceCriteria item from the execution contract/current checkpoint against changed code, tests, and verification evidence; record unmet criteria as blockers or residual risks. Check behavioral risk, stale async/race outcomes, allowed/forbidden state semantics, loading/disabled/error states, cohesive component/helper extraction for new summaries/panels/sections, unnecessary excluded-surface edits, unnecessary broad test scaffolding, verification output, and metadata drift. For metadata drift during ordinary feature work, refresh only the deterministic index with $RUNWEAVER_BIN index --repo . --changed-only --prune; do not run $RUNWEAVER_BIN refresh unless the user explicitly requested metadata regeneration. Persist lastResult, verification results, rejectedPaths, blockers, and nextVerification. Mark review complete with --complete-phase after verification/build/static checks finish, then run $RUNWEAVER_BIN workflow verify --repo . --resume latest before final response."
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
      "prompt": "Find the failing path, existing tests, logs, fixtures, and likely owner files before editing. Apply the context-discipline skill. For read-only planning, do not suggest temporary source edits or console logging; use runtime-native read/glob and rg instead of cat/grep. Select one domain owner from runtime profile by matching focusFiles, semantic.domains, and customSkills, then add at most two reviewers/skills for test/contract/config risk. Treat matching customSkills as participants too; for guard/decorator/filter/auth middleware surfaces include the matching security skill such as security-middleware when present. Persist agents and skills together in participants/findings with participant rationale, files read, decisions, artifacts, lastResult, rejectedPaths, nextAction, and nextVerification. If the user did not request planning-only/read-only mode, mark reproduce complete with --complete-phase after diagnosis and continue to the fix phase."
    },
    {
      "id": "fix",
      "name": "Fix",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "write",
      "concurrency": 1,
      "agents": ["repo-surface-engineer"],
      "prompt": "Apply the smallest fix through the selected domain participant and add or update focused tests. Use generic repo-surface-engineer only when no AI-classified domain owner matches. Persist changed files with repeated --file-changed, files read with repeated --file-read, findings with --finding, decisions with --decision, artifacts with --artifact, lastResult with --last-result, rejectedPaths with --rejected-path, blockers with --blocker, next action with --next-action, and nextVerification with --next-verification via $RUNWEAVER_BIN workflow update --resume latest --phase fix --status in_progress. Do not invent plural changed-file flags. After the edit is complete, mark fix complete with --complete-phase and continue to verification."
    },
    {
      "id": "verify",
      "name": "Regression Verification",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["post-implementation-reviewer", "repo-test-quality-reviewer", "repo-contract-reviewer"],
      "prompt": "Run focused regression checks chosen from the selected participant verification commands and summarize residual risk. Include post-implementation review for behavioral regressions, stale async/race outcomes, allowed/forbidden state semantics, scope creep, and unnecessary test scaffolding. Persist lastResult, verification commands, outcomes, rejectedPaths, residual risks, blockers, and nextVerification with $RUNWEAVER_BIN workflow update --resume latest --phase verify --status in_progress using --verification-result for exact outcomes and --residual-risk for accepted warning diagnostics. Mark verify complete with --complete-phase after checks finish so the workflow status becomes complete, then run $RUNWEAVER_BIN workflow verify --repo . --resume latest and resolve warnings or record blockers/residual risks."
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
      "prompt": "Map public contracts, imports, tests, and generated metadata anchors affected by the refactor. Select affected domain participants from runtime profile and record their focusFiles before moving code; add layer reviewers for shared contracts. Persist lastResult, rejectedPaths, nextAction, and nextVerification for resume."
    },
    {
      "id": "refactor",
      "name": "Refactor",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "write",
      "concurrency": 1,
      "agents": ["repo-surface-engineer"],
      "prompt": "Move or reshape code without behavior changes through the selected domain participant. Keep imports and tests coherent. Persist changed files with --file-changed, lastResult with --last-result, rejectedPaths with --rejected-path, nextAction with --next-action, and nextVerification with --next-verification."
    },
    {
      "id": "metadata-refresh",
      "name": "Metadata Refresh",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "write",
      "concurrency": 1,
      "agents": ["agent-skill-drift-reviewer", "stale-anchor-fixer", "profile-regenerator"],
      "prompt": "Run $RUNWEAVER_BIN refresh --repo . and apply approved runtime metadata updates if anchors changed. Preserve the current classification source; do not replace AI-classified agents/skills with deterministic/package roles unless the user requested deterministic mode. Persist refresh result, stale-index signal, rejectedPaths, nextAction, and nextVerification."
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
      "prompt": "Find current tests, gaps, fixtures, mocks, and reliable commands for the target surface. Select domain participants first, then repo-specific test reviewers from runtime profile, and record selected verification commands, lastResult, nextAction, and nextVerification."
    },
    {
      "id": "test-update",
      "name": "Test Update",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "write",
      "concurrency": 1,
      "agents": ["repo-test-quality-reviewer"],
      "prompt": "Add or update focused tests and fixtures through the selected repo-specific participant without broad unrelated rewrites. Persist changed files with --file-changed, rejectedPaths with --rejected-path, lastResult with --last-result, nextAction with --next-action, and nextVerification with --next-verification."
    },
    {
      "id": "verify",
      "name": "Verify",
      "scope": "repo",
      "mode": "parallel",
      "writeMode": "read",
      "agents": ["repo-contract-reviewer"],
      "prompt": "Run selected verification commands from the repo-specific participants and record exact lastResult, verificationResults, blockers, and nextVerification if any command cannot run."
    }
  ]
}
`
