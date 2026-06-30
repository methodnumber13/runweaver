package aitools

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// ExecuteWorkflow creates or resumes a workflow run and delegates it to a runtime.
func ExecuteWorkflow(repoPath string, opts WorkflowExecuteOptions) (WorkflowExecutionResult, error) {
	return executeWorkflow(repoPath, opts, runCommandToFiles)
}

func executeWorkflow(repoPath string, opts WorkflowExecuteOptions, runner commandRunner) (WorkflowExecutionResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return WorkflowExecutionResult{}, err
	}
	opts, err = normalizeExecuteOptions(opts)
	if err != nil {
		return WorkflowExecutionResult{}, err
	}
	cleanup := func() {}
	if opts.Runtime == RuntimeOpenCode {
		cleanup = trackOpenCodeDependencyArtifacts(root)
	}
	defer cleanup()

	var runtimePreflight *RuntimeDiscoveryResult
	if !opts.SkipRuntimeCheck {
		check := runtimeExecutionPreflight(root, opts)
		runtimePreflight = &check
		if !opts.DryRun && !check.BinaryFound {
			return WorkflowExecutionResult{}, fmt.Errorf("%s executable %q is not available on PATH", check.Name, check.Binary)
		}
	}

	var modelPreflight *ModelConfigCheck
	if opts.Runtime == RuntimeOpenCode && !opts.SkipModelCheck {
		check, err := CheckModelConfig(root, ModelConfigCheckOptions{ProviderID: opts.ProviderID})
		if err != nil {
			return WorkflowExecutionResult{}, fmt.Errorf("check model before execution: %w", err)
		}
		modelPreflight = &check
		if !check.Ready {
			return WorkflowExecutionResult{}, fmt.Errorf("OpenCode model preflight failed: %s", strings.Join(check.Issues, "; "))
		}
	}

	plan, err := workflowPlanForExecution(root, opts)
	if err != nil {
		return WorkflowExecutionResult{}, err
	}
	runweaverCommand := runweaverCommandForExecution()
	prompt := executionPrompt(plan, opts.Runtime, runweaverCommand)
	spec, err := prepareRuntimeExecutionSpec(root, opts, plan, prompt)
	if err != nil {
		return WorkflowExecutionResult{}, err
	}
	spec.Env = withRunWeaverCommandEnv(spec.Env, runweaverCommand)
	if err := os.WriteFile(spec.PromptPath, []byte(prompt), 0o644); err != nil {
		return WorkflowExecutionResult{}, fmt.Errorf("write %s execution prompt: %w", spec.DisplayName, err)
	}

	result := WorkflowExecutionResult{
		Status:           "planned",
		Summary:          spec.DisplayName + " execution command prepared",
		Runtime:          spec.Runtime,
		DryRun:           opts.DryRun,
		Executed:         false,
		Plan:             plan,
		PromptPath:       rel(root, spec.PromptPath),
		OutputPath:       relOrEmpty(root, spec.OutputPath),
		StdoutPath:       rel(root, spec.StdoutPath),
		StderrPath:       rel(root, spec.StderrPath),
		Command:          spec.Command,
		ModelPreflight:   modelPreflight,
		RuntimePreflight: runtimePreflight,
	}
	if opts.DryRun {
		_ = appendWorkflowEvent(root, plan.RunDir, WorkflowEvent{Type: "execute-dry-run", Command: spec.Command, At: Now()})
		return result, nil
	}

	_ = appendWorkflowEvent(root, plan.RunDir, WorkflowEvent{Type: "execute-start", Command: spec.Command, At: Now()})
	ctx := context.Background()
	cancel := func() {}
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
	}
	defer cancel()
	exitCode, err := runner(ctx, root, spec.Binary, spec.Args, spec.StdoutPath, spec.StderrPath, spec.Env)
	result.StdoutPath = rel(root, spec.StdoutPath)
	result.StderrPath = rel(root, spec.StderrPath)
	result.ExitCode = exitCode
	result.Executed = true
	result.Status = "success"
	result.Summary = spec.DisplayName + " execution finished"
	if err != nil {
		result.Status = "error"
		result.Summary = spec.DisplayName + " execution failed"
		_ = appendWorkflowEvent(root, plan.RunDir, WorkflowEvent{Type: "execute-failed", ExitCode: intPtr(exitCode), Error: err.Error(), At: Now()})
		return result, fmt.Errorf("run %s executor: %w", spec.DisplayName, err)
	}
	_ = appendWorkflowEvent(root, plan.RunDir, WorkflowEvent{Type: "execute-finished", ExitCode: intPtr(exitCode), At: Now()})
	postCheck := workflowExecutionPostCheck(root, plan)
	result.PostCheck = &postCheck
	if postCheck.Status == "warning" {
		result.Status = "warning"
		result.Summary = spec.DisplayName + " execution finished but workflow checkpoint did not advance"
		_ = appendWorkflowEvent(root, plan.RunDir, WorkflowEvent{Type: "execute-post-check-warning", Warnings: postCheck.Warnings, At: Now()})
	}
	return result, nil
}

func normalizeExecuteOptions(opts WorkflowExecuteOptions) (WorkflowExecuteOptions, error) {
	if opts.Runtime == "" {
		opts.Runtime = RuntimeOpenCode
	}
	opts.Runtime = normalizeRuntimeID(opts.Runtime)
	if _, ok := RuntimeProviderByID(opts.Runtime); !ok {
		return opts, fmt.Errorf("unsupported runtime %q; supported: opencode, codex, claude", opts.Runtime)
	}
	if opts.OpencodeBin == "" {
		opts.OpencodeBin = "opencode"
	}
	if opts.CodexBin == "" {
		opts.CodexBin = "codex"
	}
	if opts.ClaudeBin == "" {
		opts.ClaudeBin = "claude"
	}
	if opts.Agent == "" {
		opts.Agent = OpenCodePrimaryAgentName
	}
	if opts.Format == "" {
		opts.Format = "json"
	}
	if opts.Sandbox == "" {
		opts.Sandbox = "workspace-write"
	}
	if opts.ApprovalPolicy == "" {
		opts.ApprovalPolicy = "never"
	}
	if opts.PermissionMode == "" {
		opts.PermissionMode = "dontAsk"
	}
	if opts.ClaudeTools == "" {
		opts.ClaudeTools = "Read,Glob,Grep,Bash,Edit,MultiEdit,Write"
	}
	if opts.Runtime != RuntimeOpenCode && opts.Attach != "" {
		return opts, fmt.Errorf("--attach is only supported for OpenCode execution")
	}
	return opts, nil
}
