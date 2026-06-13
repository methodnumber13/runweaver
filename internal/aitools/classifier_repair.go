package aitools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func repairModelClassificationCoverage(root string, index RepoIndex, current RepoClassification, missing []string, opts ClassifyOptions, runner outputRunner, promptPath, rawOutputPath string) (RepoClassification, []string, string, string, error) {
	repairPrompt := repoClassifierRepairPrompt(current, missing)
	repairPromptPath := filepath.Join(filepath.Dir(promptPath), "repo-classifier-repair-prompt.md")
	repairOutputPath := filepath.Join(filepath.Dir(rawOutputPath), "repo-classifier-repair-output.json")
	if err := os.WriteFile(repairPromptPath, []byte(repairPrompt), 0o644); err != nil {
		return RepoClassification{}, nil, "", "", fmt.Errorf("write repo classifier repair prompt: %w", err)
	}
	raw, err := runModelClassifier(root, repairPrompt, opts, runner, repairOutputPath)
	if len(raw) > 0 {
		if writeErr := os.WriteFile(repairOutputPath, raw, 0o644); writeErr != nil {
			return RepoClassification{}, nil, rel(root, repairPromptPath), "", fmt.Errorf("write repo classifier repair output: %w", writeErr)
		}
	}
	if err != nil {
		return RepoClassification{}, nil, rel(root, repairPromptPath), rel(root, repairOutputPath), err
	}
	modelClassification, err := parseRepoClassification(raw)
	if err != nil {
		return RepoClassification{}, nil, rel(root, repairPromptPath), rel(root, repairOutputPath), fmt.Errorf("repair returned invalid JSON: %w", err)
	}
	validated, warnings, err := ValidateModelClassification(index, modelClassification)
	if err != nil {
		return RepoClassification{}, warnings, rel(root, repairPromptPath), rel(root, repairOutputPath), fmt.Errorf("repair validation failed: %w", err)
	}
	if len(validated.Agents) == 0 || len(validated.Skills) == 0 {
		return RepoClassification{}, warnings, rel(root, repairPromptPath), rel(root, repairOutputPath), fmt.Errorf("repair must include at least one agent and one skill")
	}
	stillMissing := missingMandatoryCoverage(index, validated)
	if len(stillMissing) > 0 {
		warnings = append(warnings, "AI classifier repair still missing mandatory coverage: "+strings.Join(stillMissing, ", "))
	}
	return validated, Unique(warnings), rel(root, repairPromptPath), rel(root, repairOutputPath), nil
}

func repairModelClassificationJSON(root string, raw []byte, parseErr error, opts ClassifyOptions, runner outputRunner, promptPath, rawOutputPath string) (RepoClassification, []string, string, string, error) {
	repairPrompt := repoClassifierJSONRepairPrompt(raw, parseErr)
	repairPromptPath := filepath.Join(filepath.Dir(promptPath), "repo-classifier-json-repair-prompt.md")
	repairOutputPath := filepath.Join(filepath.Dir(rawOutputPath), "repo-classifier-json-repair-output.json")
	if err := os.WriteFile(repairPromptPath, []byte(repairPrompt), 0o644); err != nil {
		return RepoClassification{}, nil, "", "", fmt.Errorf("write repo classifier JSON repair prompt: %w", err)
	}
	repairedRaw, err := runModelClassifier(root, repairPrompt, opts, runner, repairOutputPath)
	if len(repairedRaw) > 0 {
		if writeErr := os.WriteFile(repairOutputPath, repairedRaw, 0o644); writeErr != nil {
			return RepoClassification{}, nil, rel(root, repairPromptPath), "", fmt.Errorf("write repo classifier JSON repair output: %w", writeErr)
		}
	}
	if err != nil {
		return RepoClassification{}, nil, rel(root, repairPromptPath), rel(root, repairOutputPath), err
	}
	classification, err := parseRepoClassification(repairedRaw)
	if err != nil {
		return RepoClassification{}, nil, rel(root, repairPromptPath), rel(root, repairOutputPath), fmt.Errorf("JSON repair returned invalid JSON: %w", err)
	}
	warnings := []string{"AI classifier JSON repair recovered invalid model output: " + parseErr.Error()}
	return classification, warnings, rel(root, repairPromptPath), rel(root, repairOutputPath), nil
}

func repoClassifierJSONRepairPrompt(raw []byte, parseErr error) string {
	text := strings.TrimSpace(string(stripANSI(raw)))
	if len(text) > 20000 {
		text = text[:20000] + "\n...[truncated]"
	}
	var b strings.Builder
	b.WriteString("# Repository Classification JSON Syntax Repair\n\n")
	b.WriteString("You are repo-classifier JSON repair for runweaver. The previous classifier response was syntactically invalid JSON.\n")
	b.WriteString("Return exactly one minified JSON object matching RepoClassification. Do not wrap in markdown. Do not include prose outside JSON.\n")
	b.WriteString("Preserve the same semantic domains, agents, skills, focusFiles, workflow, and verification from the invalid response whenever possible.\n")
	b.WriteString("Do not invent files, routes, packages, URLs, environment variables, or secret values. Drop obviously malformed fragments instead of adding new facts.\n")
	b.WriteString("The repaired JSON must include at least one domain, one agent, and one skill when those concepts are present in the raw response.\n\n")
	b.WriteString("## Parse Error\n")
	b.WriteString(parseErr.Error())
	b.WriteString("\n\n## Invalid Raw Response\n")
	b.WriteString(text)
	b.WriteString("\n\nReturn only the repaired JSON object now.\n")
	return b.String()
}
