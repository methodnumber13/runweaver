package main

import (
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools"
	"time"
)

func (c cli) doctorCmd(args []string) error {
	if len(args) > 0 && args[0] == "model" {
		return c.doctorModelCmd(args[1:])
	}
	if len(args) > 0 && args[0] == "opencode" {
		return c.doctorOpenCodeCmd(args[1:])
	}
	if len(args) > 0 && args[0] == "runtime" {
		return c.doctorRuntimeCmd(args[1:])
	}
	if len(args) > 0 && args[0] == "adoption" {
		return c.doctorAdoptionCmd(args[1:])
	}
	if len(args) > 0 && args[0] == "processes" {
		return c.doctorProcessesCmd(args[1:])
	}
	fs := newFlagSet("doctor")
	repo := fs.String("repo", ".", "repository path")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "doctor", err: err}
	}
	if err := rejectExtraArgs(fs, "doctor"); err != nil {
		return err
	}
	result, err := aitools.Refresh(*repo, false)
	if err != nil {
		return fmt.Errorf("run doctor: %w", err)
	}
	status := "ok"
	statusKind := "success"
	if len(result.DriftReport.StaleAnchors) > 0 || len(result.DriftReport.MissingSurfaces) > 0 {
		status = "warning"
		statusKind = "warning"
	}
	if err := c.printJSON(map[string]any{
		"status":           status,
		"surfaceIndexPath": result.SurfaceIndexPath,
		"driftReportPath":  result.DriftReportPath,
		"staleAnchors":     len(result.DriftReport.StaleAnchors),
		"missingSurfaces":  len(result.DriftReport.MissingSurfaces),
	}); err != nil {
		return err
	}
	c.printStatus(statusKind, "doctor complete")
	return nil
}

func (c cli) doctorRuntimeCmd(args []string) error {
	fs := newFlagSet("doctor runtime")
	repo := fs.String("repo", ".", "repository path")
	runtimeProvider := fs.String("runtime", aitools.RuntimeAll, "runtime provider: opencode, codex, claude, all, or comma-separated list")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "doctor runtime", err: err}
	}
	if err := rejectExtraArgs(fs, "doctor runtime"); err != nil {
		return err
	}
	results, err := aitools.DiscoverRuntimes(*repo, aitools.RuntimeDiscoveryOptions{Runtime: *runtimeProvider})
	if err != nil {
		return fmt.Errorf("discover runtime providers: %w", err)
	}
	ready := len(results) > 0
	for _, result := range results {
		if !result.Ready {
			ready = false
			break
		}
	}
	status := "ok"
	statusKind := "success"
	if !ready {
		status = "warning"
		statusKind = "warning"
	}
	if err := c.printJSON(map[string]any{
		"status":   status,
		"ready":    ready,
		"runtime":  *runtimeProvider,
		"runtimes": results,
	}); err != nil {
		return err
	}
	c.printStatus(statusKind, "runtime provider discovery complete")
	return nil
}

func (c cli) doctorAdoptionCmd(args []string) error {
	fs := newFlagSet("doctor adoption")
	repo := fs.String("repo", ".", "repository path")
	runtimeProvider := fs.String("runtime", aitools.RuntimeAll, "runtime provider: opencode, codex, claude, all, or comma-separated list")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "doctor adoption", err: err}
	}
	if err := rejectExtraArgs(fs, "doctor adoption"); err != nil {
		return err
	}
	result, err := aitools.DoctorAdoption(*repo, aitools.AdoptionDoctorOptions{Runtime: *runtimeProvider})
	if err != nil {
		return fmt.Errorf("check RunWeaver adoption: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	if result.Ready {
		c.printStatus("success", "RunWeaver adoption contract is ready")
	} else {
		c.printStatus("warning", "RunWeaver adoption contract has issues; see JSON checks")
	}
	return nil
}

