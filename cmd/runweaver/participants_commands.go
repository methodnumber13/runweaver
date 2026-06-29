package main

import (
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools"
)

func (c cli) participantsCmd(args []string) error {
	if len(args) == 0 {
		return usageError{command: "participants select", err: fmt.Errorf("usage: runweaver participants select --task <text> [--workflow file] [--repo <path>]")}
	}
	switch args[0] {
	case "select":
		return c.participantsSelectCmd(args[1:])
	default:
		return usageError{command: "participants select", err: fmt.Errorf("usage: runweaver participants select --task <text> [--workflow file] [--repo <path>]")}
	}
}

func (c cli) participantsSelectCmd(args []string) error {
	fs := newFlagSet("participants select")
	repo := fs.String("repo", ".", "repository path")
	task := fs.String("task", "", "task description")
	workflow := fs.String("workflow", "", "workflow JSON path; selected automatically when omitted")
	runtimeProvider := fs.String("runtime", aitools.RuntimeAuto, "runtime provider profile to inspect: auto, opencode, codex, or claude")
	profilePath := fs.String("profile", "", "explicit RunWeaver profile JSON path")
	addJSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return usageError{command: "participants select", err: err}
	}
	if err := rejectExtraArgs(fs, "participants select"); err != nil {
		return err
	}
	result, err := aitools.SelectParticipants(*repo, aitools.ParticipantSelectOptions{
		Task:        *task,
		Workflow:    *workflow,
		Runtime:     *runtimeProvider,
		ProfilePath: *profilePath,
	})
	if err != nil {
		return fmt.Errorf("select participants: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	c.printStatus("success", "participants selected")
	return nil
}
