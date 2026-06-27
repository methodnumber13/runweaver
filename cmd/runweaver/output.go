package main

import (
	"encoding/json"
	"fmt"
)

func (c cli) printJSON(value any) error {
	encoder := json.NewEncoder(c.stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("write JSON output: %w", err)
	}
	return nil
}

func (c cli) printError(err error) {
	command := "help"
	showHint := false
	if usage, ok := err.(usageError); ok {
		command = usage.command
		showHint = true
	}
	if commandErr, ok := err.(commandError); ok {
		command = commandErr.command
	}
	fmt.Fprintf(c.stderr, "%s: %v\n", c.paint(ansiRed, "error"), err)
	if showHint {
		if hint := commandHint(command); hint != "" {
			fmt.Fprintf(c.stderr, "%s: %s\n", c.paint(ansiYellow, "hint"), hint)
		}
	}
}

func (c cli) printUsageHint(command string) {
	if hint := commandHint(command); hint != "" {
		fmt.Fprintf(c.stderr, "%s: %s\n", c.paint(ansiYellow, "hint"), hint)
	}
}

func (c cli) printStatus(kind, message string) {
	if c.stderr == nil || message == "" {
		return
	}
	label := "info"
	color := ansiCyan
	switch kind {
	case "success", "ok":
		label = "ok"
		color = ansiGreen
	case "warning":
		label = "warning"
		color = ansiYellow
	case "error":
		label = "error"
		color = ansiRed
	}
	fmt.Fprintf(c.stderr, "%s: %s\n", c.paint(color, label), message)
}
