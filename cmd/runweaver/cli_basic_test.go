package main

import (
	"bytes"
	"errors"
	"flag"
	"github.com/methodnumber13/runweaver/internal/aitools"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLIPrintsColoredErrorForUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"nope"}, &stdout, &stderr, true)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	text := stderr.String()
	if !strings.Contains(text, "\x1b[31merror\x1b[0m") {
		t.Fatalf("stderr = %q, want colored error label", text)
	}
	if !strings.Contains(text, "unknown command \"nope\"") {
		t.Fatalf("stderr = %q, want unknown command detail", text)
	}
	if !strings.Contains(text, "runweaver help") {
		t.Fatalf("stderr = %q, want help hint", text)
	}
}

func TestCLIPrintsPlainErrorWhenColorDisabled(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"index", "--repo", filepath.Join(t.TempDir(), "missing")}, &stdout, &stderr, false)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if strings.Contains(stderr.String(), "\x1b[") {
		t.Fatalf("stderr = %q, want no ANSI escapes", stderr.String())
	}
	if !strings.Contains(stderr.String(), "repository path") {
		t.Fatalf("stderr = %q, want contextual repo error", stderr.String())
	}
}

func TestCLISuppressesRawFlagUsageAndPrintsActionableError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"index", "--bad-flag"}, &stdout, &stderr, true)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	text := stderr.String()
	if strings.Contains(text, "Usage of index:") {
		t.Fatalf("stderr = %q, want custom usage instead of raw flag package output", text)
	}
	if !strings.Contains(text, "flag provided but not defined: -bad-flag") {
		t.Fatalf("stderr = %q, want flag parse error", text)
	}
	if !strings.Contains(text, "runweaver index --repo <path>") {
		t.Fatalf("stderr = %q, want command hint", text)
	}
}

