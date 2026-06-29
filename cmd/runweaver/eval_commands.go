package main

import (
	"fmt"
	"time"

	"github.com/methodnumber13/runweaver/internal/aitools"
)

func (c cli) evalCmd(args []string) error {
	if len(args) == 0 {
		return usageError{command: "eval adoption", err: fmt.Errorf("usage: runweaver eval adoption --repo <path> [--runtime id]")}
	}
	switch args[0] {
	case "adoption":
		return c.evalAdoptionCmd(args[1:])
	default:
		return usageError{command: "eval adoption", err: fmt.Errorf("usage: runweaver eval adoption --repo <path> [--runtime id]")}
	}
}

func (c cli) evalAdoptionCmd(args []string) error {
	fs := newFlagSet("eval adoption")
	repo := fs.String("repo", ".", "repository path")
	runtimeProvider := fs.String("runtime", aitools.RuntimeOpenCode, "runtime provider: opencode, codex, claude, or all")
	task := fs.String("task", "RunWeaver adoption smoke test", "smoke task for runweaver start")
	skipIndex := fs.Bool("skip-index", false, "skip automatic index refresh during start smoke")
	live := fs.Bool("live", false, "launch the selected runtime instead of preparing a dry-run command")
	timeout := fs.Duration("timeout", 0, "live runtime execution timeout, for example 2m; 0 disables the timeout")
	model := fs.String("model", "", "optional runtime model override for live execution")
	opencodeBin := fs.String("opencode-bin", "opencode", "OpenCode executable path for live execution")
	codexBin := fs.String("codex-bin", "codex", "Codex executable path for live execution")
	claudeBin := fs.String("claude-bin", "claude", "Claude Code executable path for live execution")
	skipGitRepoCheck := fs.Bool("skip-git-repo-check", false, "allow Codex live execution outside a Git repository")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "eval adoption", err: err}
	}
	if err := rejectExtraArgs(fs, "eval adoption"); err != nil {
		return err
	}
	result, err := aitools.EvaluateAdoption(*repo, aitools.AdoptionEvalOptions{
		Runtime:          *runtimeProvider,
		Task:             *task,
		SkipIndex:        *skipIndex,
		Live:             *live,
		Timeout:          time.Duration(*timeout),
		Model:            *model,
		OpencodeBin:      *opencodeBin,
		CodexBin:         *codexBin,
		ClaudeBin:        *claudeBin,
		SkipGitRepoCheck: *skipGitRepoCheck,
	})
	if err != nil {
		return fmt.Errorf("evaluate adoption: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	if result.Ready {
		c.printStatus("success", "adoption eval passed")
	} else {
		c.printStatus("warning", "adoption eval has issues; see JSON")
	}
	return nil
}
