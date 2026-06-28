package aitools

import (
	"os"
	"path/filepath"
	"strings"
)

// RuntimeMetadataIssue describes a runtime-specific metadata consistency problem.
type RuntimeMetadataIssue struct {
	Runtime string `json:"runtime"`
	File    string `json:"file"`
	Reason  string `json:"reason"`
}

// ValidateRuntimeMetadata checks generated runtime surfaces for missing files and cross-runtime drift.
func ValidateRuntimeMetadata(repoPath string, runtimeIDs []string) ([]RuntimeMetadataIssue, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return nil, err
	}
	if len(runtimeIDs) == 0 {
		runtimeIDs = []string{RuntimeOpenCode}
	}
	var issues []RuntimeMetadataIssue
	for _, runtimeID := range runtimeIDs {
		runtimeID = normalizeRuntimeID(runtimeID)
		rules, ok := runtimeMetadataRules(runtimeID)
		if !ok {
			continue
		}
		for _, required := range rules.required {
			if !Exists(filepath.Join(root, required)) {
				issues = append(issues, RuntimeMetadataIssue{
					Runtime: runtimeID,
					File:    required,
					Reason:  "required runtime metadata is missing",
				})
			}
		}
		for _, scanRoot := range rules.scanRoots {
			files, err := metadataFilesUnder(filepath.Join(root, scanRoot))
			if err != nil {
				return nil, err
			}
			for _, file := range files {
				data, err := os.ReadFile(file)
				if err != nil {
					return nil, err
				}
				text := string(data)
				for _, banned := range rules.banned {
					if strings.Contains(text, banned) {
						issues = append(issues, RuntimeMetadataIssue{
							Runtime: runtimeID,
							File:    rel(root, file),
							Reason:  "contains foreign runtime marker " + banned,
						})
					}
				}
			}
		}
	}
	return issues, nil
}

type runtimeMetadataValidationRules struct {
	required  []string
	scanRoots []string
	banned    []string
}

func runtimeMetadataRules(runtimeID string) (runtimeMetadataValidationRules, bool) {
	switch runtimeID {
	case RuntimeOpenCode:
		return runtimeMetadataValidationRules{
			required: []string{
				"opencode.json",
				".opencode/agents/swarm.md",
				".opencode/skills/context-discipline/SKILL.md",
				".opencode/swarm/profile.json",
			},
			scanRoots: []string{".opencode/agents", ".opencode/skills", ".opencode/swarm"},
			banned:    []string{".codex/", ".agents/skills", ".claude/", "codex exec", "claude --print"},
		}, true
	case RuntimeCodex:
		return runtimeMetadataValidationRules{
			required: []string{
				"AGENTS.md",
				".codex/agents/swarm.toml",
				".agents/skills/context-discipline/SKILL.md",
				".codex/runweaver/profile.json",
			},
			scanRoots: []string{".codex/agents", ".agents/skills", ".codex/runweaver"},
			banned:    []string{".opencode/", ".claude/", "opencode run", "claude --print"},
		}, true
	case RuntimeClaude:
		return runtimeMetadataValidationRules{
			required: []string{
				"CLAUDE.md",
				".claude/agents/swarm.md",
				".claude/skills/context-discipline/SKILL.md",
				".claude/runweaver/profile.json",
			},
			scanRoots: []string{".claude/agents", ".claude/skills", ".claude/runweaver"},
			banned:    []string{".opencode/", ".codex/", ".agents/skills", "opencode run", "codex exec"},
		}, true
	default:
		return runtimeMetadataValidationRules{}, false
	}
}

func metadataFilesUnder(root string) ([]string, error) {
	if !Exists(root) {
		return nil, nil
	}
	var files []string
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{root}, nil
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry == nil || entry.IsDir() || !isRuntimeMetadataFile(entry.Name()) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}
