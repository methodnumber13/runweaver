package modelconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckModelConfigFindsGlobalOpenAIProviderAndEnvKey(t *testing.T) {
	root := t.TempDir()
	configHome := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("OPENCODE_CONFIG_CONTENT", "")
	t.Setenv("RUNWEAVER_MODEL_API_KEY", "test-key")
	writeTestFile(t, configHome, "opencode/opencode.jsonc", `{
  // jsonc comments are allowed
  "$schema": "https://opencode.ai/config.json",
  "model": "openai-compatible/coder-model",
  "provider": {
    "openai-compatible": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "OpenAI-compatible",
      "options": {
        "baseURL": "https://llm-provider.example.com/v1",
        "apiKey": "{env:RUNWEAVER_MODEL_API_KEY}"
      },
      "models": {
        "coder-model": { "name": "Qwen Coder" }
      }
    }
  }
}`)

	result, err := CheckModelConfig(root, ModelConfigCheckOptions{ProviderID: "openai-compatible", ModelID: "coder-model", BaseURL: "https://llm-provider.example.com/v1"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ok" {
		t.Fatalf("result = %#v, want ready ok", result)
	}
	if result.Detected.ProviderSource == "" || result.Detected.CredentialSource != "env:RUNWEAVER_MODEL_API_KEY" {
		t.Fatalf("detected = %#v, want provider source and env credential", result.Detected)
	}
}

func TestCheckModelConfigInfersProviderFromConfiguredModelPrefix(t *testing.T) {
	root := t.TempDir()
	configHome := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("OPENCODE_CONFIG_CONTENT", "")
	t.Setenv("RUNWEAVER_MODEL_API_KEY", "test-key")
	writeTestFile(t, configHome, "opencode/opencode.json", `{
  "model": "company-llm/coder-model",
  "provider": {
    "company-llm": {
      "npm": "@ai-sdk/openai-compatible",
      "options": {
        "baseURL": "https://llm-provider.example.com/v1",
        "apiKey": "{env:RUNWEAVER_MODEL_API_KEY}"
      }
    }
  }
}`)

	result, err := CheckModelConfig(root, ModelConfigCheckOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready {
		t.Fatalf("result = %#v, want ready inferred provider", result)
	}
	if result.ProviderID != "company-llm" {
		t.Fatalf("provider ID = %q, want company-llm", result.ProviderID)
	}
	if result.ModelID != "company-llm/coder-model" {
		t.Fatalf("model = %q, want company-llm/coder-model", result.ModelID)
	}
}

func TestCheckModelConfigReportsMissingProviderModelAndCredential(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("OPENCODE_CONFIG_CONTENT", "")
	t.Setenv("RUNWEAVER_MODEL_API_KEY", "")

	result, err := CheckModelConfig(root, ModelConfigCheckOptions{ProviderID: "openai-compatible"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Ready || len(result.Issues) == 0 {
		t.Fatalf("result = %#v, want not ready with issues", result)
	}
	if result.SampleConfig["provider"] == nil {
		t.Fatalf("sample config missing provider: %#v", result.SampleConfig)
	}
}

func TestCheckModelConfigUsesEffectiveOpenCodePrecedence(t *testing.T) {
	root := t.TempDir()
	configHome := filepath.Join(t.TempDir(), "xdg")
	customConfig := filepath.Join(t.TempDir(), "custom.json")
	customDir := filepath.Join(t.TempDir(), "custom-dir")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("OPENCODE_CONFIG", customConfig)
	t.Setenv("OPENCODE_CONFIG_DIR", customDir)
	t.Setenv("RUNWEAVER_MODEL_API_KEY", "test-key")
	t.Setenv("OPENCODE_CONFIG_CONTENT", `{
  "model": "openai-compatible/inline-model",
  "provider": {
    "openai-compatible": {
      "options": {
        "baseURL": "https://inline.example.com/v1"
      }
    }
  }
}`)
	writeTestFile(t, configHome, "opencode/opencode.json", `{
  "model": "openai-compatible/global-model",
  "provider": {
    "openai-compatible": {
      "options": {
        "baseURL": "https://global.example.com/v1"
      }
    }
  }
}`)
	writeTestFile(t, filepath.Dir(customConfig), filepath.Base(customConfig), `{
  "model": "openai-compatible/custom-model"
}`)
	writeTestFile(t, root, "opencode.json", `{
  "model": "openai-compatible/project-model"
}`)
	writeTestFile(t, customDir, "opencode.json", `{
  "model": "openai-compatible/custom-dir-model"
}`)

	result, err := CheckModelConfig(root, ModelConfigCheckOptions{ProviderID: "openai-compatible"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ModelID != "openai-compatible/inline-model" {
		t.Fatalf("model = %q, want inline override", result.ModelID)
	}
	if result.BaseURL != "https://inline.example.com/v1" {
		t.Fatalf("baseURL = %q, want inline override", result.BaseURL)
	}
	if result.Detected.ModelSource != "env:OPENCODE_CONFIG_CONTENT" {
		t.Fatalf("model source = %q, want OPENCODE_CONFIG_CONTENT", result.Detected.ModelSource)
	}
}

func TestCheckModelConfigDoesNotTreatUnsetEnvReferenceAsCredential(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("OPENCODE_CONFIG_CONTENT", "")
	t.Setenv("RUNWEAVER_MODEL_API_KEY", "")
	writeTestFile(t, root, "opencode.json", `{
  "model": "openai-compatible/coder-model",
  "provider": {
    "openai-compatible": {
      "options": {
        "baseURL": "https://llm-provider.example.com/v1",
        "apiKey": "{env:RUNWEAVER_MODEL_API_KEY}"
      }
    }
  }
}`)

	result, err := CheckModelConfig(root, ModelConfigCheckOptions{ProviderID: "openai-compatible"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Detected.HasCredential {
		t.Fatalf("detected credential = true, want false when env reference is unset")
	}
	if result.Ready {
		t.Fatalf("ready = true, want false without resolved credential")
	}
	if len(result.ProjectConfig.Issues) == 0 {
		t.Fatalf("project config issues = empty, want unset env reference warning")
	}
}

func TestCheckModelConfigResolvesFileCredentialReference(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("OPENCODE_CONFIG_CONTENT", "")
	t.Setenv("RUNWEAVER_MODEL_API_KEY", "")
	writeTestFile(t, root, "secret.txt", "test-key\n")
	writeTestFile(t, root, "opencode.json", `{
  "model": "openai-compatible/coder-model",
  "provider": {
    "openai-compatible": {
      "options": {
        "baseURL": "https://llm-provider.example.com/v1",
        "apiKey": "{file:secret.txt}"
      }
    }
  }
}`)

	result, err := CheckModelConfig(root, ModelConfigCheckOptions{ProviderID: "openai-compatible"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready {
		t.Fatalf("result = %#v, want ready with file credential", result)
	}
	if result.ProjectConfig.APIKeySource != "file" || result.ProjectConfig.HasLiteralAPIKey {
		t.Fatalf("project config = %#v, want file credential source", result.ProjectConfig)
	}
}

func TestCheckModelConfigUsesAuthFileCredential(t *testing.T) {
	root := t.TempDir()
	dataHome := filepath.Join(t.TempDir(), "data")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("OPENCODE_CONFIG_CONTENT", "")
	t.Setenv("RUNWEAVER_MODEL_API_KEY", "")
	writeTestFile(t, root, "opencode.json", `{
  "model": "openai-compatible/coder-model",
  "provider": {
    "openai-compatible": {
      "options": {
        "baseURL": "https://llm-provider.example.com/v1"
      }
    }
  }
}`)
	writeTestFile(t, dataHome, "opencode/auth.json", `{
  "openai-compatible": {
    "key": "test-key"
  }
}`)

	result, err := CheckModelConfig(root, ModelConfigCheckOptions{ProviderID: "openai-compatible"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || !result.Detected.HasCredential {
		t.Fatalf("result = %#v, want ready with auth file credential", result)
	}
	if result.Detected.CredentialSource == "" || len(result.AuthFiles) == 0 || !result.AuthFiles[0].HasKey {
		t.Fatalf("auth files = %#v detected=%#v, want auth credential", result.AuthFiles, result.Detected)
	}
}

func writeTestFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
