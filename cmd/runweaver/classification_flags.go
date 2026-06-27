package main

import (
	"flag"
	"strings"
	"time"

	"github.com/methodnumber13/runweaver/internal/aitools"
)

type classificationFlags struct {
	mode              *string
	classifierMode    *string
	classifierRuntime *string
	provider          *string
	model             *string
	opencodeBin       *string
	codexBin          *string
	claudeBin         *string
	agent             *string
	sandbox           *string
	approvalPolicy    *string
	permissionMode    *string
	claudeTools       *string
	skipGitRepoCheck  *bool
	timeout           *time.Duration
	skipModelCheck    *bool
	skipRuntimeCheck  *bool
}

func addClassificationFlags(fs *flag.FlagSet, defaultMode string) classificationFlags {
	return classificationFlags{
		mode:              fs.String("classification", defaultMode, "repository classification mode: auto, ai, or deterministic"),
		classifierMode:    fs.String("classifier", "", "alias for --classification"),
		classifierRuntime: fs.String("classifier-runtime", "", "runtime used for AI classification: opencode, codex, or claude"),
		provider:          fs.String("classifier-provider", "", "OpenCode provider id for AI classification; defaults to the provider prefix in the configured model"),
		model:             fs.String("classifier-model", "", "optional runtime model override for AI classification"),
		opencodeBin:       fs.String("classifier-opencode-bin", "opencode", "OpenCode executable path for AI classification"),
		codexBin:          fs.String("classifier-codex-bin", "codex", "Codex executable path for AI classification"),
		claudeBin:         fs.String("classifier-claude-bin", "claude", "Claude Code executable path for AI classification"),
		agent:             fs.String("classifier-agent", "repo-classifier", "OpenCode agent used for AI classification"),
		sandbox:           fs.String("classifier-sandbox", "read-only", "Codex classifier sandbox mode"),
		approvalPolicy:    fs.String("classifier-approval-policy", "never", "Codex classifier approval policy"),
		permissionMode:    fs.String("classifier-permission-mode", "dontAsk", "Claude classifier permission mode"),
		claudeTools:       fs.String("classifier-claude-tools", "", "optional Claude classifier tools list for --tools"),
		skipGitRepoCheck:  fs.Bool("classifier-skip-git-repo-check", false, "allow Codex classifier outside a Git repository"),
		timeout:           fs.Duration("classifier-timeout", 180*time.Second, "AI classifier timeout, for example 60s or 3m"),
		skipModelCheck:    fs.Bool("classifier-skip-model-check", false, "skip model preflight before AI classification"),
		skipRuntimeCheck:  fs.Bool("classifier-skip-runtime-check", false, "skip runtime binary/config discovery before AI classification"),
	}
}

func (f classificationFlags) options() (aitools.ClassifyOptions, error) {
	mode := strings.TrimSpace(*f.mode)
	if f.classifierMode != nil && strings.TrimSpace(*f.classifierMode) != "" {
		mode = strings.TrimSpace(*f.classifierMode)
	}
	return aitools.ClassifyOptions{
		Mode:             aitools.ClassificationMode(mode),
		Runtime:          strings.TrimSpace(*f.classifierRuntime),
		ProviderID:       strings.TrimSpace(*f.provider),
		Model:            strings.TrimSpace(*f.model),
		OpencodeBin:      strings.TrimSpace(*f.opencodeBin),
		CodexBin:         strings.TrimSpace(*f.codexBin),
		ClaudeBin:        strings.TrimSpace(*f.claudeBin),
		Agent:            strings.TrimSpace(*f.agent),
		Sandbox:          strings.TrimSpace(*f.sandbox),
		ApprovalPolicy:   strings.TrimSpace(*f.approvalPolicy),
		PermissionMode:   strings.TrimSpace(*f.permissionMode),
		ClaudeTools:      strings.TrimSpace(*f.claudeTools),
		SkipGitRepoCheck: *f.skipGitRepoCheck,
		Timeout:          *f.timeout,
		SkipModelCheck:   *f.skipModelCheck,
		SkipRuntimeCheck: *f.skipRuntimeCheck,
	}, nil
}