func TestCLIHelpPrintsCommands(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"help"}, &stdout, &stderr, false)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "runweaver commands") || !strings.Contains(stdout.String(), "doctor opencode") {
		t.Fatalf("stdout = %q, want command list", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestCLIPublicHelpDoesNotExposePrivateVendorNames(t *testing.T) {
	for _, args := range [][]string{
		{"help"},
		{"classify", "--help"},
		{"doctor", "model", "--help"},
		{"workflow", "run", "--help"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := runCLI(args, &stdout, &stderr, false)

			if code != 0 {
				t.Fatalf("exit code = %d stderr=%q", code, stderr.String())
			}
			output := strings.ToLower(stdout.String() + stderr.String())
			for _, forbidden := range privateVendorTermsForHelpTest() {
				if strings.Contains(output, forbidden) {
					t.Fatalf("help output for %v contains private/vendor term %q:\n%s", args, forbidden, output)
				}
			}
		})
	}
}

func privateVendorTermsForHelpTest() []string {
	return []string{
		"gpus" + "tack",
		"key" + "cloak",
		"git" + "lab",
		"yan" + "dex",
		"sen" + "try",
		"tech" + "gate",
		"lm" + "ru",
		"i" + "dp",
	}
}

func TestClassificationFlagsOptionsUseRuntimeNeutralDefaults(t *testing.T) {
	fs := newFlagSet("classification")
	flags := addClassificationFlags(fs, "auto")

	err := fs.Parse([]string{
		"--classifier", "ai",
		"--classifier-runtime", "codex",
		"--classifier-model", "coder-model",
		"--classifier-codex-bin", "codex-test",
		"--classifier-timeout", "3m",
		"--classifier-skip-runtime-check",
	})
	if err != nil {
		t.Fatalf("parse classification flags: %v", err)
	}

	opts, err := flags.options()
	if err != nil {
		t.Fatalf("classification options: %v", err)
	}
	if opts.Mode != aitools.ClassificationAI {
		t.Fatalf("Mode = %q, want %q", opts.Mode, aitools.ClassificationAI)
	}
	if opts.ProviderID != "" {
		t.Fatalf("ProviderID = %q, want empty auto-detect default", opts.ProviderID)
	}
	if opts.Runtime != "codex" || opts.Model != "coder-model" || opts.CodexBin != "codex-test" {
		t.Fatalf("runtime/model/bin = %q/%q/%q, want codex/coder-model/codex-test", opts.Runtime, opts.Model, opts.CodexBin)
	}
	if opts.Timeout != 3*time.Minute {
		t.Fatalf("Timeout = %s, want 3m", opts.Timeout)
	}
	if !opts.SkipRuntimeCheck {
		t.Fatalf("SkipRuntimeCheck = false, want true")
	}
}

func TestCLISubcommandHelpPrintsUsageWithoutError(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{name: "classify", args: []string{"classify", "--help"}, want: "--classifier-timeout duration"},
		{name: "doctor-runtime", args: []string{"doctor", "runtime", "--help"}, want: "runweaver doctor runtime"},
		{name: "doctor-processes", args: []string{"doctor", "processes", "--help"}, want: "runweaver doctor processes [--summary]"},
		{name: "workflow-run", args: []string{"workflow", "run", "--help"}, want: "--execute"},
		{name: "workflow-verify", args: []string{"workflow", "verify", "--help"}, want: "runweaver workflow verify"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := runCLI(test.args, &stdout, &stderr, false)

			if code != 0 {
				t.Fatalf("exit code = %d stderr=%q", code, stderr.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			if !strings.Contains(stdout.String(), test.want) {
				t.Fatalf("stdout = %q, want %q", stdout.String(), test.want)
			}
		})
	}
}

func TestCommandHintsAndColorEnabledFallbacks(t *testing.T) {
	for _, command := range []string{"scan", "index", "index clean", "refresh", "doctor", "doctor model", "doctor opencode", "doctor runtime", "doctor processes", "init", "workflow run", "missing"} {
		if hint := commandHint(command); hint == "" {
			t.Fatalf("commandHint(%q) returned empty", command)
		}
	}
	var buffer bytes.Buffer
	if colorEnabled(&buffer) {
		t.Fatal("colorEnabled returned true for non-file writer")
	}
	t.Setenv("NO_COLOR", "1")
	if colorEnabled(&buffer) {
		t.Fatal("colorEnabled returned true when NO_COLOR is set")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if colorEnabled(&buffer) {
		t.Fatal("colorEnabled returned true when TERM=dumb")
	}
}

func TestCommandUsageCoversEveryPublicCommand(t *testing.T) {
	for _, command := range []string{
		"scan",
		"index",
		"index clean",
		"refresh",
		"status",
		"classify",
		"init",
		"bootstrap",
		"doctor",
		"doctor model",
		"doctor opencode",
		"doctor runtime",
		"doctor processes",
		"workflow run",
		"workflow update",
		"workflow verify",
	} {
		if usage := commandUsage(command); !strings.Contains(usage, "runweaver") {
			t.Fatalf("commandUsage(%q) = %q, want runweaver usage text", command, usage)
		}
	}
}

func TestCommandErrorHelpers(t *testing.T) {
	if err := rejectExtraArgs(newFlagSet("empty"), "scan"); err != nil {
		t.Fatalf("rejectExtraArgs(empty) = %v, want nil", err)
	}
	fs := newFlagSet("extra")
	if err := fs.Parse([]string{"unexpected"}); err != nil {
		t.Fatal(err)
	}
	if err := rejectExtraArgs(fs, "scan"); err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("rejectExtraArgs(extra) = %v, want unexpected argument error", err)
	}
	if got := commandFromError(usageError{command: "scan", err: flag.ErrHelp}); got != "scan" {
		t.Fatalf("commandFromError(usage) = %q, want scan", got)
	}
	if got := commandFromError(commandError{command: "doctor", err: errors.New("failed")}); got != "doctor" {
		t.Fatalf("commandFromError(command) = %q, want doctor", got)
	}
	if got := commandFromError(errors.New("plain")); got != "help" {
		t.Fatalf("commandFromError(plain) = %q, want help", got)
	}
	var stderr bytes.Buffer
	cli{stderr: &stderr}.printUsageHint("scan")
	if !strings.Contains(stderr.String(), "runweaver scan") {
		t.Fatalf("usage hint = %q, want scan hint", stderr.String())
	}
}
