package aitools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const runWeaverProjectBin = ".runweaver/tmp/bin/runweaver"

var requiredRunWeaverCommands = []string{
	"start --repo",
	"status --repo",
	"context query",
}

func runWeaverProjectBinPath(root string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(root, ".runweaver", "tmp", "bin", "runweaver.cmd")
	}
	return filepath.Join(root, filepath.FromSlash(runWeaverProjectBin))
}

func runWeaverProjectBinForDocs() string {
	if runtime.GOOS == "windows" {
		return ".runweaver/tmp/bin/runweaver.cmd"
	}
	return runWeaverProjectBin
}

func writeRunWeaverCommandShim(root string) (string, error) {
	target := runweaverCommandForExecution()
	path := runWeaverProjectBinPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	var content string
	if runtime.GOOS == "windows" {
		content = "@echo off\r\n" + windowsCommandQuote(target) + " %*\r\n"
	} else {
		content = "#!/usr/bin/env sh\nexec " + shellQuote(target) + " \"$@\"\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return "", err
	}
	return rel(root, path), nil
}

func runWeaverCommandCandidates(root string) []string {
	var candidates []string
	if env := strings.TrimSpace(os.Getenv("RUNWEAVER_BIN")); env != "" {
		candidates = append(candidates, env)
	}
	if root != "" {
		candidates = append(candidates, runWeaverProjectBinPath(root))
	}
	if executable, err := os.Executable(); err == nil && isRunWeaverExecutable(executable) {
		candidates = append(candidates, executable)
	}
	if path, err := exec.LookPath("runweaver"); err == nil {
		candidates = append(candidates, path)
	}
	return uniqueStrings(candidates)
}

func runWeaverCommandSurfaceCheck(path string) ([]string, []string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "help")
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, requiredRunWeaverCommands, fmt.Errorf("runweaver help timed out")
	}
	if err != nil {
		return nil, requiredRunWeaverCommands, fmt.Errorf("runweaver help failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	text := strings.ToLower(string(output))
	var found []string
	var missing []string
	for _, command := range requiredRunWeaverCommands {
		if strings.Contains(text, strings.ToLower(command)) {
			found = append(found, command)
		} else {
			missing = append(missing, command)
		}
	}
	return found, missing, nil
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func windowsCommandQuote(value string) string {
	if value == "" {
		return `""`
	}
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}
