package runtimeenv

import "testing"

func TestHostFromBaseURL(t *testing.T) {
	if got := HostFromBaseURL("https://llm.example.com/v1"); got != "llm.example.com" {
		t.Fatalf("HostFromBaseURL() = %q", got)
	}
	if got := HostFromBaseURL("not a url"); got != "" {
		t.Fatalf("HostFromBaseURL(invalid) = %q", got)
	}
}

func TestWithNoProxyHostsMergesExistingValues(t *testing.T) {
	env := WithNoProxyHosts([]string{"NO_PROXY=localhost", "OTHER=value"}, []string{"llm.example.com", "localhost"})
	if len(env) != 3 {
		t.Fatalf("env = %#v", env)
	}
	if env[0] != "NO_PROXY=localhost,llm.example.com" {
		t.Fatalf("NO_PROXY = %q", env[0])
	}
	if env[2] != "no_proxy=llm.example.com,localhost" {
		t.Fatalf("no_proxy = %q", env[2])
	}
}
