package aitools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runCommandOutput(ctx context.Context, dir string, name string, args []string) ([]byte, error) {
	return runCommandOutputWithEnv(ctx, dir, name, args, nil)
}

func runCommandOutputWithEnv(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("command path is empty")
	}
	stdout, err := os.CreateTemp("", "runweaver-opencode-*.out")
	if err != nil {
		return nil, fmt.Errorf("create command stdout buffer: %w", err)
	}
	stdoutPath := stdout.Name()
	defer os.Remove(stdoutPath)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = env
	}
	cmd.Stdout = stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	if closeErr := stdout.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	output, readErr := os.ReadFile(stdoutPath)
	if readErr != nil && err == nil {
		err = readErr
	}
	if err == nil {
		return output, nil
	}
	detail := strings.TrimSpace(stderr.String())
	if detail == "" {
		detail = err.Error()
	}
	if ctx.Err() != nil {
		return output, ctx.Err()
	}
	return output, fmt.Errorf("%s %s failed: %s", name, strings.Join(args, " "), detail)
}
