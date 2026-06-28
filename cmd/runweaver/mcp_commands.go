package main

import (
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools"
	"os"
)

func (c cli) mcpCmd(args []string) error {
	if len(args) == 0 {
		return usageError{command: "mcp serve", err: fmt.Errorf("mcp subcommand is required")}
	}
	switch args[0] {
	case "serve":
		return c.mcpServeCmd(args[1:])
	default:
		return usageError{command: "mcp serve", err: fmt.Errorf("unknown mcp subcommand %q", args[0])}
	}
}

func (c cli) mcpServeCmd(args []string) error {
	fs := newFlagSet("mcp serve")
	repo := fs.String("repo", ".", "repository path")
	allowWorkflowWrites := fs.Bool("allow-workflow-writes", false, "expose MCP tools that create/update .runweaver workflow state")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "mcp serve", err: err}
	}
	if err := rejectExtraArgs(fs, "mcp serve"); err != nil {
		return err
	}
	stdin := c.stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	return aitools.ServeMCPStdio(stdin, c.stdout, aitools.MCPServerOptions{
		RepoPath:            *repo,
		Version:             resolveVersionInfo().Version,
		AllowWorkflowWrites: *allowWorkflowWrites,
	})
}
