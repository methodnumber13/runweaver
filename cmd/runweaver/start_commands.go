package main

import (
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools"
)

func (c cli) startCmd(args []string) error {
	fs := newFlagSet("start")
	repo := fs.String("repo", ".", "repository path")
	task := fs.String("task", "", "task description")
	runtimeProvider := fs.String("runtime", aitools.RuntimeOpenCode, "runtime provider: opencode, codex, or claude")
	workflow := fs.String("workflow", "", "explicit workflow JSON path")
	profilePath := fs.String("profile", "", "explicit RunWeaver profile JSON path")
	skipIndex := fs.Bool("skip-index", false, "skip automatic index refresh")
	forceNew := fs.Bool("force-new", false, "create a new workflow even when latest matches")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "start", err: err}
	}
	if err := rejectExtraArgs(fs, "start"); err != nil {
		return err
	}
	result, err := aitools.StartWorkflow(*repo, aitools.StartOptions{
		Task:        *task,
		Runtime:     *runtimeProvider,
		Workflow:    *workflow,
		ProfilePath: *profilePath,
		SkipIndex:   *skipIndex,
		ForceNew:    *forceNew,
	})
	if err != nil {
		return fmt.Errorf("start workflow: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	c.printStatus("success", "workflow "+result.Action)
	return nil
}
