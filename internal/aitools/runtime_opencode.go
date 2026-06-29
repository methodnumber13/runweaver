package aitools

import (
	"path/filepath"

	opencodecatalog "github.com/methodnumber13/runweaver/internal/aitools/runtimecatalog/opencode"
	"github.com/methodnumber13/runweaver/internal/aitools/runtimeenv"
)

type openCodeRuntimeAdapter struct {
	metadata opencodecatalog.Adapter
}

func (a openCodeRuntimeAdapter) ID() string {
	return a.metadata.ID()
}

func (a openCodeRuntimeAdapter) Provider() RuntimeProvider {
	return runtimeProviderFromCatalog(a.metadata.Provider())
}

func (a openCodeRuntimeAdapter) ProfilePath() string {
	return a.metadata.ProfilePath()
}

func (a openCodeRuntimeAdapter) GeneratedPaths() []string {
	return a.metadata.GeneratedPaths()
}

func (a openCodeRuntimeAdapter) Capabilities() map[string]RuntimeFlag {
	return runtimeFlagsFromCatalog(a.metadata.Capabilities())
}

func (openCodeRuntimeAdapter) PathChecks(root string) (configs, auth, metadata, managed []RuntimeFileCheck) {
	configs = runtimeChecks("config", runtimeOpenCodeConfigCandidates(root))
	auth = runtimeChecks("auth", runtimeOpenCodeAuthCandidates())
	metadata = runtimeChecks("metadata", []configCandidate{
		{Path: filepath.Join(root, ".opencode", "agents"), Source: "project"},
		{Path: filepath.Join(root, ".opencode", "commands"), Source: "project"},
		{Path: filepath.Join(root, ".opencode", "skills"), Source: "project"},
		{Path: filepath.Join(root, ".opencode", "workflows"), Source: "project"},
		{Path: filepath.Join(root, ".opencode", "swarm", "profile.json"), Source: "project"},
	})
	managed = runtimeChecks("managed", runtimeOpenCodeManagedCandidates())
	return configs, auth, metadata, managed
}

func (openCodeRuntimeAdapter) MaterializeProfile(root string, profile Profile, force bool) error {
	return materializeOpenCodeProfile(root, profile, force)
}

func (openCodeRuntimeAdapter) ExecutionSpec(root string, opts WorkflowExecuteOptions, plan WorkflowRunSummary, prompt string) (workflowRuntimeExecutionSpec, error) {
	return openCodeExecutionSpec(root, opts, plan, prompt), nil
}

func (openCodeRuntimeAdapter) ClassifierSpec(root, prompt string, opts ClassifyOptions, finalOutputPath string) (classifierCommandSpec, error) {
	cleanup := trackOpenCodeDependencyArtifacts(root)
	args := []string{
		"run",
		"--pure",
		"--agent", opts.Agent,
		"--dir", root,
		"--format", "default",
		"--title", "runweaver repo classifier",
	}
	if opts.Model != "" {
		args = append(args, "--model", modelForOpenCode(opts.ProviderID, opts.Model))
	}
	args = append(args, prompt)
	return classifierCommandSpec{
		Runtime:     RuntimeOpenCode,
		DisplayName: "OpenCode",
		Binary:      opts.OpencodeBin,
		Args:        args,
		Env:         runtimeenv.OpenCodeProviderEnv(root, opts.ProviderID),
		Cleanup:     cleanup,
	}, nil
}

func (a openCodeRuntimeAdapter) DelegationGuidance() string {
	return a.metadata.DelegationGuidance()
}
