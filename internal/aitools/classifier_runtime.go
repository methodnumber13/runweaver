package aitools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type classifierCommandSpec struct {
	Runtime         string
	DisplayName     string
	Binary          string
	Args            []string
	Env             []string
	FinalOutputPath string
	Cleanup         func()
}

func runModelClassifier(root, prompt string, opts ClassifyOptions, runner outputRunner, finalOutputPath string) ([]byte, error) {
	spec, err := classifierRuntimeCommandSpec(root, prompt, opts, finalOutputPath)
	if err != nil {
		return nil, err
	}
	if spec.Cleanup != nil {
		defer spec.Cleanup()
	}
	ctx := context.Background()
	cancel := func() {}
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
	}
	defer cancel()
	raw, err := runner(ctx, root, spec.Binary, spec.Args, spec.Env)
	if spec.FinalOutputPath != "" {
		if data, readErr := os.ReadFile(spec.FinalOutputPath); readErr == nil && len(bytes.TrimSpace(data)) > 0 {
			return data, err
		}
	}
	return raw, err
}

func classifierRuntimeCommandSpec(root, prompt string, opts ClassifyOptions, finalOutputPath string) (classifierCommandSpec, error) {
	adapter, err := mustRuntimeAdapter(opts.Runtime)
	if err != nil {
		return classifierCommandSpec{}, fmt.Errorf("unsupported classifier runtime %q", opts.Runtime)
	}
	return adapter.ClassifierSpec(root, prompt, opts, finalOutputPath)
}

func trackOpenCodeDependencyArtifacts(root string) func() {
	existed := map[string]bool{}
	for _, relPath := range []string{
		".opencode/node_modules",
		".opencode/package.json",
		".opencode/package-lock.json",
		".opencode/pnpm-lock.yaml",
		".opencode/yarn.lock",
	} {
		existed[relPath] = Exists(filepath.Join(root, relPath))
	}
	return func() {
		for relPath, alreadyExisted := range existed {
			if alreadyExisted {
				continue
			}
			_ = os.RemoveAll(filepath.Join(root, relPath))
		}
	}
}

func modelForOpenCode(providerID, model string) string {
	model = strings.TrimSpace(model)
	if model == "" || strings.Contains(model, "/") {
		return model
	}
	return providerID + "/" + model
}

func artifactAbsPath(root, relPath, fallback string) string {
	if relPath == "" {
		return fallback
	}
	if filepath.IsAbs(relPath) {
		return relPath
	}
	return filepath.Join(root, relPath)
}
