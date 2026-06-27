package main

import (
	"bytes"
	"github.com/methodnumber13/runweaver/internal/aitools"
	"strings"
	"testing"
	"time"
)

func TestProgressBarAnimatedKeepsStableWidth(t *testing.T) {
	bar := progressBarAnimated(2, 8, 12, 3)

	if len(bar) != 12 {
		t.Fatalf("len(progress bar) = %d, want 12: %q", len(bar), bar)
	}
	if !strings.Contains(bar, ">") {
		t.Fatalf("progress bar = %q, want animated marker", bar)
	}
}

func TestProgressFormattingIsBoundedAndReadable(t *testing.T) {
	if got := formatProgressElapsed(65 * time.Second); got != "01:05" {
		t.Fatalf("elapsed = %q, want 01:05", got)
	}
	if got := formatProgressElapsed(-1 * time.Second); got != "00:00" {
		t.Fatalf("negative elapsed = %q, want 00:00", got)
	}
	if got := truncateProgressLine("0123456789", 7); got != "0123..." {
		t.Fatalf("truncated = %q, want 0123...", got)
	}
	if got := truncateProgressLine("0123456789", 2); got != "01" {
		t.Fatalf("small truncated = %q, want 01", got)
	}
	t.Setenv("COLUMNS", "60")
	if got := progressLineLimit(); got != 59 {
		t.Fatalf("progressLineLimit = %d, want 59", got)
	}
	t.Setenv("COLUMNS", "bad")
	if got := progressLineLimit(); got != 100 {
		t.Fatalf("progressLineLimit invalid = %d, want 100", got)
	}
}

func TestProgressAndStatusOutputBranches(t *testing.T) {
	var stderr bytes.Buffer
	c := cli{stderr: &stderr, color: true}

	c.printProgress(aitoolsInitProgressEvent(4, 2, "verify", "done"), "")
	if !strings.Contains(stderr.String(), "2/2 verify: done") {
		t.Fatalf("progress output = %q, want clamped progress", stderr.String())
	}
	stderr.Reset()

	c.printProgress(aitoolsInitProgressEvent(1, 3, "classify", "running"), "|")
	if !strings.Contains(stderr.String(), "\r") || !strings.Contains(stderr.String(), "classify") {
		t.Fatalf("animated progress output = %q, want carriage return classify line", stderr.String())
	}
	stderr.Reset()

	for _, kind := range []string{"success", "warning", "error", "info"} {
		c.printStatus(kind, "message")
	}
	output := stderr.String()
	for _, want := range []string{"ok", "warning", "error", "info"} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output = %q, want %q", output, want)
		}
	}
}

func aitoolsInitProgressEvent(current, total int, step, message string) aitools.InitProgressEvent {
	return aitools.InitProgressEvent{Current: current, Total: total, Step: step, Message: message}
}

func TestInitProgressReporterModes(t *testing.T) {
	reporter, stop := (cli{}).initProgressReporter()
	reporter(aitoolsInitProgressEvent(1, 1, "noop", "ignored"))
	stop()

	var plain bytes.Buffer
	plainCLI := cli{stderr: &plain, color: false}
	reporter, stop = plainCLI.initProgressReporter()
	reporter(aitoolsInitProgressEvent(1, 2, "scan", "Scanning"))
	stop()
	if !strings.Contains(plain.String(), "1/2 scan: Scanning") {
		t.Fatalf("plain progress = %q, want scan line", plain.String())
	}

	var animated bytes.Buffer
	animatedCLI := cli{stderr: &animated, color: true}
	reporter, stop = animatedCLI.initProgressReporter()
	reporter(aitoolsInitProgressEvent(1, 2, "classify", "Classifying"))
	time.Sleep(140 * time.Millisecond)
	stop()
	if !strings.Contains(animated.String(), "classify") || !strings.Contains(animated.String(), "\n") {
		t.Fatalf("animated progress = %q, want rendered line and newline on stop", animated.String())
	}
}
