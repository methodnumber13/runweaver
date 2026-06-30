package main

import (
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools"
)

func (c cli) workflowCmd(args []string) error {
	if len(args) == 0 {
		return usageError{command: "workflow run", err: fmt.Errorf("usage: runweaver workflow run --workflow <file> --task <text> [--repo <path>]")}
	}
	switch args[0] {
	case "update":
		return c.workflowUpdateCmd(args[1:])
	case "verify":
		return c.workflowVerifyCmd(args[1:])
	case "select":
		return c.workflowSelectCmd(args[1:])
	case "run":
	default:
		return usageError{command: "workflow run", err: fmt.Errorf("usage: runweaver workflow run --workflow <file> --task <text> [--repo <path>]")}
	}
	fs := newFlagSet("workflow run")
	repo := fs.String("repo", ".", "repository path")
	workflow := fs.String("workflow", "", "workflow JSON path")
	task := fs.String("task", "", "task description")
	resume := fs.String("resume", "", "run directory or latest")
	status := fs.Bool("status", false, "print workflow status")
	execute := fs.Bool("execute", false, "execute the workflow through the selected runtime after creating/loading the plan")
	dryRun := fs.Bool("dry-run", false, "prepare the runtime execution command without running it")
	runtimeProvider := fs.String("runtime", aitools.RuntimeOpenCode, "runtime provider for execution: opencode, codex, or claude")
	skipModelCheck := fs.Bool("skip-model-check", false, "skip OpenCode model preflight before execution")
	opencodeBin := fs.String("opencode-bin", "opencode", "OpenCode executable path")
	codexBin := fs.String("codex-bin", "codex", "Codex executable path")
	claudeBin := fs.String("claude-bin", "claude", "Claude Code executable path")
	agent := fs.String("agent", aitools.OpenCodePrimaryAgentName, "OpenCode primary agent for execution")
	provider := fs.String("provider", "", "OpenCode provider id for model preflight; defaults to the provider prefix in the configured model")
	model := fs.String("model", "", "optional runtime model override")
	attach := fs.String("attach", "", "optional opencode serve URL to attach to")
	sandbox := fs.String("sandbox", "workspace-write", "Codex sandbox mode: read-only, workspace-write, or danger-full-access")
	approvalPolicy := fs.String("approval-policy", "never", "Codex approval policy: never, on-request, on-failure, or untrusted")
	permissionMode := fs.String("permission-mode", "dontAsk", "Claude permission mode, for example dontAsk, acceptEdits, plan, or default")
	claudeTools := fs.String("claude-tools", "Read,Glob,Grep,Bash,Edit,MultiEdit,Write", "Claude tools list for --tools")
	skipGitRepoCheck := fs.Bool("skip-git-repo-check", false, "allow Codex execution outside a Git repository")
	skipRuntimeCheck := fs.Bool("skip-runtime-check", false, "skip runtime binary/config discovery before execution")
	if err := fs.Parse(args[1:]); err != nil {
		return usageError{command: "workflow run", err: err}
	}
	if err := rejectExtraArgs(fs, "workflow run"); err != nil {
		return err
	}
	if *status && *execute {
		return usageError{command: "workflow run", err: fmt.Errorf("--status cannot be combined with --execute")}
	}
	if *status || *resume != "" {
		if *execute {
			result, err := aitools.ExecuteWorkflow(*repo, aitools.WorkflowExecuteOptions{
				WorkflowPath:     *workflow,
				Task:             *task,
				Resume:           *resume,
				Runtime:          *runtimeProvider,
				OpencodeBin:      *opencodeBin,
				CodexBin:         *codexBin,
				ClaudeBin:        *claudeBin,
				Agent:            *agent,
				ProviderID:       *provider,
				Model:            *model,
				Attach:           *attach,
				Sandbox:          *sandbox,
				ApprovalPolicy:   *approvalPolicy,
				PermissionMode:   *permissionMode,
				ClaudeTools:      *claudeTools,
				SkipGitRepoCheck: *skipGitRepoCheck,
				SkipRuntimeCheck: *skipRuntimeCheck,
				DryRun:           *dryRun,
				SkipModelCheck:   *skipModelCheck,
			})
			if err != nil {
				if result.Command != nil {
					_ = c.printJSON(result)
				}
				return commandError{command: "workflow run", err: fmt.Errorf("execute workflow: %w", err)}
			}
			if err := c.printJSON(result); err != nil {
				return err
			}
			c.printStatus("success", result.Summary)
			return nil
		}
		value, err := aitools.WorkflowStatus(*repo, *resume)
		if err != nil {
			return usageError{command: "workflow run", err: fmt.Errorf("read workflow status: %w", err)}
		}
		if err := c.printJSON(value); err != nil {
			return err
		}
		c.printStatus("success", "workflow status loaded")
		return nil
	}
	if *execute {
		result, err := aitools.ExecuteWorkflow(*repo, aitools.WorkflowExecuteOptions{
			WorkflowPath:     *workflow,
			Task:             *task,
			Runtime:          *runtimeProvider,
			OpencodeBin:      *opencodeBin,
			CodexBin:         *codexBin,
			ClaudeBin:        *claudeBin,
			Agent:            *agent,
			ProviderID:       *provider,
			Model:            *model,
			Attach:           *attach,
			Sandbox:          *sandbox,
			ApprovalPolicy:   *approvalPolicy,
			PermissionMode:   *permissionMode,
			ClaudeTools:      *claudeTools,
			SkipGitRepoCheck: *skipGitRepoCheck,
			SkipRuntimeCheck: *skipRuntimeCheck,
			DryRun:           *dryRun,
			SkipModelCheck:   *skipModelCheck,
		})
		if err != nil {
			if result.Command != nil {
				_ = c.printJSON(result)
			}
			return commandError{command: "workflow run", err: fmt.Errorf("execute workflow: %w", err)}
		}
		if err := c.printJSON(result); err != nil {
			return err
		}
		c.printStatus("success", result.Summary)
		return nil
	}
	result, err := aitools.PlanWorkflow(*repo, *workflow, *task)
	if err != nil {
		return fmt.Errorf("plan workflow: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	c.printStatus("success", "workflow plan/checkpoint created")
	return nil
}

func (c cli) workflowSelectCmd(args []string) error {
	fs := newFlagSet("workflow select")
	repo := fs.String("repo", ".", "repository path")
	task := fs.String("task", "", "task description")
	workflow := fs.String("workflow", "", "explicit workflow JSON path")
	addJSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return usageError{command: "workflow select", err: err}
	}
	if err := rejectExtraArgs(fs, "workflow select"); err != nil {
		return err
	}
	result, err := aitools.SelectWorkflow(*repo, aitools.WorkflowSelectOptions{
		Task:     *task,
		Workflow: *workflow,
	})
	if err != nil {
		return fmt.Errorf("select workflow: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	c.printStatus("success", "workflow selected")
	return nil
}

func (c cli) workflowUpdateCmd(args []string) error {
	fs := newFlagSet("workflow update")
	repo := fs.String("repo", ".", "repository path")
	resume := fs.String("resume", "latest", "run directory or latest")
	phase := fs.String("phase", "", "current workflow phase")
	status := fs.String("status", "", "checkpoint status, for example in_progress or complete")
	participants := fs.String("participants", "", "comma-separated participant names")
	replaceParticipants := fs.Bool("replace-participants", false, "replace checkpoint participants instead of appending")
	lastResult := fs.String("last-result", "", "last command, agent, or phase result that explains why the workflow is moving or pausing")
	nextAction := fs.String("next-action", "", "next action to persist in checkpoint")
	nextVerification := fs.String("next-verification", "", "next verification step that should be run before continuing or finishing")
	completePhase := fs.Bool("complete-phase", false, "mark the current phase complete and advance nextPhase")
	var participantRationale repeatedStringFlag
	var findings repeatedStringFlag
	var decisions repeatedStringFlag
	var filesRead repeatedStringFlag
	var filesChanged repeatedStringFlag
	var artifacts repeatedStringFlag
	var rejectedPaths repeatedStringFlag
	var verification repeatedStringFlag
	var verificationResults repeatedStringFlag
	var blockers repeatedStringFlag
	fs.Var(&participantRationale, "participant-rationale", "participant selection rationale to append; may be repeated")
	fs.Var(&findings, "finding", "finding to append; may be repeated")
	fs.Var(&decisions, "decision", "decision to append; may be repeated")
	fs.Var(&filesRead, "file-read", "file read during the phase; may be repeated")
	fs.Var(&filesChanged, "file-changed", "file changed during the phase; may be repeated")
	fs.Var(&artifacts, "artifact", "artifact path to append; may be repeated")
	fs.Var(&rejectedPaths, "rejected-path", "path, command, or approach rejected/paused with reason; may be repeated")
	fs.Var(&verification, "verification", "verification command to append; may be repeated")
	fs.Var(&verificationResults, "verification-result", "verification result to append; may be repeated")
	fs.Var(&blockers, "blocker", "blocker to append; may be repeated")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "workflow update", err: err}
	}
	if err := rejectExtraArgs(fs, "workflow update"); err != nil {
		return err
	}
	result, err := aitools.UpdateWorkflow(*repo, aitools.WorkflowUpdateOptions{
		Resume:               *resume,
		Phase:                *phase,
		Status:               *status,
		Participants:         splitCSV(*participants),
		ReplaceParticipants:  *replaceParticipants,
		ParticipantRationale: participantRationale.Values(),
		Findings:             findings.Values(),
		Decisions:            decisions.Values(),
		FilesRead:            filesRead.Values(),
		FilesChanged:         filesChanged.Values(),
		Artifacts:            artifacts.Values(),
		LastResult:           *lastResult,
		RejectedPaths:        rejectedPaths.Values(),
		NextAction:           *nextAction,
		NextVerification:     *nextVerification,
		Verification:         verification.Values(),
		VerificationResults:  verificationResults.Values(),
		Blockers:             blockers.Values(),
		CompletePhase:        *completePhase,
	})
	if err != nil {
		return fmt.Errorf("update workflow checkpoint: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	c.printStatus("success", "workflow checkpoint updated")
	return nil
}

func (c cli) workflowVerifyCmd(args []string) error {
	fs := newFlagSet("workflow verify")
	repo := fs.String("repo", ".", "repository path")
	resume := fs.String("resume", "latest", "run directory or latest")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "workflow verify", err: err}
	}
	if err := rejectExtraArgs(fs, "workflow verify"); err != nil {
		return err
	}
	result, err := aitools.VerifyWorkflowRun(*repo, *resume)
	if err != nil {
		return fmt.Errorf("verify workflow run: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	if result.Ready {
		c.printStatus("success", "workflow run verification passed")
	} else if result.Status == "error" {
		return commandError{command: "workflow verify", err: fmt.Errorf("workflow run verification failed; see JSON checks")}
	} else {
		c.printStatus("warning", "workflow run verification has warnings; see JSON checks")
	}
	return nil
}
