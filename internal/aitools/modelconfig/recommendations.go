package modelconfig

import (
	"fmt"
	"strings"
)

func modelConfigIssues(result ModelConfigCheck, opts ModelConfigCheckOptions) []string {
	var issues []string
	if !result.Detected.HasProviderConfig {
		issues = append(issues, "provider "+result.ProviderID+" is not configured in any detected OpenCode config")
	}
	if !result.Detected.HasModelConfig {
		issues = append(issues, "default model is not configured")
	}
	if !result.Detected.HasCredential {
		issues = append(issues, "no credential found for "+result.ProviderID+" in OpenCode auth, provider.options.apiKey, or RUNWEAVER_MODEL_API_KEY")
	}
	if result.ProviderID == "openai-compatible" && result.Detected.HasProviderConfig && result.BaseURL == "" {
		issues = append(issues, "provider "+result.ProviderID+" is configured without options.baseURL")
	}
	if opts.BaseURL != "" && result.BaseURL == "" {
		issues = append(issues, fmt.Sprintf("configured baseURL is missing; expected %q", opts.BaseURL))
	}
	if opts.BaseURL != "" && result.BaseURL != "" && result.BaseURL != opts.BaseURL {
		issues = append(issues, fmt.Sprintf("configured baseURL %q does not match expected %q", result.BaseURL, opts.BaseURL))
	}
	if opts.ModelID != "" && result.ModelID != "" && result.ModelID != opts.ProviderID+"/"+opts.ModelID && result.ModelID != opts.ModelID {
		issues = append(issues, fmt.Sprintf("configured model %q does not match expected %q", result.ModelID, opts.ModelID))
	}
	return issues
}

func modelConfigWarnings(result ModelConfigCheck, opts ModelConfigCheckOptions) []string {
	var warnings []string
	if !result.ProjectConfig.Exists {
		warnings = append(warnings, "project opencode.json is missing; runweaver init will create it, but run this doctor again after init")
	}
	if result.BaseURL != "" && !strings.Contains(result.BaseURL, "/v1") && !strings.Contains(result.BaseURL, "/v1-openai") {
		warnings = append(warnings, "OpenAI-compatible provider baseURL usually ends with /v1 or /v1-openai")
	}
	for _, check := range result.ConfigFiles {
		if check.HasLiteralAPIKey {
			warnings = append(warnings, "provider.options.apiKey is present in "+check.Path+"; prefer {env:RUNWEAVER_MODEL_API_KEY} or OpenCode auth storage instead of literal secrets")
		}
	}
	return warnings
}

func modelConfigRecommendations(result ModelConfigCheck, opts ModelConfigCheckOptions) []string {
	var out []string
	if !result.Detected.HasProviderConfig {
		out = append(out, "add provider."+result.ProviderID+" with npm @ai-sdk/openai-compatible and a baseURL")
	}
	if !result.Detected.HasModelConfig {
		out = append(out, "set model to "+result.ProviderID+"/<model-id>")
	}
	if !result.Detected.HasCredential {
		out = append(out, "run OpenCode /connect for provider "+result.ProviderID+" or export RUNWEAVER_MODEL_API_KEY and reference it as {env:RUNWEAVER_MODEL_API_KEY}")
	}
	out = append(out, "run runweaver doctor model --repo . before runweaver init --require-model")
	return out
}

func sampleOpenAIProviderConfig(opts ModelConfigCheckOptions) map[string]any {
	providerID := opts.ProviderID
	if providerID == "" {
		providerID = "openai-compatible"
	}
	modelID := opts.ModelID
	if modelID == "" {
		modelID = "your-model-id"
	}
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "https://llm-provider.example.com/v1"
	}
	return map[string]any{
		"$schema": "https://opencode.ai/config.json",
		"model":   providerID + "/" + modelID,
		"provider": map[string]any{
			providerID: map[string]any{
				"npm":  "@ai-sdk/openai-compatible",
				"name": "OpenAI-compatible",
				"options": map[string]any{
					"baseURL": baseURL,
					"apiKey":  "{env:RUNWEAVER_MODEL_API_KEY}",
				},
				"models": map[string]any{
					modelID: map[string]any{"name": modelID},
				},
			},
		},
	}
}
