package aitools

import (
	"fmt"
	"strings"
	"time"
)

func normalizeClassifyOptions(opts ClassifyOptions, defaultMode ClassificationMode) (ClassifyOptions, error) {
	if defaultMode == "" {
		defaultMode = ClassificationDeterministic
	}
	mode := strings.TrimSpace(strings.ToLower(string(opts.Mode)))
	if mode == "" {
		mode = string(defaultMode)
	}
	switch ClassificationMode(mode) {
	case ClassificationAuto, ClassificationAI, ClassificationDeterministic:
		opts.Mode = ClassificationMode(mode)
	default:
		return opts, fmt.Errorf("unsupported classification mode %q; expected auto, ai, or deterministic", opts.Mode)
	}
	if opts.Runtime == "" {
		opts.Runtime = RuntimeOpenCode
	}
	opts.Runtime = normalizeRuntimeID(opts.Runtime)
	if _, ok := RuntimeProviderByID(opts.Runtime); !ok {
		return opts, fmt.Errorf("unsupported classifier runtime %q; supported: opencode, codex, claude", opts.Runtime)
	}
	if opts.OpencodeBin == "" {
		opts.OpencodeBin = "opencode"
	}
	if opts.CodexBin == "" {
		opts.CodexBin = "codex"
	}
	if opts.ClaudeBin == "" {
		opts.ClaudeBin = "claude"
	}
	if opts.Agent == "" {
		opts.Agent = "repo-classifier"
	}
	if opts.Sandbox == "" {
		opts.Sandbox = "read-only"
	}
	if opts.ApprovalPolicy == "" {
		opts.ApprovalPolicy = "never"
	}
	if opts.PermissionMode == "" {
		opts.PermissionMode = "dontAsk"
	}
	if opts.Timeout == 0 {
		opts.Timeout = 180 * time.Second
	}
	return opts, nil
}

func fallbackOrError(opts ClassifyOptions, deterministic RepoClassification, run ClassifyRunSummary, reason string) (RepoClassification, ClassifyRunSummary, error) {
	if opts.Mode == ClassificationAI {
		run.Status = "error"
		run.FallbackReason = reason
		return RepoClassification{}, run, fmt.Errorf("%s", reason)
	}
	run.Status = "warning"
	run.UsedFallback = true
	run.FallbackReason = reason
	run.Source = deterministic.Source
	run.Warnings = Unique(append(run.Warnings, reason))
	deterministic.Warnings = Unique(append(deterministic.Warnings, "AI classifier fallback: "+reason))
	deterministic.ValidationStatus = "warning"
	return deterministic, run, nil
}
