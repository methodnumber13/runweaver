package main

import (
	"fmt"
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
	if err := fs.Parse(args); err != nil {
		return usageError{command: "eval adoption", err: err}
	}
	if err := rejectExtraArgs(fs, "eval adoption"); err != nil {
		return err
	}
	result, err := aitools.EvaluateAdoption(*repo, aitools.AdoptionEvalOptions{
		Runtime:   *runtimeProvider,
		Task:      *task,
		SkipIndex: *skipIndex,
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
