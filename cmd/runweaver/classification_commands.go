package main

import (
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools"
)

func (c cli) classifyCmd(args []string) error {
	fs := newFlagSet("classify")
	repo := fs.String("repo", ".", "repository path")
	apply := fs.Bool("apply", false, "materialize generated profile, agents, and skills into .opencode")
	runtimeProvider := fs.String("runtime", aitools.RuntimeOpenCode, "runtime provider to materialize when --apply is set: opencode, codex, claude, all, or comma-separated list")
	classification := addClassificationFlags(fs, "auto")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "classify", err: err}
	}
	if err := rejectExtraArgs(fs, "classify"); err != nil {
		return err
	}
	opts, err := classification.options()
	if err != nil {
		return usageError{command: "classify", err: err}
	}
	opts.Apply = *apply
	opts.ApplyRuntime = *runtimeProvider
	result, err := aitools.ClassifyRepository(*repo, opts)
	if err != nil {
		return fmt.Errorf("classify repository: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	status := "success"
	if result.Classifier.UsedFallback || len(result.Warnings) > 0 {
		status = "warning"
	}
	c.printStatus(status, fmt.Sprintf("repository classified via %s", result.Classifier.Source))
	return nil
}

func (c cli) refreshCmd(args []string) error {
	fs := newFlagSet("refresh")
	repo := fs.String("repo", ".", "repository path")
	apply := fs.Bool("apply", false, "write generated profile to selected runtime metadata instead of .runweaver/tmp/profile.generated.json")
	runtimeProvider := fs.String("runtime", aitools.RuntimeOpenCode, "runtime provider to materialize when --apply is set: opencode, codex, claude, all, or comma-separated list")
	classification := addClassificationFlags(fs, "auto")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "refresh", err: err}
	}
	if err := rejectExtraArgs(fs, "refresh"); err != nil {
		return err
	}
	classifyOptions, err := classification.options()
	if err != nil {
		return usageError{command: "refresh", err: err}
	}
	result, err := aitools.RefreshWithOptions(*repo, aitools.RefreshOptions{Apply: *apply, Runtime: *runtimeProvider, Classification: classifyOptions})
	if err != nil {
		return fmt.Errorf("refresh runtime metadata: %w", err)
	}
	status := "success"
	if len(result.DriftReport.StaleAnchors) > 0 || len(result.DriftReport.MissingSurfaces) > 0 {
		status = "warning"
	}
	if err := c.printJSON(map[string]any{
		"surfaceIndexPath": result.SurfaceIndexPath,
		"driftReportPath":  result.DriftReportPath,
		"profilePath":      result.ProfilePath,
		"staleAnchors":     len(result.DriftReport.StaleAnchors),
		"missingSurfaces":  len(result.DriftReport.MissingSurfaces),
		"recommendations":  result.DriftReport.Recommendations,
		"classifier":       result.Classifier,
	}); err != nil {
		return err
	}
	c.printStatus(status, "metadata refresh complete")
	return nil
}
