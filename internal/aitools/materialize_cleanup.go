package aitools

import (
	"os"
	"path/filepath"
	"strings"
)

func cleanupStaleGeneratedMetadata(root string, desiredAgents, desiredSkills map[string]bool) error {
	if err := cleanupGeneratedFiles(filepath.Join(root, ".opencode", "agents"), desiredAgents); err != nil {
		return err
	}
	return cleanupGeneratedSkillFiles(filepath.Join(root, ".opencode", "skills"), desiredSkills)
}

func cleanupGeneratedFiles(dir string, keep map[string]bool) error {
	return cleanupGeneratedFilesWithSuffixes(dir, keep, []string{".md"})
}

func cleanupGeneratedFilesWithSuffixes(dir string, keep map[string]bool, suffixes []string) error {
	if !Exists(dir) {
		return nil
	}
	return filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry == nil || entry.IsDir() || keep[path] {
			return nil
		}
		if !hasAnySuffix(entry.Name(), suffixes) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), generatedMarker) {
			return os.Remove(path)
		}
		return nil
	})
}

func hasAnySuffix(name string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

func cleanupGeneratedSkillFiles(dir string, keep map[string]bool) error {
	if !Exists(dir) {
		return nil
	}
	return filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry == nil || entry.IsDir() || entry.Name() != "SKILL.md" || keep[path] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.Contains(string(data), generatedMarker) {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		_ = os.Remove(filepath.Dir(path))
		return nil
	})
}
