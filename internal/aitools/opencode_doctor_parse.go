package aitools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func parseJSONObject(data []byte) (map[string]any, error) {
	data = bytes.TrimSpace(data)
	start := bytes.IndexByte(data, '{')
	end := bytes.LastIndexByte(data, '}')
	if start < 0 || end < start {
		return nil, fmt.Errorf("no JSON object found")
	}
	var out map[string]any
	if err := json.Unmarshal(data[start:end+1], &out); err != nil {
		return nil, err
	}
	return out, nil
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func agentExistsInConfig(config map[string]any, agent string) bool {
	for _, key := range []string{"agent", "agents"} {
		if agents, ok := config[key].(map[string]any); ok {
			if _, ok := agents[agent]; ok {
				return true
			}
		}
	}
	return false
}

func permissionsAllowTools(config map[string]any, tools ...string) bool {
	permission, ok := config["permission"].(map[string]any)
	if !ok {
		return false
	}
	for _, tool := range tools {
		if !permissionAllows(permission[tool]) {
			return false
		}
	}
	return true
}

func agentToolEnabled(agentConfig map[string]any, tool string) bool {
	if tools, ok := agentConfig["tools"].(map[string]any); ok {
		if permissionAllows(tools[tool]) {
			return true
		}
	}
	if permission, ok := agentConfig["permission"].(map[string]any); ok {
		if permissionAllows(permission[tool]) {
			return true
		}
	}
	return false
}

func permissionAllows(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "allow" || typed == "true" || typed == "enabled"
	case map[string]any:
		for _, nested := range typed {
			if permissionAllows(nested) {
				return true
			}
		}
	}
	return false
}

func countSkillFiles(root string) int {
	count := 0
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if entry.Name() == "SKILL.md" {
			count++
		}
		return nil
	})
	return count
}

func firstLines(text string, limit int) []string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	if len(lines) > limit {
		return lines[:limit]
	}
	return lines
}

func compactStrings(items []string) []string {
	var out []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return Unique(out)
}
