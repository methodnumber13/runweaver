package runtimeenv

import (
	"net/url"
	"os"
	"strings"

	"github.com/methodnumber13/runweaver/internal/aitools/modelconfig"
)

// OpenCodeProviderEnv returns an environment with provider hosts added to NO_PROXY.
func OpenCodeProviderEnv(root, providerID string) []string {
	env := os.Environ()
	check, err := modelconfig.CheckModelConfig(root, modelconfig.ModelConfigCheckOptions{ProviderID: providerID})
	if err != nil {
		return env
	}
	host := HostFromBaseURL(check.BaseURL)
	if host == "" {
		return env
	}
	return WithNoProxyHosts(env, []string{host, "localhost", "127.0.0.1"})
}

// HostFromBaseURL extracts a hostname from a provider base URL.
func HostFromBaseURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Hostname())
}

// WithNoProxyHosts merges hosts into NO_PROXY and no_proxy entries.
func WithNoProxyHosts(env []string, hosts []string) []string {
	if len(hosts) == 0 {
		return env
	}
	normalizedHosts := uniqueNoProxyValues(hosts)
	return upsertEnv(upsertEnv(env, "NO_PROXY", mergeNoProxyValues(EnvValue(env, "NO_PROXY"), normalizedHosts)), "no_proxy", mergeNoProxyValues(EnvValue(env, "no_proxy"), normalizedHosts))
}

// EnvValue returns a key's value from a KEY=value environment slice.
func EnvValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

func upsertEnv(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	updated := false
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			if !updated {
				out = append(out, prefix+value)
				updated = true
			}
			continue
		}
		out = append(out, item)
	}
	if !updated {
		out = append(out, prefix+value)
	}
	return out
}

func mergeNoProxyValues(existing string, hosts []string) string {
	values := splitNoProxy(existing)
	values = append(values, hosts...)
	return strings.Join(uniqueNoProxyValues(values), ",")
}

func splitNoProxy(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func uniqueNoProxyValues(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
