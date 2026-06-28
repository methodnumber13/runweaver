package main

import (
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools"
)

func (c cli) statusCmd(args []string) error {
	fs := newFlagSet("status")
	repo := fs.String("repo", ".", "repository path")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "status", err: err}
	}
	if err := rejectExtraArgs(fs, "status"); err != nil {
		return err
	}
	result, err := aitools.RunWeaverStatus(*repo)
	if err != nil {
		return fmt.Errorf("read RunWeaver status: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	if result.Ready {
		c.printStatus("success", "RunWeaver workflow state is available")
	} else {
		c.printStatus("warning", "RunWeaver has no active workflow yet")
	}
	return nil
}