func (c cli) doctorProcessesCmd(args []string) error {
	fs := newFlagSet("doctor processes")
	summaryOnly := fs.Bool("summary", false, "print only counts and duplicate process groups")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "doctor processes", err: err}
	}
	if err := rejectExtraArgs(fs, "doctor processes"); err != nil {
		return err
	}
	result, err := aitools.DoctorProcesses()
	if err != nil {
		return fmt.Errorf("check runtime processes: %w", err)
	}
	if *summaryOnly {
		duplicates := make([]map[string]any, 0, len(result.Duplicates))
		for _, duplicate := range result.Duplicates {
			duplicates = append(duplicates, map[string]any{
				"kind":    duplicate.Kind,
				"count":   duplicate.Count,
				"command": duplicate.Command,
			})
		}
		if err := c.printJSON(map[string]any{
			"status":          result.Status,
			"summary":         result.Summary,
			"supervisorCount": len(result.Supervisors),
			"duplicateGroups": duplicates,
			"vscode": map[string]any{
				"detected":                 result.VSCode.Detected,
				"helperProcessCount":       result.VSCode.HelperProcessCount,
				"debuggerProcessCount":     result.VSCode.DebuggerProcessCount,
				"nodeLikeProcessCount":     result.VSCode.NodeLikeProcessCount,
				"settingsPath":             result.VSCode.SettingsPath,
				"hasAutoAttachSetting":     result.VSCode.HasAutoAttachSetting,
				"autoAttachFilter":         result.VSCode.AutoAttachFilter,
				"autoAttachRecommendation": result.VSCode.AutoAttachRecommendation,
			},
			"recommendations": result.Recommendations,
		}); err != nil {
			return err
		}
	} else {
		if err := c.printJSON(result); err != nil {
			return err
		}
	}
	statusKind := "success"
	if result.Status != "ok" {
		statusKind = "warning"
	}
	c.printStatus(statusKind, "process doctor complete")
	return nil
}

func (c cli) doctorOpenCodeCmd(args []string) error {
	fs := newFlagSet("doctor opencode")
	repo := fs.String("repo", ".", "repository path")
	opencodeBin := fs.String("opencode-bin", "opencode", "OpenCode executable path")
	agent := fs.String("agent", aitools.OpenCodePrimaryAgentName, "OpenCode primary agent name")
	provider := fs.String("provider", "", "OpenCode provider id for model preflight; defaults to the provider prefix in the configured model")
	skipModelCheck := fs.Bool("skip-model-check", false, "skip OpenCode model preflight")
	timeout := fs.Duration("timeout", 45*time.Second, "per OpenCode command timeout, for example 60s or 2m")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "doctor opencode", err: err}
	}
	if err := rejectExtraArgs(fs, "doctor opencode"); err != nil {
		return err
	}
	result, err := aitools.DoctorOpenCode(*repo, aitools.OpenCodeDoctorOptions{
		OpencodeBin:    *opencodeBin,
		Agent:          *agent,
		ProviderID:     *provider,
		SkipModelCheck: *skipModelCheck,
		Timeout:        *timeout,
	})
	if err != nil {
		return fmt.Errorf("check OpenCode readiness: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	if result.Ready {
		c.printStatus("success", "OpenCode RunWeaver readiness check passed")
	} else {
		c.printStatus("warning", "OpenCode RunWeaver readiness has issues; see JSON checks")
	}
	return nil
}

func (c cli) doctorModelCmd(args []string) error {
	fs := newFlagSet("doctor model")
	repo := fs.String("repo", ".", "repository path")
	provider := fs.String("provider", "", "OpenCode provider id; defaults to the provider prefix in the configured model")
	model := fs.String("model", "", "expected model id, without provider prefix")
	baseURL := fs.String("base-url", "", "expected OpenAI-compatible base URL")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "doctor model", err: err}
	}
	if err := rejectExtraArgs(fs, "doctor model"); err != nil {
		return err
	}
	result, err := aitools.CheckModelConfig(*repo, aitools.ModelConfigCheckOptions{
		ProviderID: *provider,
		ModelID:    *model,
		BaseURL:    *baseURL,
	})
	if err != nil {
		return fmt.Errorf("check OpenCode model config: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	if result.Ready {
		c.printStatus("success", "OpenCode model preflight is ready")
	} else {
		c.printStatus("warning", "OpenCode model preflight has issues; see JSON issues")
	}
	return nil
}
