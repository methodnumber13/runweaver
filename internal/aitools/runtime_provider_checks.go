package aitools

import (
	"github.com/methodnumber13/runweaver/internal/aitools/modelconfig"
	"os"
	"strings"
)

func runtimeChecks(kind string, candidates []configCandidate) []RuntimeFileCheck {
	checks := make([]RuntimeFileCheck, 0, len(candidates))
	for _, candidate := range uniqueConfigCandidates(candidates) {
		checks = append(checks, runtimeFileCheck(candidate.Path, candidate.Source, kind))
	}
	return checks
}

func runtimeFileCheck(path, source, kind string) RuntimeFileCheck {
	check := RuntimeFileCheck{Path: modelconfig.ExpandHome(path), Source: source, Kind: kind}
	if strings.HasPrefix(path, "env:") {
		check.Exists = true
		check.Readable = true
		return check
	}
	info, err := os.Stat(check.Path)
	if err != nil {
		if !os.IsNotExist(err) {
			check.Issues = append(check.Issues, err.Error())
		}
		return check
	}
	check.Exists = true
	if info.IsDir() {
		entries, readErr := os.ReadDir(check.Path)
		check.Readable = readErr == nil
		if readErr != nil {
			check.Issues = append(check.Issues, readErr.Error())
		} else if len(entries) == 0 {
			check.Issues = append(check.Issues, "directory is empty")
		}
		return check
	}
	file, err := os.Open(check.Path)
	if err != nil {
		check.Issues = append(check.Issues, err.Error())
		return check
	}
	_ = file.Close()
	check.Readable = true
	return check
}

func hasAnyReadable(checks []RuntimeFileCheck) bool {
	for _, check := range checks {
		if check.Readable {
			return true
		}
	}
	return false
}

func addConfigCandidate(out *[]configCandidate, path, source string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	*out = append(*out, configCandidate{Path: path, Source: source})
}

func uniqueConfigCandidates(items []configCandidate) []configCandidate {
	seen := map[string]bool{}
	out := make([]configCandidate, 0, len(items))
	for _, item := range items {
		item.Path = modelconfig.ExpandHome(item.Path)
		if item.Path == "" || seen[item.Path] {
			continue
		}
		seen[item.Path] = true
		out = append(out, item)
	}
	return out
}

func runtimeCapabilities(id string) map[string]RuntimeFlag {
	adapter, ok := RuntimeAdapterByID(id)
	if !ok {
		return nil
	}
	return adapter.Capabilities()
}

func runtimeGeneratedPaths(id string) []string {
	adapter, ok := RuntimeAdapterByID(id)
	if !ok {
		return nil
	}
	return adapter.GeneratedPaths()
}
