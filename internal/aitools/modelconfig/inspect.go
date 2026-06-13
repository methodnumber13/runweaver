package modelconfig

import (
	"encoding/json"
	"github.com/methodnumber13/runweaver/internal/aitools/jsonc"
	"os"
	"path/filepath"
	"strings"
)

func inspectConfigFile(path, source string, opts ModelConfigCheckOptions) (ConfigFileCheck, map[string]any) {
	check := ConfigFileCheck{Path: path, Source: source}
	data, err := os.ReadFile(path)
	if err != nil {
		check.Exists = false
		if !os.IsNotExist(err) {
			check.Issues = append(check.Issues, err.Error())
		}
		return check, nil
	}
	check.Exists = true
	check.Readable = true
	var parsed map[string]any
	if err := json.Unmarshal(jsonc.StripComments(data), &parsed); err != nil {
		check.Issues = append(check.Issues, "cannot parse JSON/JSONC: "+err.Error())
		return check, nil
	}
	check.Parseable = true
	if model, ok := parsed["model"].(string); ok {
		check.Model = model
		check.HasModel = model != ""
	}
	if provider, ok := providerConfig(parsed, opts.ProviderID); ok {
		check.HasProvider = true
		if baseURL, ok := nestedString(provider, "options", "baseURL"); ok {
			check.BaseURL = baseURL
			check.HasBaseURL = baseURL != ""
		}
		if apiKey, ok := nestedString(provider, "options", "apiKey"); ok && strings.TrimSpace(apiKey) != "" {
			check.HasAPIKey = true
			check.APIKeySource = apiKeySource(apiKey)
			if strings.HasPrefix(apiKey, "{env:") && strings.HasSuffix(apiKey, "}") {
				envName := strings.TrimSuffix(strings.TrimPrefix(apiKey, "{env:"), "}")
				if os.Getenv(envName) == "" {
					check.Issues = append(check.Issues, "provider.options.apiKey references unset environment variable "+envName)
				}
			} else if strings.HasPrefix(apiKey, "{file:") && strings.HasSuffix(apiKey, "}") {
				fileRef := strings.TrimSuffix(strings.TrimPrefix(apiKey, "{file:"), "}")
				if !credentialFileHasContent(check.Path, fileRef) {
					check.Issues = append(check.Issues, "provider.options.apiKey file reference is missing or empty")
				}
			} else {
				check.HasLiteralAPIKey = true
			}
		}
	}
	return check, parsed
}

func inspectConfigContent(content, source string, opts ModelConfigCheckOptions) (ConfigFileCheck, map[string]any) {
	check := ConfigFileCheck{Path: source, Source: source, Exists: true, Readable: true}
	var parsed map[string]any
	if err := json.Unmarshal(jsonc.StripComments([]byte(content)), &parsed); err != nil {
		check.Issues = append(check.Issues, "cannot parse JSON/JSONC: "+err.Error())
		return check, nil
	}
	check.Parseable = true
	if model, ok := parsed["model"].(string); ok {
		check.Model = model
		check.HasModel = model != ""
	}
	if provider, ok := providerConfig(parsed, opts.ProviderID); ok {
		check.HasProvider = true
		if baseURL, ok := nestedString(provider, "options", "baseURL"); ok {
			check.BaseURL = baseURL
			check.HasBaseURL = baseURL != ""
		}
		if apiKey, ok := nestedString(provider, "options", "apiKey"); ok && strings.TrimSpace(apiKey) != "" {
			check.HasAPIKey = true
			check.APIKeySource = apiKeySource(apiKey)
			if strings.HasPrefix(apiKey, "{env:") && strings.HasSuffix(apiKey, "}") {
				envName := strings.TrimSuffix(strings.TrimPrefix(apiKey, "{env:"), "}")
				if os.Getenv(envName) == "" {
					check.Issues = append(check.Issues, "provider.options.apiKey references unset environment variable "+envName)
				}
			} else if strings.HasPrefix(apiKey, "{file:") && strings.HasSuffix(apiKey, "}") {
				check.Issues = append(check.Issues, "provider.options.apiKey file references cannot be resolved from OPENCODE_CONFIG_CONTENT")
			} else {
				check.HasLiteralAPIKey = true
			}
		}
	}
	return check, parsed
}

func apiKeySource(apiKey string) string {
	apiKey = strings.TrimSpace(apiKey)
	if strings.HasPrefix(apiKey, "{env:") && strings.HasSuffix(apiKey, "}") {
		return "env:" + strings.TrimSuffix(strings.TrimPrefix(apiKey, "{env:"), "}")
	}
	if strings.HasPrefix(apiKey, "{file:") && strings.HasSuffix(apiKey, "}") {
		return "file"
	}
	return "literal"
}

func configCheckHasResolvedCredential(check ConfigFileCheck) bool {
	if !check.HasAPIKey {
		return false
	}
	if check.HasLiteralAPIKey {
		return true
	}
	if strings.HasPrefix(check.APIKeySource, "env:") {
		return os.Getenv(strings.TrimPrefix(check.APIKeySource, "env:")) != ""
	}
	if check.APIKeySource == "file" {
		return len(check.Issues) == 0
	}
	return false
}

func credentialFileHasContent(configPath, ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(configPath, "env:") {
		return false
	}
	path := ExpandHome(ref)
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(configPath), path)
	}
	data, err := os.ReadFile(path)
	return err == nil && strings.TrimSpace(string(data)) != ""
}

func inspectAuthFile(path, providerID string) AuthFileCheck {
	check := AuthFileCheck{Path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		check.Exists = false
		if !os.IsNotExist(err) {
			check.Issues = append(check.Issues, err.Error())
		}
		return check
	}
	check.Exists = true
	check.Readable = true
	var parsed map[string]any
	if err := json.Unmarshal(jsonc.StripComments(data), &parsed); err != nil {
		check.Issues = append(check.Issues, "cannot parse auth JSON: "+err.Error())
		return check
	}
	check.Parseable = true
	provider, ok := parsed[providerID].(map[string]any)
	if !ok {
		return check
	}
	check.HasProvider = true
	if key, ok := provider["key"].(string); ok && strings.TrimSpace(key) != "" {
		check.HasKey = true
	}
	return check
}

func providerConfig(parsed map[string]any, providerID string) (map[string]any, bool) {
	providers, ok := parsed["provider"].(map[string]any)
	if !ok {
		return nil, false
	}
	provider, ok := providers[providerID].(map[string]any)
	return provider, ok
}

func nestedString(root map[string]any, path ...string) (string, bool) {
	var current any = root
	for _, key := range path {
		next, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = next[key]
		if !ok {
			return "", false
		}
	}
	value, ok := current.(string)
	return value, ok
}
