package main

import (
	"fmt"
	"time"

	"github.com/methodnumber13/runweaver/internal/aitools"
)

func (c cli) smokeCmd(args []string) error {
	if len(args) == 0 {
		return usageError{command: "smoke codex", err: fmt.Errorf("usage: runweaver smoke codex [--live]")}
	}
	switch args[0] {
	case "codex":
		return c.smokeCodexCmd(args[1:])
	default:
		return usageError{command: "smoke codex", err: fmt.Errorf("usage: runweaver smoke codex [--live]")}
	}
}

func (c cli) smokeCodexCmd(args []string) error {
	fs := newFlagSet("smoke codex")
	repo := fs.String("repo", "", "empty repository path to create/use; defaults to a temporary disposable repo")
	force := fs.Bool("force", false, "allow an existing non-empty repo path and overwrite smoke fixture/RunWeaver metadata")
	keep := fs.Bool("keep", false, "keep the generated temporary repo after the smoke run")
	live := fs.Bool("live", false, "launch Codex instead of preparing a dry-run execution command")
	timeout := fs.Duration("timeout", 4*time.Minute, "live Codex execution timeout")
	model := fs.String("model", "", "optional Codex model override")
	codexBin := fs.String("codex-bin", "codex", "Codex executable path")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "smoke codex", err: err}
	}
	if err := rejectExtraArgs(fs, "smoke codex"); err != nil {
		return err
	}
	result, err := aitools.RunCodexSmoke(aitools.CodexSmokeOptions{
		Repo:     *repo,
		Force:    *force,
		Keep:     *keep,
		Live:     *live,
		Timeout:  time.Duration(*timeout),
		Model:    *model,
		CodexBin: *codexBin,
	})
	if err != nil {
		return fmt.Errorf("run Codex smoke: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	if result.Ready {
		c.printStatus("success", "Codex smoke passed")
	} else {
		c.printStatus("warning", "Codex smoke has issues; see JSON")
	}
	return nil
}
