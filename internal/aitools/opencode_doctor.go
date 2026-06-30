package aitools

import (
	"context"
	"github.com/methodnumber13/runweaver/internal/aitools/runtimeenv"
	"time"
)

// DoctorOpenCode validates whether OpenCode can run the generated RunWeaver metadata.
func DoctorOpenCode(repoPath string, opts OpenCodeDoctorOptions) (OpenCodeDoctorResult, error) {
	return doctorOpenCode(repoPath, opts, runCommandOutputWithEnv)
}

func doctorOpenCode(repoPath string, opts OpenCodeDoctorOptions, runner outputRunner) (OpenCodeDoctorResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return OpenCodeDoctorResult{}, err
	}
	cleanup := trackOpenCodeDependencyArtifacts(root)
	defer cleanup()
	opts = normalizeOpenCodeDoctorOptions(opts)
	result := OpenCodeDoctorResult{
		Status:      "ok",
		Ready:       true,
		RepoRoot:    root,
		OpencodeBin: opts.OpencodeBin,
		Agent:       opts.Agent,
	}

	checkRunWeaverAvailability(&result)
	checkLocalMetadata(root, opts.Agent, &result)

	configRaw, err := runOpenCodeDoctorCommand(root, opts, runner, "debug", "config")
	if err != nil {
		addDoctorCheck(&result, "opencode-debug-config", "error", "OpenCode config could not be resolved", []string{err.Error()}, []string{"Install OpenCode, ensure it is on PATH for Desktop/CLI, then run runweaver doctor opencode --repo . again."})
	} else {
		config, parseErr := parseJSONObject(configRaw)
		if parseErr != nil {
			addDoctorCheck(&result, "opencode-debug-config", "error", "OpenCode debug config output is not JSON", []string{parseErr.Error()}, []string{"Run opencode debug config manually and check for unexpected output."})
		} else {
			checkResolvedConfig(config, opts.Agent, &result)
		}
	}

	agentRaw, err := runOpenCodeDoctorCommand(root, opts, runner, "debug", "agent", opts.Agent)
	if err != nil {
		addDoctorCheck(&result, "opencode-debug-agent", "error", "OpenCode cannot resolve the RunWeaver OpenCode agent", []string{err.Error()}, []string{"Run runweaver init --repo . --force, then check opencode debug agent " + opts.Agent + "."})
	} else {
		agent, parseErr := parseJSONObject(agentRaw)
		if parseErr != nil {
			addDoctorCheck(&result, "opencode-debug-agent", "error", "OpenCode debug agent output is not JSON", []string{parseErr.Error()}, []string{"Run opencode debug agent " + opts.Agent + " manually and inspect the output."})
		} else {
			checkResolvedAgent(agent, opts.Agent, &result)
		}
	}

	agentListRaw, err := runOpenCodeDoctorCommand(root, opts, runner, "agent", "list")
	if err != nil {
		addDoctorCheck(&result, "opencode-agent-list", "warning", "OpenCode agent list command was not available", []string{err.Error()}, []string{"This is non-fatal if opencode debug agent " + opts.Agent + " succeeds."})
	} else {
		addDoctorCheck(&result, "opencode-agent-list", "ok", "OpenCode agent discovery command completed", firstLines(string(agentListRaw), 5), nil)
	}

	if !opts.SkipModelCheck {
		check, err := CheckModelConfig(root, ModelConfigCheckOptions{ProviderID: opts.ProviderID})
		if err != nil {
			addDoctorCheck(&result, "model-preflight", "error", "OpenCode model preflight failed internally", []string{err.Error()}, []string{"Fix local OpenCode config readability/parsing and rerun doctor."})
		} else {
			result.ModelPreflight = &check
			if check.Ready {
				addDoctorCheck(&result, "model-preflight", "ok", "OpenCode provider, model, and credential are configured", []string{check.Detected.ProviderSource, check.Detected.ModelSource, check.Detected.CredentialSource}, nil)
			} else {
				addDoctorCheck(&result, "model-preflight", "warning", "OpenCode model setup is incomplete", check.Issues, check.Recommendations)
			}
			if check.Detected.CredentialSource == "env:RUNWEAVER_MODEL_API_KEY" {
				addDoctorCheck(&result, "desktop-credential-source", "warning", "Model provider key is only available through the current shell environment", []string{"credentialSource=env:RUNWEAVER_MODEL_API_KEY"}, []string{"For OpenCode Desktop, prefer OpenCode auth storage or provider.options.apiKey with a {file:...} reference so GUI-launched processes can read the key."})
			}
		}
	}

	finalizeDoctorResult(&result)
	return result, nil
}

func normalizeOpenCodeDoctorOptions(opts OpenCodeDoctorOptions) OpenCodeDoctorOptions {
	if opts.OpencodeBin == "" {
		opts.OpencodeBin = "opencode"
	}
	if opts.Agent == "" {
		opts.Agent = OpenCodePrimaryAgentName
	}
	if opts.Timeout == 0 {
		opts.Timeout = 45 * time.Second
	}
	return opts
}

func runOpenCodeDoctorCommand(root string, opts OpenCodeDoctorOptions, runner outputRunner, args ...string) ([]byte, error) {
	ctx := context.Background()
	cancel := func() {}
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
	}
	defer cancel()
	return runner(ctx, root, opts.OpencodeBin, args, runtimeenv.OpenCodeProviderEnv(root, opts.ProviderID))
}
