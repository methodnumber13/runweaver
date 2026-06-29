package aitools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func executionPrompt(plan WorkflowRunSummary, runtimeID string, runweaverCommand string) string {
	displayName := runtimeDisplayName(runtimeID)
	profilePath := runtimeProfilePathForExecution(runtimeID)
	delegation := runtimeDelegationGuidance(runtimeID)
	runweaverCommand = fallbackString(strings.TrimSpace(runweaverCommand), "runweaver")
	return strings.TrimSpace(fmt.Sprintf(`# RunWeaver Swarm Execution

You are the repository primary swarm agent running through %s.
RunWeaver command for this execution: $RUNWEAVER_BIN (%s). Use $RUNWEAVER_BIN for every RunWeaver CLI call; do not use a different global runweaver binary.

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
5. Always refresh the deterministic index with $RUNWEAVER_BIN index --repo . --changed-only --prune --classifier-runtime %s before relying on stale anchors.
6. Continue phase by phase until the workflow is complete or a real blocker prevents safe progress.
7. Keep plans, checkpoints, logs, indexes, and generated proposals under .runweaver/tmp. Do not commit .runweaver/tmp. Resume only from .runweaver/tmp/swarm-runs.
8. At the end, run $RUNWEAVER_BIN workflow verify --repo . --resume latest. Fix verifier warnings when feasible or record a blocker.
9. Return workflow, run_state, participants, files_changed, verification, workflow_verify, risks, and resume instructions.
10. Do not use web search, browser tools, or unrelated global skills unless the task explicitly requires current external facts. Keep phase notes minimal; prefer runweaver workflow update over manually expanding phase files. If the verifier reports an over-cap or stale participant list, rerun workflow update with --replace-participants instead of editing checkpoint.json by hand. After the final checkpoint update and workflow verify, stop and return the summary.
`, displayName, runweaverCommand, plan.Task, plan.Workflow, plan.RunDir, plan.CheckpointPath, plan.TodoPath, profilePath, delegation, runtimeID))
}

func runweaverCommandForExecution() string {
	if executable, err := os.Executable(); err == nil && isRunWeaverExecutable(executable) {
		return executable
	}
	if path, err := exec.LookPath("runweaver"); err == nil {
		return path
	}
	return "runweaver"
}

func isRunWeaverExecutable(path string) bool {
	base := strings.TrimSuffix(strings.ToLower(filepath.Base(path)), ".exe")
	return base == "runweaver"
}

func withRunWeaverCommandEnv(env []string, command string) []string {
	command = strings.TrimSpace(command)
	if command == "" {
		return env
	}
	if len(env) == 0 {
		env = os.Environ()
	}
	key := "RUNWEAVER_BIN="
	out := make([]string, 0, len(env)+1)
	replaced := false
	for _, item := range env {
		if strings.HasPrefix(item, key) {
			out = append(out, key+command)
			replaced = true
			continue
		}
		out = append(out, item)
	}
	if !replaced {
		out = append(out, key+command)
	}
	return out
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
