package aitools

import (
	"fmt"
	"strings"
)

func executionPrompt(plan WorkflowRunSummary, runtimeID string) string {
	displayName := runtimeDisplayName(runtimeID)
	profilePath := runtimeProfilePathForExecution(runtimeID)
	delegation := runtimeDelegationGuidance(runtimeID)
	return strings.TrimSpace(fmt.Sprintf(`# RunWeaver Swarm Execution

You are the repository primary swarm agent running through %s.

Task:
%s

Workflow: %s
Run directory: %s
Checkpoint: %s
Todo: %s

Execution rules:
1. Read plan.json, checkpoint.json, todo.md, %s, repo-context.md, repo-index.compact.json, manifest.json, and the context-discipline skill before editing. Load the full repo-index.json only when the compact context is insufficient.
2. Treat checkpoint.json as durable state. Update it after each phase with completedPhases, nextPhase, status, updatedAt, participants, participantRationale, findings, decisions, filesRead, filesChanged, artifacts, verification, verificationResults, blockers, and nextAction.
3. Update todo.md as phases complete. Append important lifecycle events to events.ndjson. Keep phase handoffs under phases/<phase>/handoff.md, notes under phases/<phase>/notes.md, and exact verification outcomes under phases/<phase>/verification.jsonl.
4. For each workflow phase, select repo-specific participants from the runtime profile by matching semantic.agents, repos[0].agents, customSkills, focusFiles, and workflow text to the task. Prefer one domain owner plus up to two reviewers/skills and do not exceed workflow maxParticipants. Use the agents listed in each phase as fallback participants. %s
5. Always refresh the deterministic index with runweaver index --repo . --changed-only --prune before relying on stale anchors.
6. Continue phase by phase until the workflow is complete or a real blocker prevents safe progress.
7. Keep plans, checkpoints, logs, indexes, and generated proposals under .runweaver/tmp. Do not commit .runweaver/tmp. Resume only from .runweaver/tmp/swarm-runs.
8. At the end, run runweaver workflow verify --repo . --resume latest. Fix verifier warnings when feasible or record a blocker.
9. Return workflow, run_state, participants, files_changed, verification, workflow_verify, risks, and resume instructions.
`, displayName, plan.Task, plan.Workflow, plan.RunDir, plan.CheckpointPath, plan.TodoPath, profilePath, delegation))
}

func runtimeDisplayName(runtimeID string) string {
	switch runtimeID {
	case RuntimeCodex:
		return "Codex"
	case RuntimeClaude:
		return "Claude Code"
	default:
		return "OpenCode"
	}
}

func runtimeProfilePathForExecution(runtimeID string) string {
	adapter, ok := RuntimeAdapterByID(runtimeID)
	if !ok {
		return openCodeRuntimeAdapter{}.ProfilePath()
	}
	return adapter.ProfilePath()
}

func runtimeDelegationGuidance(runtimeID string) string {
	adapter, ok := RuntimeAdapterByID(runtimeID)
	if !ok {
		return openCodeRuntimeAdapter{}.DelegationGuidance()
	}
	return adapter.DelegationGuidance()
}
