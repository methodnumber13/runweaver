package aitools

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
)

func TestEvaluateAdoptionRunsDoctorAndStartSmoke(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/tool\n")
	writeTestFile(t, root, "cmd/tool/main.go", "package main\nfunc main() {}\n")
	if _, err := InitSmartWithOptions(root, InitOptions{
		Force:          true,
		Runtime:        RuntimeAll,
		Classification: ClassifyOptions{Mode: ClassificationDeterministic},
	}); err != nil {
		t.Fatal(err)
	}

	result, err := EvaluateAdoption(root, AdoptionEvalOptions{
		Runtime: RuntimeOpenCode,
		Task:    "Add a small CLI smoke feature",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ok" {
		t.Fatalf("eval = %#v, want ready ok", result)
	}
	if result.Start.Action != "created" {
		t.Fatalf("start action = %q, want created", result.Start.Action)
	}
	if result.Doctor.Status != "ok" {
		t.Fatalf("doctor status = %q, want ok", result.Doctor.Status)
	}
	for _, name := range []string{"first-action-contract", "start-smoke", "workflow-state", "participants-recorded", "context-returned", "runtime-dry-run"} {
		if !adoptionEvalCheckNamed(result.Checks, name) {
			t.Fatalf("eval checks = %#v, want %s", result.Checks, name)
		}
	}
	if result.ExecutionDryRun == nil || result.ExecutionDryRun.Command == nil {
		t.Fatalf("execution dry-run = %#v, want prepared runtime command", result.ExecutionDryRun)
	}
}

func TestEvaluateAdoptionLiveCodexExecutesRuntimeAndAdvancesCheckpoint(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/tool\n")
	writeTestFile(t, root, "cmd/tool/main.go", "package main\nfunc main() {}\n")
	if _, err := InitSmartWithOptions(root, InitOptions{
		Force:          true,
		Runtime:        RuntimeCodex,
		Classification: ClassifyOptions{Mode: ClassificationDeterministic},
	}); err != nil {
		t.Fatal(err)
	}
	fakeCodex := writeFakeCodexBinary(t, root)

	result, err := EvaluateAdoption(root, AdoptionEvalOptions{
		Runtime:          RuntimeCodex,
		Task:             "Add a Codex adoption smoke feature",
		Live:             true,
		CodexBin:         fakeCodex,
		Model:            "test-model",
		SkipGitRepoCheck: true,
		Timeout:          5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ok" {
		t.Fatalf("eval = %#v, want ready ok", result)
	}
	if !result.Live {
		t.Fatalf("live = false, want true")
	}
	if result.ExecutionDryRun != nil {
		t.Fatalf("execution dry-run = %#v, want empty in live mode", result.ExecutionDryRun)
	}
	if result.Execution == nil || !result.Execution.Executed || result.Execution.DryRun {
		t.Fatalf("execution = %#v, want live executed result", result.Execution)
	}
	if !adoptionEvalCheckNamed(result.Checks, "runtime-execution") {
		t.Fatalf("eval checks = %#v, want runtime-execution", result.Checks)
	}
	stdoutPath := filepath.Join(root, result.Execution.StdoutPath)
	stdout, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("read fake Codex stdout: %v", err)
	}
	if !strings.Contains(string(stdout), "fake codex completed") {
		t.Fatalf("stdout = %q, want fake Codex marker", string(stdout))
	}
	outputPath := filepath.Join(root, result.Execution.OutputPath)
	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read fake Codex final output: %v", err)
	}
	if !strings.Contains(string(output), "fake Codex final message") {
		t.Fatalf("final output = %q, want fake final message", string(output))
	}
	if result.Execution.PostCheck == nil || result.Execution.PostCheck.Status != "ok" {
		t.Fatalf("post check = %#v, want ok", result.Execution.PostCheck)
	}
}

