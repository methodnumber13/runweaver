package aitools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func classifyRepoIndex(root string, index RepoIndex, opts ClassifyOptions, runner outputRunner) (RepoClassification, ClassifyRunSummary, error) {
	opts, err := normalizeClassifyOptions(opts, ClassificationDeterministic)
	if err != nil {
		return RepoClassification{}, ClassifyRunSummary{}, err
	}
	deterministic := ClassifyRepoDeterministic(index)
	promptIndex := index
	promptIndex.Classification = deterministic
	prompt := repoClassifierPrompt(promptIndex)
	promptPath := artifactAbsPath(root, index.Artifacts.ClassifierPrompt, filepath.Join(root, ".opencode", "tmp", "runweaver", "index", "repo-classifier-prompt.md"))
	rawOutputPath := artifactAbsPath(root, index.Artifacts.ClassifierOutput, filepath.Join(root, ".opencode", "tmp", "runweaver", "index", "repo-classifier-output.json"))
	if err := os.MkdirAll(filepath.Dir(promptPath), 0o755); err != nil {
		return RepoClassification{}, ClassifyRunSummary{}, fmt.Errorf("create classifier artifact directory: %w", err)
	}
	if err := os.WriteFile(promptPath, []byte(prompt), 0o644); err != nil {
		return RepoClassification{}, ClassifyRunSummary{}, fmt.Errorf("write repo classifier prompt: %w", err)
	}
	run := ClassifyRunSummary{
		Status:     "success",
		Mode:       string(opts.Mode),
		Runtime:    opts.Runtime,
		Source:     deterministic.Source,
		PromptPath: rel(root, promptPath),
	}
	if opts.Mode == ClassificationDeterministic {
		return deterministic, run, nil
	}

	if opts.Runtime != RuntimeOpenCode && !opts.SkipRuntimeCheck {
		runtimeChecks, err := DiscoverRuntimes(root, RuntimeDiscoveryOptions{Runtime: opts.Runtime})
		if err != nil {
			return fallbackOrError(opts, deterministic, run, "runtime preflight failed internally: "+err.Error())
		}
		if len(runtimeChecks) > 0 {
			run.RuntimePreflight = &runtimeChecks[0]
			if !runtimeChecks[0].BinaryFound {
				return fallbackOrError(opts, deterministic, run, runtimeChecks[0].Binary+" is not available on PATH")
			}
			if !runtimeChecks[0].Ready {
				run.Warnings = Unique(append(run.Warnings, "runtime metadata/auth discovery is incomplete; classifier will still try "+runtimeChecks[0].Name))
			}
		}
	}

	var modelPreflight *ModelConfigCheck
	if opts.Runtime == RuntimeOpenCode && !opts.SkipModelCheck {
		check, err := CheckModelConfig(root, ModelConfigCheckOptions{
			ProviderID: opts.ProviderID,
		})
		if err != nil {
			return fallbackOrError(opts, deterministic, run, "model preflight failed internally: "+err.Error())
		}
		modelPreflight = &check
		run.ModelPreflight = modelPreflight
		if !check.Ready {
			return fallbackOrError(opts, deterministic, run, "model preflight failed: "+strings.Join(check.Issues, "; "))
		}
	}

	run.ModelAttempted = true
	raw, err := runModelClassifier(root, prompt, opts, runner, rawOutputPath)
	if len(raw) > 0 {
		if writeErr := os.WriteFile(rawOutputPath, raw, 0o644); writeErr != nil {
			run.Warnings = append(run.Warnings, "could not write classifier raw output: "+writeErr.Error())
		} else {
			run.RawOutputPath = rel(root, rawOutputPath)
		}
	}
	if err != nil {
		return fallbackOrError(opts, deterministic, run, runtimeDisplayName(opts.Runtime)+" model classifier failed: "+err.Error())
	}
	modelClassification, err := parseRepoClassification(raw)
	if err != nil {
		repaired, repairWarnings, repairPromptPath, repairOutputPath, repairErr := repairModelClassificationJSON(root, raw, err, opts, runner, promptPath, rawOutputPath)
		run.Warnings = Unique(append(run.Warnings, repairWarnings...))
		if repairPromptPath != "" {
			run.PromptPath = repairPromptPath
		}
		if repairOutputPath != "" {
			run.RawOutputPath = repairOutputPath
		}
		if repairErr != nil {
			return fallbackOrError(opts, deterministic, run, "model classifier returned invalid JSON and JSON repair failed: "+repairErr.Error())
		}
		modelClassification = repaired
	}
	validated, validationWarnings, err := ValidateModelClassification(index, modelClassification)
	run.Warnings = Unique(append(run.Warnings, validationWarnings...))
	if err != nil {
		run.ValidationError = err.Error()
		return fallbackOrError(opts, deterministic, run, "model classification validation failed: "+err.Error())
	}
	if len(validated.Agents) == 0 || len(validated.Skills) == 0 {
		err := fmt.Errorf("model classification must include at least one agent and one skill")
		run.ValidationError = err.Error()
		return fallbackOrError(opts, deterministic, run, "model classification validation failed: "+err.Error())
	}
	if missingCoverage := missingMandatoryCoverage(index, validated); len(missingCoverage) > 0 {
		repaired, repairWarnings, repairPromptPath, repairOutputPath, err := repairModelClassificationCoverage(root, index, validated, missingCoverage, opts, runner, promptPath, rawOutputPath)
		run.Warnings = Unique(append(run.Warnings, repairWarnings...))
		if repairPromptPath != "" {
			run.PromptPath = repairPromptPath
		}
		if repairOutputPath != "" {
			run.RawOutputPath = repairOutputPath
		}
		if err != nil {
			run.Warnings = Unique(append(run.Warnings, "AI classifier repair failed; using initial valid AI classification: "+err.Error()))
		} else {
			validated = repaired
		}
	}
	validated = normalizeDomainFirstClassification(index, validated)
	validated.Source = "model-backed-" + opts.Runtime
	validated.ModelReady = true
	if len(run.Warnings) > 0 {
		validated.Warnings = Unique(append(validated.Warnings, run.Warnings...))
	}
	if len(validated.Warnings) > 0 || len(run.Warnings) > 0 {
		validated.ValidationStatus = "warning"
	} else {
		validated.ValidationStatus = "valid"
	}
	run.Status = "success"
	run.Source = validated.Source
	run.ModelUsed = true
	return validated, run, nil
}
