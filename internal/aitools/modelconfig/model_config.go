package modelconfig

import (
	"github.com/methodnumber13/runweaver/internal/aitools/foundation"
	"os"
	"strings"
)

// ModelConfigCheckOptions configures provider, model, and endpoint expectations.
type ModelConfigCheckOptions struct {
	ProviderID string
	ModelID    string
	BaseURL    string
}

// ModelConfigCheck reports whether local configuration can reach a model provider.
type ModelConfigCheck struct {
	Status          string                   `json:"status"`
	Ready           bool                     `json:"ready"`
	ProviderID      string                   `json:"providerId"`
	ModelID         string                   `json:"modelId,omitempty"`
	BaseURL         string                   `json:"baseURL,omitempty"`
	ProjectConfig   ConfigFileCheck          `json:"projectConfig"`
	ConfigFiles     []ConfigFileCheck        `json:"configFiles"`
	AuthFiles       []AuthFileCheck          `json:"authFiles"`
	Env             map[string]bool          `json:"env"`
	Issues          []string                 `json:"issues,omitempty"`
	Warnings        []string                 `json:"warnings,omitempty"`
	Recommendations []string                 `json:"recommendations"`
	SampleConfig    map[string]any           `json:"sampleConfig"`
	Detected        DetectedModelConfigState `json:"detected"`
}

// ConfigFileCheck records findings for one runtime configuration file.
type ConfigFileCheck struct {
	Path             string   `json:"path"`
	Exists           bool     `json:"exists"`
	Readable         bool     `json:"readable"`
	Parseable        bool     `json:"parseable"`
	Source           string   `json:"source"`
	HasProvider      bool     `json:"hasProvider"`
	HasModel         bool     `json:"hasModel"`
	HasBaseURL       bool     `json:"hasBaseURL"`
	HasAPIKey        bool     `json:"hasAPIKey"`
	HasLiteralAPIKey bool     `json:"hasLiteralAPIKey,omitempty"`
	APIKeySource     string   `json:"apiKeySource,omitempty"`
	Model            string   `json:"model,omitempty"`
	BaseURL          string   `json:"baseURL,omitempty"`
	Issues           []string `json:"issues,omitempty"`
}

// AuthFileCheck records findings for one runtime auth file.
type AuthFileCheck struct {
	Path        string   `json:"path"`
	Exists      bool     `json:"exists"`
	Readable    bool     `json:"readable"`
	Parseable   bool     `json:"parseable"`
	HasProvider bool     `json:"hasProvider"`
	HasKey      bool     `json:"hasKey"`
	Issues      []string `json:"issues,omitempty"`
}

// DetectedModelConfigState summarizes the effective configuration sources.
type DetectedModelConfigState struct {
	HasProviderConfig bool   `json:"hasProviderConfig"`
	HasModelConfig    bool   `json:"hasModelConfig"`
	HasCredential     bool   `json:"hasCredential"`
	ProviderSource    string `json:"providerSource,omitempty"`
	ModelSource       string `json:"modelSource,omitempty"`
	CredentialSource  string `json:"credentialSource,omitempty"`
}

// CheckModelConfig inspects project, global, managed, and env model settings.
func CheckModelConfig(repoPath string, opts ModelConfigCheckOptions) (ModelConfigCheck, error) {
	root, err := foundation.RepoRoot(repoPath)
	if err != nil {
		return ModelConfigCheck{}, err
	}
	opts.ProviderID = resolveProviderID(root, opts)
	env := map[string]bool{
		"RUNWEAVER_MODEL_API_KEY": os.Getenv("RUNWEAVER_MODEL_API_KEY") != "",
		"OPENCODE_CONFIG":         os.Getenv("OPENCODE_CONFIG") != "",
		"OPENCODE_CONFIG_DIR":     os.Getenv("OPENCODE_CONFIG_DIR") != "",
		"OPENCODE_CONFIG_CONTENT": os.Getenv("OPENCODE_CONFIG_CONTENT") != "",
		"XDG_CONFIG_HOME":         os.Getenv("XDG_CONFIG_HOME") != "",
		"XDG_DATA_HOME":           os.Getenv("XDG_DATA_HOME") != "",
	}
	result := ModelConfigCheck{
		Status:     "error",
		ProviderID: opts.ProviderID,
		Env:        env,
	}

	configPaths := ConfigPaths(root)
	for _, candidate := range configPaths {
		check, parsed := inspectConfigFile(candidate.Path, candidate.Source, opts)
		if candidate.Source == "project" {
			result.ProjectConfig = check
		}
		result.ConfigFiles = append(result.ConfigFiles, check)
		if parsed == nil {
			continue
		}
		applyModelConfigState(&result, check, parsed, opts)
	}
	if content := os.Getenv("OPENCODE_CONFIG_CONTENT"); content != "" {
		check, parsed := inspectConfigContent(content, "env:OPENCODE_CONFIG_CONTENT", opts)
		result.ConfigFiles = append(result.ConfigFiles, check)
		if parsed != nil {
			applyModelConfigState(&result, check, parsed, opts)
		}
	}
	for _, candidate := range ManagedConfigPaths() {
		check, parsed := inspectConfigFile(candidate.Path, candidate.Source, opts)
		result.ConfigFiles = append(result.ConfigFiles, check)
		if parsed != nil {
			applyModelConfigState(&result, check, parsed, opts)
		}
	}

	for _, candidate := range AuthFilePaths() {
		check := inspectAuthFile(candidate, opts.ProviderID)
		result.AuthFiles = append(result.AuthFiles, check)
		if check.HasProvider && check.HasKey && !result.Detected.HasCredential {
			result.Detected.HasCredential = true
			result.Detected.CredentialSource = check.Path
		}
	}
	if !result.Detected.HasCredential && env["RUNWEAVER_MODEL_API_KEY"] {
		result.Detected.HasCredential = true
		result.Detected.CredentialSource = "env:RUNWEAVER_MODEL_API_KEY"
	}
	for _, check := range result.ConfigFiles {
		if configCheckHasResolvedCredential(check) && !result.Detected.HasCredential {
			result.Detected.HasCredential = true
			result.Detected.CredentialSource = check.Path + " provider.options.apiKey"
		}
	}

	result.Issues = modelConfigIssues(result, opts)
	result.Warnings = modelConfigWarnings(result, opts)
	result.Ready = len(result.Issues) == 0
	if result.Ready {
		result.Status = "ok"
	} else {
		result.Status = "warning"
	}
	result.Recommendations = modelConfigRecommendations(result, opts)
	result.SampleConfig = sampleOpenAIProviderConfig(opts)
	return result, nil
}

func applyModelConfigState(result *ModelConfigCheck, check ConfigFileCheck, parsed map[string]any, opts ModelConfigCheckOptions) {
	if provider, ok := providerConfig(parsed, opts.ProviderID); ok {
		result.Detected.HasProviderConfig = true
		result.Detected.ProviderSource = check.Path
		if baseURL, ok := nestedString(provider, "options", "baseURL"); ok && strings.TrimSpace(baseURL) != "" {
			result.BaseURL = baseURL
		}
	}
	if model, ok := parsed["model"].(string); ok && strings.TrimSpace(model) != "" {
		result.Detected.HasModelConfig = true
		result.Detected.ModelSource = check.Path
		result.ModelID = model
	}
}
