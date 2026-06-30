package aitools

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CodexSmokeOptions configures a disposable Codex adoption smoke run.
type CodexSmokeOptions struct {
	Repo     string
	Force    bool
	Keep     bool
	Live     bool
	CodexBin string
	Model    string
	Timeout  time.Duration
}

// CodexSmokeResult describes a dry-run or live Codex smoke run.
type CodexSmokeResult struct {
	Status     string              `json:"status"`
	Runtime    string              `json:"runtime"`
	Ready      bool                `json:"ready"`
	Live       bool                `json:"live"`
	RepoRoot   string              `json:"repoRoot"`
	Kept       bool                `json:"kept"`
	Created    bool                `json:"created"`
	SmokeFile  string              `json:"smokeFile"`
	TestFile   string              `json:"testFile"`
	Init       InitResult          `json:"init"`
	Evaluation AdoptionEvalResult  `json:"evaluation"`
	Checks     []AdoptionEvalCheck `json:"checks,omitempty"`
	Warnings   []string            `json:"warnings,omitempty"`
}

// RunCodexSmoke creates a tiny repository and verifies that Codex can adopt the RunWeaver workflow contract.
func RunCodexSmoke(opts CodexSmokeOptions) (CodexSmokeResult, error) {
	root, created, cleanup, err := prepareCodexSmokeRepo(opts)
	if err != nil {
		return CodexSmokeResult{}, err
	}
	defer cleanup()
	if err := writeCodexSmokeFixture(root); err != nil {
		return CodexSmokeResult{}, err
	}
	warnings := initGitIfAvailable(root)
	initResult, err := InitSmartWithOptions(root, InitOptions{
		Force:   true,
		Runtime: RuntimeCodex,
		Classification: ClassifyOptions{
			Mode:    ClassificationDeterministic,
			Runtime: RuntimeCodex,
		},
	})
	if err != nil {
		return CodexSmokeResult{}, fmt.Errorf("initialize Codex smoke repo: %w", err)
	}
	task := `Update cmd/smoke/main.go so Message returns "after", then run go test ./... and complete the RunWeaver workflow checkpoint.`
	eval, err := EvaluateAdoption(root, AdoptionEvalOptions{
		Runtime:          RuntimeCodex,
		Task:             task,
		Live:             opts.Live,
		CodexBin:         opts.CodexBin,
		Model:            opts.Model,
		Timeout:          opts.Timeout,
		SkipGitRepoCheck: true,
	})
	if err != nil {
		return CodexSmokeResult{}, fmt.Errorf("evaluate Codex smoke adoption: %w", err)
	}
	checks := append([]AdoptionEvalCheck{}, eval.Checks...)
	if opts.Live {
		checks = append(checks, codexSmokeGoTestCheck(root))
	}
	ready := eval.Ready && adoptionEvalChecksReady(checks)
	status := "ok"
	if !ready {
		status = "warning"
	}
	return CodexSmokeResult{
		Status:     status,
		Runtime:    RuntimeCodex,
		Ready:      ready,
		Live:       opts.Live,
		RepoRoot:   filepath.ToSlash(root),
		Kept:       opts.Keep || opts.Repo != "",
		Created:    created,
		SmokeFile:  "cmd/smoke/main.go",
		TestFile:   "cmd/smoke/main_test.go",
		Init:       initResult,
		Evaluation: eval,
		Checks:     checks,
		Warnings:   warnings,
	}, nil
}

func prepareCodexSmokeRepo(opts CodexSmokeOptions) (root string, created bool, cleanup func(), err error) {
	cleanup = func() {}
	if strings.TrimSpace(opts.Repo) == "" {
		root, err = os.MkdirTemp("", "runweaver-codex-smoke-*")
		if err != nil {
			return "", false, cleanup, fmt.Errorf("create temporary Codex smoke repo: %w", err)
		}
		if !opts.Keep {
			cleanup = func() {
				_ = os.RemoveAll(root)
			}
		}
		return root, true, cleanup, nil
	}
	abs, err := filepath.Abs(opts.Repo)
	if err != nil {
		return "", false, cleanup, fmt.Errorf("resolve Codex smoke repo path %q: %w", opts.Repo, err)
	}
	if info, statErr := os.Stat(abs); statErr == nil {
		if !info.IsDir() {
			return "", false, cleanup, fmt.Errorf("Codex smoke repo path must be a directory: %s", abs)
		}
		empty, err := dirIsEmpty(abs)
		if err != nil {
			return "", false, cleanup, err
		}
		if !empty && !opts.Force {
			return "", false, cleanup, fmt.Errorf("Codex smoke repo path %s already exists and is not empty; choose a fresh path or rerun runweaver smoke codex --repo <path> --force", abs)
		}
		return abs, false, cleanup, nil
	} else if !os.IsNotExist(statErr) {
		return "", false, cleanup, fmt.Errorf("inspect Codex smoke repo path %s: %w", abs, statErr)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", false, cleanup, fmt.Errorf("create Codex smoke repo %s: %w", abs, err)
	}
	return abs, true, cleanup, nil
}

func dirIsEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, fmt.Errorf("read Codex smoke repo path %s: %w", path, err)
	}
	return len(entries) == 0, nil
}

func writeCodexSmokeFixture(root string) error {
	files := map[string]string{
		"go.mod": `module example.com/runweaver-smoke

go 1.22
`,
		"cmd/smoke/main.go": `package main

func Message() string {
	return "before"
}

func main() {
	println(Message())
}
`,
		"cmd/smoke/main_test.go": `package main

import "testing"

func TestMessage(t *testing.T) {
	if got := Message(); got != "after" {
		t.Fatalf("Message() = %q, want after", got)
	}
}
`,
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create Codex smoke fixture directory %s: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write Codex smoke fixture %s: %w", path, err)
		}
	}
	return nil
}

func initGitIfAvailable(root string) []string {
	if _, err := exec.LookPath("git"); err != nil {
		return []string{"git executable was not found; Codex smoke will use skip-git-repo-check"}
	}
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		return []string{fmt.Sprintf("git init failed: %v: %s", err, strings.TrimSpace(string(output)))}
	}
	return nil
}

func codexSmokeGoTestCheck(root string) AdoptionEvalCheck {
	if _, err := exec.LookPath("go"); err != nil {
		return AdoptionEvalCheck{
			Name:        "codex-smoke-go-test",
			Status:      "warning",
			Summary:     "Go executable was not found, so the smoke fixture could not be verified",
			NextActions: []string{"Install Go and rerun runweaver smoke codex --live."},
		}
	}
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = root
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return AdoptionEvalCheck{
			Name:     "codex-smoke-go-test",
			Status:   "error",
			Summary:  "Codex smoke fixture tests failed after live execution",
			Evidence: []string{err.Error(), strings.TrimSpace(out.String())},
		}
	}
	return AdoptionEvalCheck{
		Name:     "codex-smoke-go-test",
		Status:   "ok",
		Summary:  "Codex smoke fixture tests passed after live execution",
		Evidence: []string{strings.TrimSpace(out.String())},
	}
}
