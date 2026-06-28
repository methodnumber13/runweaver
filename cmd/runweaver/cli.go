package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	ansiReset  = "\x1b[0m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
)

type cli struct {
	stdout io.Writer
	stderr io.Writer
	color  bool
}

type usageError struct {
	command string
	err     error
}

func (e usageError) Error() string {
	return e.err.Error()
}

func (e usageError) Unwrap() error {
	return e.err
}

type commandError struct {
	command string
	err     error
}

func (e commandError) Error() string {
	return e.err.Error()
}

func (e commandError) Unwrap() error {
	return e.err
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr, colorEnabled(os.Stderr)))
}

func runCLI(args []string, stdout, stderr io.Writer, color bool) (code int) {
	app := cli{stdout: stdout, stderr: stderr, color: color}
	defer func() {
		if recovered := recover(); recovered != nil {
			app.printError(fmt.Errorf("internal error: %v", recovered))
			code = 1
		}
	}()
	if err := app.run(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			app.commandUsage(commandFromError(err))
			return 0
		}
		app.printError(err)
		return 1
	}
	return 0
}

func (c cli) run(args []string) error {
	if len(args) == 0 {
		c.usage()
		return nil
	}
	switch args[0] {
	case "scan":
		return wrapCommandError("scan", c.scanCmd(args[1:]))
	case "index":
		return wrapCommandError("index", c.indexCmd(args[1:]))
	case "classify":
		return wrapCommandError("classify", c.classifyCmd(args[1:]))
	case "refresh":
		return wrapCommandError("refresh", c.refreshCmd(args[1:]))
	case "status":
		return wrapCommandError("status", c.statusCmd(args[1:]))
	case "doctor":
		return wrapCommandError("doctor", c.doctorCmd(args[1:]))
	case "init":
		return wrapCommandError("init", c.initCmd(args[1:]))
	case "bootstrap":
		return wrapCommandError("bootstrap", c.initCmd(args[1:]))
	case "workflow":
		return wrapCommandError("workflow run", c.workflowCmd(args[1:]))
	case "help", "-h", "--help":
		c.usage()
		return nil
	default:
		return usageError{command: "help", err: fmt.Errorf("unknown command %q", args[0])}
	}
}

func wrapCommandError(command string, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(usageError); ok {
		return err
	}
	if _, ok := err.(commandError); ok {
		return err
	}
	return commandError{command: command, err: err}
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func rejectExtraArgs(fs *flag.FlagSet, command string) error {
	if fs.NArg() == 0 {
		return nil
	}
	return usageError{command: command, err: fmt.Errorf("unexpected argument(s): %s", strings.Join(fs.Args(), " "))}
}

func commandFromError(err error) string {
	var usage usageError
	if errors.As(err, &usage) {
		return usage.command
	}
	var command commandError
	if errors.As(err, &command) {
		return command.command
	}
	return "help"
}

func (c cli) paint(color, text string) string {
	if !c.color {
		return text
	}
	return color + text + ansiReset
}

func colorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
