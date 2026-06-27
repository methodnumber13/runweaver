package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/methodnumber13/runweaver/internal/aitools"
)

func (c cli) initProgressReporter() (aitools.InitProgressReporter, func()) {
	if c.stderr == nil {
		return func(aitools.InitProgressEvent) {}, func() {}
	}
	if !c.color {
		return func(event aitools.InitProgressEvent) {
			c.printProgress(event, "")
		}, func() {}
	}
	updates := make(chan aitools.InitProgressEvent, 16)
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		frames := []string{"|", "/", "-", "\\"}
		frame := 0
		current := aitools.InitProgressEvent{Current: 0, Total: 1, Step: "starting", Message: "Starting init"}
		rendered := false
		startedAt := time.Now()
		currentKey := current.Step
		for {
			select {
			case event := <-updates:
				if event.Total > 0 {
					key := fmt.Sprintf("%d/%d/%s", event.Current, event.Total, event.Step)
					if key != currentKey {
						startedAt = time.Now()
						currentKey = key
					}
					current = event
				}
			case <-ticker.C:
				frame++
			case <-done:
				if rendered {
					fmt.Fprint(c.stderr, "\n")
				}
				return
			}
			display := current
			display.Elapsed = time.Since(startedAt)
			display.Pulse = frame
			c.printProgress(display, frames[frame%len(frames)])
			rendered = true
		}
	}()
	reporter := func(event aitools.InitProgressEvent) {
		select {
		case updates <- event:
		default:
		}
	}
	stop := func() {
		close(done)
		<-finished
	}
	return reporter, stop
}

func (c cli) printProgress(event aitools.InitProgressEvent, spinner string) {
	if c.stderr == nil || event.Total <= 0 {
		return
	}
	current := event.Current
	if current < 0 {
		current = 0
	}
	if current > event.Total {
		current = event.Total
	}
	bar := progressBar(current, event.Total, 20)
	if spinner != "" {
		bar = progressBarAnimated(current, event.Total, 20, event.Pulse)
	}
	label := "init"
	step := event.Step
	if step != "" {
		step = " " + step
	}
	if spinner != "" && event.Elapsed >= time.Second {
		step += " " + formatProgressElapsed(event.Elapsed)
	}
	message := event.Message
	if message == "" {
		message = "working"
	}
	if spinner != "" {
		line := fmt.Sprintf("%s [%s] %d/%d%s: %s", label, bar, current, event.Total, step, message)
		line = truncateProgressLine(line, progressLineLimit()-2)
		fmt.Fprintf(c.stderr, "\r\x1b[2K%s %s", c.paint(ansiYellow, spinner), line)
		return
	}
	if c.color {
		label = c.paint(ansiCyan, label)
	}
	fmt.Fprintf(c.stderr, "%s [%s] %d/%d%s: %s\n", label, bar, current, event.Total, step, message)
}

func progressBar(current, total, width int) string {
	if total <= 0 {
		total = 1
	}
	if width <= 0 {
		width = 20
	}
	if current < 0 {
		current = 0
	}
	if current > total {
		current = total
	}
	filled := current * width / total
	if filled > width {
		filled = width
	}
	return strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
}

func progressBarAnimated(current, total, width, pulse int) string {
	bar := []byte(progressBar(current, total, width))
	if len(bar) == 0 || current >= total {
		return string(bar)
	}
	filled := current * width / total
	if filled < 0 {
		filled = 0
	}
	if filled >= width {
		filled = width - 1
	}
	span := width - filled
	if span <= 0 {
		return string(bar)
	}
	bar[filled+(pulse%span)] = '>'
	return string(bar)
}

func formatProgressElapsed(elapsed time.Duration) string {
	if elapsed < 0 {
		elapsed = 0
	}
	total := int(elapsed.Round(time.Second).Seconds())
	minutes := total / 60
	seconds := total % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func progressLineLimit() int {
	raw := strings.TrimSpace(os.Getenv("COLUMNS"))
	if raw == "" {
		return 100
	}
	width, err := strconv.Atoi(raw)
	if err != nil || width < 40 {
		return 100
	}
	return width - 1
}

func truncateProgressLine(line string, limit int) string {
	if limit <= 0 || len(line) <= limit {
		return line
	}
	if limit <= 3 {
		return line[:limit]
	}
	return strings.TrimRight(line[:limit-3], " ") + "..."
}
