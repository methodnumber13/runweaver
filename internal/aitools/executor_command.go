package aitools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func runCommandToFiles(ctx context.Context, dir string, name string, args []string, stdoutPath string, stderrPath string, env []string) (int, error) {
	if name == "" {
		return -1, fmt.Errorf("runtime binary path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(stdoutPath), 0o755); err != nil {
		return -1, fmt.Errorf("create output directory: %w", err)
	}
	stdout, err := os.Create(stdoutPath)
	if err != nil {
		return -1, fmt.Errorf("create stdout log: %w", err)
	}
	defer stdout.Close()
	stderr, err := os.Create(stderrPath)
	if err != nil {
		return -1, fmt.Errorf("create stderr log: %w", err)
	}
	defer stderr.Close()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = env
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	if err == nil {
		return 0, nil
	}
	if ctx.Err() != nil {
		return -1, ctx.Err()
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), err
	}
	return -1, err
}
