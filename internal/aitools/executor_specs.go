package aitools

import (
	"github.com/methodnumber13/runweaver/internal/aitools/runtimeenv"
	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
	"path/filepath"
)

func prepareRuntimeExecutionSpec(root string, opts WorkflowExecuteOptions, plan WorkflowRunSummary, prompt string) (workflowRuntimeExecutionSpec, error) {
	adapter, err := mustRuntimeAdapter(opts.Runtime)
	if err != nil {
		return workflowRuntimeExecutionSpec{}, err
	}
	return adapter.ExecutionSpec(root, opts, plan, prompt)
}

func openCodeExecutionSpec(root string, opts WorkflowExecuteOptions, plan WorkflowRunSummary, prompt string) workflowRuntimeExecutionSpec {
	runDir := filepath.Join(root, plan.RunDir)
	args := opencodeRunArgs(root, opts, plan, prompt)
	return workflowRuntimeExecutionSpec{
		Runtime:     RuntimeOpenCode,
		DisplayName: "OpenCode",
		Binary:      opts.OpencodeBin,
		Args:        args,
		Command:     append([]string{opts.OpencodeBin}, args...),
		Env:         runtimeenv.OpenCodeProviderEnv(root, opts.ProviderID),
		PromptPath:  filepath.Join(runDir, "opencode-exec-prompt.md"),
		StdoutPath:  filepath.Join(runDir, "opencode-stdout.jsonl"),
		StderrPath:  filepath.Join(runDir, "opencode-stderr.log"),
	}
}

func codexExecutionSpec(root string, opts WorkflowExecuteOptions, plan WorkflowRunSummary, prompt string) workflowRuntimeExecutionSpec {
	runDir := filepath.Join(root, plan.RunDir)
	outputPath := filepath.Join(runDir, "codex-final-message.md")
	args := []string{
		"-a", opts.ApprovalPolicy,
		"exec",
		"--json",
		"--ephemeral",
		"-C", root,
		"--sandbox", opts.Sandbox,
		"--color", "never",
		"--output-last-message", outputPath,
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.SkipGitRepoCheck {
		args = append(args, "--skip-git-repo-check")
	}
	args = append(args, prompt)
	return workflowRuntimeExecutionSpec{
		Runtime:     RuntimeCodex,
		DisplayName: "Codex",
		Binary:      opts.CodexBin,
		Args:        args,
		Command:     append([]string{opts.CodexBin}, args...),
		PromptPath:  filepath.Join(runDir, "codex-exec-prompt.md"),
		OutputPath:  outputPath,
		StdoutPath:  filepath.Join(runDir, "codex-stdout.jsonl"),
		StderrPath:  filepath.Join(runDir, "codex-stderr.log"),
	}
}

func claudeExecutionSpec(root string, opts WorkflowExecuteOptions, plan WorkflowRunSummary, prompt string) workflowRuntimeExecutionSpec {
	runDir := filepath.Join(root, plan.RunDir)
	args := []string{
		"--print",
		"--output-format", "stream-json",
		"--permission-mode", opts.PermissionMode,
		"--tools", opts.ClaudeTools,
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	args = append(args, prompt)
	return workflowRuntimeExecutionSpec{
		Runtime:     RuntimeClaude,
		DisplayName: "Claude Code",
		Binary:      opts.ClaudeBin,
		Args:        args,
		Command:     append([]string{opts.ClaudeBin}, args...),
		PromptPath:  filepath.Join(runDir, "claude-exec-prompt.md"),
		StdoutPath:  filepath.Join(runDir, "claude-stdout.jsonl"),
		StderrPath:  filepath.Join(runDir, "claude-stderr.log"),
	}
}

func phaseIDsFromRunPlan(root, runDir string) []string {
	path := runDir
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, runDir)
	}
	var plan WorkflowPlanFile
	if err := ReadJSON(filepath.Join(path, "plan.json"), &plan); err != nil {
		return nil
	}
	phaseIDs := make([]string, 0, len(plan.Workflow.Phases))
	for _, phase := range plan.Workflow.Phases {
		if phase.ID != "" {
			phaseIDs = append(phaseIDs, phase.ID)
		}
	}
	return phaseIDs
}

func opencodeRunArgs(root string, opts WorkflowExecuteOptions, plan WorkflowRunSummary, prompt string) []string {
	args := []string{"run", "--agent", opts.Agent, "--dir", root, "--format", opts.Format, "--title", "runweaver " + plan.Workflow}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.Attach != "" {
		args = append(args, "--attach", opts.Attach)
	}
	for _, path := range []string{
		filepath.Join(root, plan.RunDir, "plan.json"),
		filepath.Join(root, plan.CheckpointPath),
		filepath.Join(root, plan.TodoPath),
		filepath.Join(root, ".opencode", "swarm", "profile.json"),
		statepath.TmpPath(root, "index", "repo-context.md"),
		statepath.TmpPath(root, "index", "repo-index.compact.json"),
		statepath.TmpPath(root, "index", "manifest.json"),
	} {
		if Exists(path) {
			args = append(args, "--file", path)
		}
	}
	fullIndexPath := statepath.TmpPath(root, "index", "repo-index.json")
	compactPath := statepath.TmpPath(root, "index", "repo-index.compact.json")
	if Exists(fullIndexPath) && !Exists(compactPath) {
		args = append(args, "--file", fullIndexPath)
	}
	args = append(args, prompt)
	return args
}
