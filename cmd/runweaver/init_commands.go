package main

import (
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools"
)

func (c cli) initCmd(args []string) error {
	fs := newFlagSet("init")
	repo := fs.String("repo", ".", "repository path")
	force := fs.Bool("force", false, "overwrite existing generated runtime files")
	runtimeProvider := fs.String("runtime", aitools.RuntimeOpenCode, "runtime provider: opencode, codex, claude, all, or comma-separated list")
	requireModel := fs.Bool("require-model", false, "fail if OpenCode model/provider preflight is not ready")
	provider := fs.String("provider", "", "OpenCode provider id for model preflight; defaults to the provider prefix in the configured model")
	model := fs.String("model", "", "expected model id for model preflight, without provider prefix")
	baseURL := fs.String("base-url", "", "expected OpenAI-compatible base URL for model preflight")
	classification := addClassificationFlags(fs, "auto")
	if err := fs.Parse(args); err != nil {
		return usageError{command: "init", err: err}
	}
	if err := rejectExtraArgs(fs, "init"); err != nil {
		return err
	}
	classifyOptions, err := classification.options()
	if err != nil {
		return usageError{command: "init", err: err}
	}
	if classifyOptions.ProviderID == "" {
		classifyOptions.ProviderID = *provider
	}
	if classifyOptions.Model == "" {
		classifyOptions.Model = *model
	}
	progress, stopProgress := c.initProgressReporter()
	defer stopProgress()
	result, err := aitools.InitSmartWithOptions(*repo, aitools.InitOptions{
		Force:        *force,
		RequireModel: *requireModel,
		Runtime:      *runtimeProvider,
		ModelCheck: aitools.ModelConfigCheckOptions{
			ProviderID: *provider,
			ModelID:    *model,
			BaseURL:    *baseURL,
		},
		Classification: classifyOptions,
		Progress:       progress,
	})
	if err != nil {
		return fmt.Errorf("initialize RunWeaver metadata: %w", err)
	}
	if err := c.printJSON(result); err != nil {
		return err
	}
	if result.ModelPreflight.Ready {
		c.printStatus("success", "initialized repository and model preflight is ready")
	} else if result.ModelPreflight.Status == "skipped" {
		c.printStatus("success", "initialized repository; OpenCode model preflight was skipped")
	} else {
		c.printStatus("warning", "initialized repository; model preflight is not ready")
	}
	return nil
}
