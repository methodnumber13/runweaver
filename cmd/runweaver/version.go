package main

import (
	"fmt"
	"runtime/debug"
	"strings"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Module  string `json:"module,omitempty"`
	Dirty   bool   `json:"dirty,omitempty"`
}

func (c cli) versionCmd(args []string) error {
	fs := newFlagSet("version")
	jsonOutput := fs.Bool("json", false, "print version metadata as JSON")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "version", err: err}
	}
	if err := rejectExtraArgs(fs, "version"); err != nil {
		return err
	}
	info := resolveVersionInfo()
	if *jsonOutput {
		return c.printJSON(info)
	}
	fmt.Fprintf(c.stdout, "runweaver %s\n", info.Version)
	fmt.Fprintf(c.stdout, "commit: %s\n", info.Commit)
	fmt.Fprintf(c.stdout, "date: %s\n", info.Date)
	if info.Module != "" {
		fmt.Fprintf(c.stdout, "module: %s\n", info.Module)
	}
	if info.Dirty {
		fmt.Fprintln(c.stdout, "dirty: true")
	}
	return nil
}

func resolveVersionInfo() versionInfo {
	info := versionInfo{
		Version: cleanBuildValue(version, "dev"),
		Commit:  cleanBuildValue(commit, "unknown"),
		Date:    cleanBuildValue(buildDate, "unknown"),
	}
	build, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}
	info.Module = strings.TrimSpace(build.Main.Path)
	if info.Version == "dev" && build.Main.Version != "" && build.Main.Version != "(devel)" {
		info.Version = build.Main.Version
	}
	for _, setting := range build.Settings {
		switch setting.Key {
		case "vcs.revision":
			if info.Commit == "unknown" {
				info.Commit = cleanBuildValue(setting.Value, "unknown")
			}
		case "vcs.time":
			if info.Date == "unknown" {
				info.Date = cleanBuildValue(setting.Value, "unknown")
			}
		case "vcs.modified":
			info.Dirty = setting.Value == "true"
		}
	}
	return info
}

func cleanBuildValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
