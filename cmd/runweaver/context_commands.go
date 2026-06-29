package main

import (
	"fmt"

	"github.com/methodnumber13/runweaver/internal/aitools"
)

func (c cli) contextCmd(args []string) error {
	if len(args) == 0 {
		return usageError{command: "context query", err: fmt.Errorf("usage: runweaver context query --task <text> [--repo <path>]")}
	}
	switch args[0] {
	case "query":
		return c.contextQueryCmd(args[1:])
	default:
		return usageError{command: "context query", err: fmt.Errorf("usage: runweaver context query --task <text> [--repo <path>]")}
	}
}

func (c cli) contextQueryCmd(args []string) error {
	fs := newFlagSet("context query")
	repo := fs.String("repo", ".", "repository path")
	task := fs.String("task", "", "task description")
	limit := fs.Int("limit", 12, "max context files/symbols/routes/tests to return; clamped to 5..20")
	includeGenerated := fs.Bool("include-generated", false, "include generated files in context candidates")
	addJSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return usageError{command: "context query", err: err}
	}
	if err := rejectExtraArgs(fs, "context query"); err != nil {
		return err
	}
	result, err := aitools.QueryContext(*repo, aitools.ContextQueryOptions{
		Task:             *task,
		Limit:            *limit,
		IncludeGenerated: *includeGenerated,
	})
	if err != nil {
		return fmt.Errorf("query context: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	c.printStatus("success", "context query complete")
	return nil
}
