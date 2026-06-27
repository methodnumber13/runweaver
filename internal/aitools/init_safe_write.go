package aitools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/methodnumber13/runweaver/internal/aitools/jsonc"
)

const (
	runweaverBlockBegin = "<!-- BEGIN RUNWEAVER -->"
	runweaverBlockEnd   = "<!-- END RUNWEAVER -->"
)

func writeBaselineFile(root, name, content string, opts InitOptions) error {
	path := filepath.Join(root, name)
	switch name {
	case "opencode.json":
		return writeMergedOpenCodeConfig(projectOpenCodeConfigPath(root), content)
	case "AGENTS.md", "CLAUDE.md":
		return writeManagedMarkdownBlock(path, content)
	case ".runweaver/.gitignore", ".opencode/.gitignore":
		return writeMergedLineFile(path, content)
	}
	if isGeneratedMetadataPath(name) {
		return writeGenerated(path, contentWithGeneratedMarker(path, content), opts.Force)
	}
	return writeIfAllowed(path, content, opts.Force)
}

func projectOpenCodeConfigPath(root string) string {
	candidates := []string{
		"opencode.json",
		"opencode.jsonc",
		filepath.Join(".opencode", "opencode.json"),
		filepath.Join(".opencode", "opencode.jsonc"),
	}
	for _, candidate := range candidates {
		path := filepath.Join(root, candidate)
		if Exists(path) {
			return path
		}
	}
	return filepath.Join(root, "opencode.json")
}

func writeMergedOpenCodeConfig(path, baseline string) error {
	if !Exists(path) {
		return writeIfAllowed(path, baseline, true)
	}
	existingData, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var existing map[string]any
	if err := json.Unmarshal(jsonc.StripComments(existingData), &existing); err != nil {
		return fmt.Errorf("parse existing OpenCode config %s before merge: %w", path, err)
	}
	var generated map[string]any
	if err := json.Unmarshal([]byte(baseline), &generated); err != nil {
		return fmt.Errorf("parse RunWeaver OpenCode baseline: %w", err)
	}
	merged := mergeOpenCodeConfig(existing, generated)
	if reflect.DeepEqual(existing, merged) {
		return nil
	}
	if err := writeBackup(path, existingData); err != nil {
		return err
	}
	return WriteJSON(path, merged)
}

func mergeOpenCodeConfig(existing, generated map[string]any) map[string]any {
	merged := copyMap(existing)
	for key, generatedValue := range generated {
		switch key {
		case "instructions":
			merged[key] = mergeStringArrays(asStringArray(existing[key]), asStringArray(generatedValue))
		case "permission", "watcher":
			merged[key] = deepMergeMaps(asMap(existing[key]), asMap(generatedValue))
		default:
			if _, ok := existing[key]; ok && key != "default_agent" {
				continue
			}
			merged[key] = generatedValue
		}
	}
	return merged
}

func writeManagedMarkdownBlock(path, content string) error {
	block := managedMarkdownBlock(content)
	if !Exists(path) {
		return writeIfAllowed(path, content, true)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	current := string(data)
	next := replaceOrAppendManagedBlock(current, block)
	if current == next {
		return nil
	}
	if err := writeBackup(path, data); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(next), 0o644)
}

func managedMarkdownBlock(content string) string {
	content = strings.TrimSpace(content)
	return runweaverBlockBegin + "\n" + content + "\n" + runweaverBlockEnd + "\n"
}

func replaceOrAppendManagedBlock(current, block string) string {
	start := strings.Index(current, runweaverBlockBegin)
	end := strings.Index(current, runweaverBlockEnd)
	if start >= 0 && end >= start {
		end += len(runweaverBlockEnd)
		next := current[:start] + strings.TrimRight(block, "\n") + current[end:]
		return strings.TrimRight(next, "\n") + "\n"
	}
	return strings.TrimRight(current, "\n") + "\n\n" + block
}

func writeMergedLineFile(path, content string) error {
	if !Exists(path) {
		return writeIfAllowed(path, content, true)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	next := mergeLineFile(string(data), content)
	if string(data) == next {
		return nil
	}
	if err := writeBackup(path, data); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(next), 0o644)
}

func mergeLineFile(existing, generated string) string {
	seen := map[string]bool{}
	var lines []string
	for _, line := range strings.Split(existing, "\n") {
		if line == "" {
			continue
		}
		seen[line] = true
		lines = append(lines, line)
	}
	for _, line := range strings.Split(generated, "\n") {
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n") + "\n"
}

func writeBackup(path string, data []byte) error {
	backupPath := path + ".runweaver.bak"
	if Exists(backupPath) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(backupPath, data, 0o644)
}

func contentWithGeneratedMarker(path, content string) string {
	if strings.Contains(content, generatedMarker) {
		return content
	}
	marker := generatedMarker
	if filepath.Ext(path) == ".toml" {
		marker = "# " + generatedMarker
	}
	if strings.HasPrefix(content, "---\n") {
		closeIndex := strings.Index(content[4:], "\n---\n")
		if closeIndex >= 0 {
			insertAt := 4 + closeIndex + len("\n---\n")
			return content[:insertAt] + "\n" + generatedMarker + "\n" + content[insertAt:]
		}
	}
	return marker + "\n" + content
}

func isGeneratedMetadataPath(path string) bool {
	path = filepath.ToSlash(path)
	generatedPrefixes := []string{
		".opencode/agents/",
		".opencode/skills/",
		".codex/agents/",
		".agents/skills/",
		".claude/agents/",
		".claude/skills/",
	}
	for _, prefix := range generatedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func deepMergeMaps(existing, generated map[string]any) map[string]any {
	merged := copyMap(existing)
	for key, generatedValue := range generated {
		if existingValue, ok := merged[key]; ok {
			if existingMap, ok := existingValue.(map[string]any); ok {
				if generatedMap, ok := generatedValue.(map[string]any); ok {
					merged[key] = deepMergeMaps(existingMap, generatedMap)
					continue
				}
			}
			if existingArray := asStringArray(existingValue); len(existingArray) > 0 {
				if generatedArray := asStringArray(generatedValue); len(generatedArray) > 0 {
					merged[key] = mergeStringArrays(existingArray, generatedArray)
					continue
				}
			}
		}
		merged[key] = generatedValue
	}
	return merged
}

func copyMap(value map[string]any) map[string]any {
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

func asMap(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func asStringArray(value any) []string {
	switch typed := value.(type) {
	case string:
		return []string{typed}
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	}
	return nil
}

func mergeStringArrays(existing, generated []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(existing)+len(generated))
	for _, value := range append(existing, generated...) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func hasGeneratedMarker(data []byte) bool {
	return bytes.Contains(data, []byte(generatedMarker))
}