func TestEvaluateAdoptionLivePreservesFailedExecutionResult(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/tool\n")
	writeTestFile(t, root, "cmd/tool/main.go", "package main\nfunc main() {}\n")
	if _, err := InitSmartWithOptions(root, InitOptions{
		Force:          true,
		Runtime:        RuntimeCodex,
		Classification: ClassifyOptions{Mode: ClassificationDeterministic},
	}); err != nil {
		t.Fatal(err)
	}
	failingCodex := writeFailingRuntimeBinary(t, root)

	result, err := EvaluateAdoption(root, AdoptionEvalOptions{
		Runtime:          RuntimeCodex,
		Task:             "Run a failing Codex adoption smoke",
		Live:             true,
		CodexBin:         failingCodex,
		SkipGitRepoCheck: true,
		Timeout:          5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "warning" || result.Ready {
		t.Fatalf("eval = %#v, want non-ready warning", result)
	}
	if result.Execution == nil || !result.Execution.Executed || result.Execution.Status != "error" {
		t.Fatalf("execution = %#v, want preserved failed execution result", result.Execution)
	}
	if result.Execution.ExitCode != 7 {
		t.Fatalf("exit code = %d, want 7", result.Execution.ExitCode)
	}
}

func adoptionEvalCheckNamed(checks []AdoptionEvalCheck, name string) bool {
	for _, check := range checks {
		if check.Name == name {
			return true
		}
	}
	return false
}

func writeFailingRuntimeBinary(t *testing.T, root string) string {
	t.Helper()
	name := "failing-runtime"
	if runtime.GOOS == "windows" {
		name += ".cmd"
	}
	path := filepath.Join(root, name)
	content := "#!/bin/sh\necho failing runtime >&2\nexit 7\n"
	if runtime.GOOS == "windows" {
		content = "@echo off\r\necho failing runtime 1>&2\r\nexit /b 7\r\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFakeCodexBinary(t *testing.T, root string) string {
	t.Helper()
	name := "fake-codex"
	if runtime.GOOS == "windows" {
		name += ".cmd"
	}
	path := filepath.Join(root, name)
	testBinary := strconv.Quote(os.Args[0])
	content := fmt.Sprintf("#!/bin/sh\nRUNWEAVER_FAKE_CODEX=1 %s -test.run=TestFakeCodexHelper -- \"$@\"\n", testBinary)
	if runtime.GOOS == "windows" {
		content = fmt.Sprintf("@echo off\r\nset RUNWEAVER_FAKE_CODEX=1\r\n%s -test.run=TestFakeCodexHelper -- %%*\r\n", testBinary)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFakeCodexHelper(t *testing.T) {
	if os.Getenv("RUNWEAVER_FAKE_CODEX") != "1" {
		return
	}
	if err := runFakeCodexHelper(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

func runFakeCodexHelper(args []string) error {
	runtimeArgs := argsAfterDoubleDash(args)
	root := flagValue(runtimeArgs, "-C")
	outputPath := flagValue(runtimeArgs, "--output-last-message")
	if root == "" {
		return fmt.Errorf("fake Codex did not receive -C <repo>")
	}
	var latest WorkflowLatest
	if err := ReadJSON(statepath.WorkflowLatestPath(root), &latest); err != nil {
		return fmt.Errorf("read latest workflow: %w", err)
	}
	checkpointPath := filepath.Join(root, latest.RunDir, "checkpoint.json")
	var checkpoint WorkflowCheckpoint
	if err := ReadJSON(checkpointPath, &checkpoint); err != nil {
		return fmt.Errorf("read checkpoint: %w", err)
	}
	phase := checkpoint.NextPhase
	if phase == "" {
		phase = checkpoint.CurrentPhase
	}
	if phase == "" {
		phase = "start"
	}
	checkpoint.Status = "in_progress"
	checkpoint.CurrentPhase = phase
	checkpoint.NextPhase = ""
	checkpoint.CompletedPhases = Unique(append(checkpoint.CompletedPhases, phase))
	checkpoint.LastResult = "fake Codex runtime completed " + phase
	checkpoint.NextAction = "review fake Codex result"
	checkpoint.NextVerification = "run runweaver workflow verify --repo . --resume latest"
	checkpoint.UpdatedAt = Now()
	if err := WriteJSON(checkpointPath, checkpoint); err != nil {
		return fmt.Errorf("write checkpoint: %w", err)
	}
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte("fake Codex final message\n"), 0o644); err != nil {
			return fmt.Errorf("write final message: %w", err)
		}
	}
	fmt.Println(`{"type":"message","content":"fake codex completed"}`)
	return nil
}

func argsAfterDoubleDash(args []string) []string {
	for index, arg := range args {
		if arg == "--" {
			return args[index+1:]
		}
	}
	return args
}

func flagValue(args []string, flag string) string {
	for index, arg := range args {
		if arg == flag && index+1 < len(args) {
			return args[index+1]
		}
	}
	return ""
}
