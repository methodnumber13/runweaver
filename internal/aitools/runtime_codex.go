package aitools

import (
	"path/filepath"

	"github.com/methodnumber13/runweaver/internal/aitools/modelconfig"
	codexcatalog "github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog/codex"
)

type codexRuntimeAdapter struct {
	metadata codexcatalog.Adapter
}

func (a codexRuntimeAdapter) ID() string {
	return a.metadata.ID()
}

func (a codexRuntimeAdapter) Provider() RuntimeProvider {
	return runtimeProviderFromCatalog(a.metadata.Provider())
}

func (a codexRuntimeAdapter) ProfilePath() string {
	return a.metadata.ProfilePath()
}

func (a codexRuntimeAdapter) GeneratedPaths() []string {
	return a.metadata.GeneratedPaths()
}

func (a codexRuntimeAdapter) Capabilities() map[string]RuntimeFlag {
	return runtimeFlagsFromCatalog(a.metadata.Capabilities())
}

func (codexRuntimeAdapter) PathChecks(root string) (configs, auth, metadata, managed []RuntimeFileCheck) {
	configs = runtimeChecks("config", runtimeCodexConfigCandidates(root))
	auth = runtimeChecks("auth", runtimeCodexAuthCandidates())
	metadata = runtimeChecks("metadata", []configCandidate{
		{Path: filepath.Join(root, "AGENTS.md"), Source: "project"},
		{Path: filepath.Join(root, ".agents", "skills"), Source: "project"},
		{Path: filepath.Join(root, ".codex", "agents"), Source: "project"},
		{Path: filepath.Join(root, ".codex", "runweaver", "profile.json"), Source: "project"},
		{Path: modelconfig.ExpandHome("~/.agents/skills"), Source: "user"},
		{Path: modelconfig.ExpandHome("~/.codex/agents"), Source: "user"},
	})
	managed = runtimeChecks("managed", runtimeCodexManagedCandidates())
	return configs, auth, metadata, managed
}

func (codexRuntimeAdapter) MaterializeProfile(root string, profile Profile, force bool) error {
	return materializeCodexProfile(root, profile, force)
}

func (codexRuntimeAdapter) ExecutionSpec(root string, opts WorkflowExecuteOptions, plan WorkflowRunSummary, prompt string) (workflowRuntimeExecutionSpec, error) {
	return codexExecutionSpec(root, opts, plan, prompt), nil
}

func (codexRuntimeAdapter) ClassifierSpec(root, prompt string, opts ClassifyOptions, finalOutputPath string) (classifierCommandSpec, error) {
	args := []string{
		"-a", opts.ApprovalPolicy,
		"exec",
		"--json",
		"--ephemeral",
		"-C", root,
		"--sandbox", opts.Sandbox,
		"--color", "never",
	}
	if finalOutputPath != "" {
		args = append(args, "--output-last-message", finalOutputPath)
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.SkipGitRepoCheck {
		args = append(args, "--skip-git-repo-check")
	}
	args = append(args, prompt)
	return classifierCommandSpec{
		Runtime:         RuntimeCodex,
		DisplayName:     "Codex",
		Binary:          opts.CodexBin,
		Args:            args,
		FinalOutputPath: finalOutputPath,
	}, nil
}

func (a codexRuntimeAdapter) DelegationGuidance() string {
	return a.metadata.DelegationGuidance()
}
