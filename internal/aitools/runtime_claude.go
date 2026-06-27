package aitools

import (
	"path/filepath"

	"github.com/methodnumber13/runweaver/internal/aitools/modelconfig"
	claudecatalog "github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog/claude"
)

type claudeRuntimeAdapter struct {
	metadata claudecatalog.Adapter
}

func (a claudeRuntimeAdapter) ID() string {
	return a.metadata.ID()
}

func (a claudeRuntimeAdapter) Provider() RuntimeProvider {
	return runtimeProviderFromCatalog(a.metadata.Provider())
}

func (a claudeRuntimeAdapter) ProfilePath() string {
	return a.metadata.ProfilePath()
}

func (a claudeRuntimeAdapter) GeneratedPaths() []string {
	return a.metadata.GeneratedPaths()
}

func (a claudeRuntimeAdapter) Capabilities() map[string]RuntimeFlag {
	return runtimeFlagsFromCatalog(a.metadata.Capabilities())
}

func (claudeRuntimeAdapter) PathChecks(root string) (configs, auth, metadata, managed []RuntimeFileCheck) {
	configs = runtimeChecks("config", runtimeClaudeConfigCandidates(root))
	auth = runtimeChecks("auth", runtimeClaudeAuthCandidates())
	metadata = runtimeChecks("metadata", []configCandidate{
		{Path: filepath.Join(root, "CLAUDE.md"), Source: "project"},
		{Path: filepath.Join(root, ".claude", "CLAUDE.md"), Source: "project"},
		{Path: filepath.Join(root, ".claude", "agents"), Source: "project"},
		{Path: filepath.Join(root, ".claude", "skills"), Source: "project"},
		{Path: filepath.Join(root, ".claude", "runweaver", "profile.json"), Source: "project"},
		{Path: modelconfig.ExpandHome("~/.claude/CLAUDE.md"), Source: "user"},
		{Path: modelconfig.ExpandHome("~/.claude/agents"), Source: "user"},
		{Path: modelconfig.ExpandHome("~/.claude/skills"), Source: "user"},
	})
	managed = runtimeChecks("managed", runtimeClaudeManagedCandidates())
	return configs, auth, metadata, managed
}

func (claudeRuntimeAdapter) MaterializeProfile(root string, profile Profile, force bool) error {
	return materializeClaudeProfile(root, profile, force)
}

func (claudeRuntimeAdapter) ExecutionSpec(root string, opts WorkflowExecuteOptions, plan WorkflowRunSummary, prompt string) (workflowRuntimeExecutionSpec, error) {
	return claudeExecutionSpec(root, opts, plan, prompt), nil
}

func (claudeRuntimeAdapter) ClassifierSpec(root, prompt string, opts ClassifyOptions, finalOutputPath string) (classifierCommandSpec, error) {
	args := []string{
		"--print",
		"--output-format", "text",
		"--permission-mode", opts.PermissionMode,
	}
	if opts.ClaudeTools != "" {
		args = append(args, "--tools", opts.ClaudeTools)
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	args = append(args, prompt)
	return classifierCommandSpec{
		Runtime:     RuntimeClaude,
		DisplayName: "Claude Code",
		Binary:      opts.ClaudeBin,
		Args:        args,
	}, nil
}

func (a claudeRuntimeAdapter) DelegationGuidance() string {
	return a.metadata.DelegationGuidance()
}
