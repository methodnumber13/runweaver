package aitools

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
)

var backtickPattern = regexp.MustCompile("`([^`]+)`")

// Drift compares generated metadata anchors with the current repository surface.
func Drift(repoPath string, index SurfaceIndex) (DriftReport, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return DriftReport{}, err
	}
	report := DriftReport{
		SchemaVersion: 1,
		GeneratedAt:   Now(),
		RepoRoot:      root,
	}
	runtimeIssues, err := ValidateRuntimeMetadata(root, detectedRuntimeMetadataIDs(root))
	if err != nil {
		return DriftReport{}, err
	}
	report.RuntimeIssues = runtimeIssues

	aiFiles := runtimeMetadataFiles(root)

	combined := strings.Builder{}
	for _, file := range aiFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		text := string(data)
		combined.WriteString(text)
		for _, match := range backtickPattern.FindAllStringSubmatch(text, -1) {
			anchor := strings.TrimSpace(match[1])
			if shouldIgnoreAnchor(anchor) {
				continue
			}
			if !anchorExists(root, anchor) {
				report.StaleAnchors = append(report.StaleAnchors, Anchor{
					File:      rel(root, file),
					Anchor:    anchor,
					Reason:    "anchor path does not exist in current repository",
					Suggested: suggestAnchor(index, anchor),
				})
			}
		}
	}

	allText := combined.String()
	for _, surface := range importantSurfaces(index) {
		if !strings.Contains(allText, surface) {
			report.MissingSurfaces = append(report.MissingSurfaces, surface)
		}
	}
	report.MissingSurfaces = Limit(Unique(report.MissingSurfaces), 80)
	report.Recommendations = recommendations(report)
	return report, nil
}

func runtimeMetadataFiles(root string) []string {
	files := []string{}
	for _, relPath := range runtimeMetadataScanPaths() {
		abs := filepath.Join(root, relPath)
		if !Exists(abs) {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			files = append(files, abs)
			continue
		}
		_ = filepath.WalkDir(abs, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry == nil || entry.IsDir() {
				return nil
			}
			if isRuntimeMetadataFile(entry.Name()) {
				files = append(files, path)
			}
			return nil
		})
	}
	return Unique(files)
}

func runtimeMetadataScanPaths() []string {
	paths := []string{statepath.WorkflowDir, statepath.LegacyOpenCodeWorkflowDir}
	for _, adapter := range RuntimeAdapters() {
		paths = append(paths, adapter.GeneratedPaths()...)
	}
	return Unique(paths)
}

func isRuntimeMetadataFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".json", ".jsonc", ".toml":
		return true
	default:
		return name == "SKILL.md"
	}
}

func shouldIgnoreAnchor(anchor string) bool {
	if anchor == "" {
		return true
	}
	if strings.Contains(anchor, " ") || strings.Contains(anchor, "<") || strings.Contains(anchor, ">") || strings.Contains(anchor, "|") {
		return true
	}
	if strings.Contains(anchor, "[") || strings.Contains(anchor, "]") {
		return true
	}
	if strings.HasPrefix(anchor, "npm ") || strings.HasPrefix(anchor, "node ") || strings.HasPrefix(anchor, "go ") || strings.HasPrefix(anchor, "rg ") || strings.HasPrefix(anchor, "runweaver ") || strings.HasPrefix(anchor, "--") {
		return true
	}
	if strings.HasPrefix(anchor, ".runweaver/tmp/") {
		return true
	}
	if strings.HasSuffix(anchor, "-swarm.json") {
		return true
	}
	switch filepath.Base(anchor) {
	case "plan.json", "checkpoint.json", "todo.md", "current.md", "events.ndjson", "latest.json", "surface-index.json", "drift-report.json", "profile.generated.json", "repo-index.json", "repo-index.compact.json", "repo-context.md", "manifest.json":
		return true
	}
	if strings.Contains(anchor, "*") || strings.Contains(anchor, "{{") || strings.Contains(anchor, "<run-id>") {
		return true
	}
	if strings.HasPrefix(anchor, "http://") || strings.HasPrefix(anchor, "https://") || strings.Contains(anchor, "://") {
		return true
	}
	if !strings.Contains(anchor, "/") && strings.Contains(anchor, ".") && !isLikelyFileAnchorExtension(filepath.Ext(anchor)) {
		return true
	}
	return !(strings.Contains(anchor, "/") || strings.Contains(anchor, "."))
}

func isLikelyFileAnchorExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case ".cjs", ".css", ".env", ".go", ".html", ".java", ".js", ".json", ".jsonc", ".jsx", ".kt", ".md", ".mjs", ".prisma", ".py", ".rs", ".scss", ".sql", ".toml", ".ts", ".tsx", ".txt", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func anchorExists(root, anchor string) bool {
	if strings.HasPrefix(anchor, "../") {
		return Exists(filepath.Clean(filepath.Join(root, anchor)))
	}
	return Exists(filepath.Join(root, anchor))
}

func importantSurfaces(index SurfaceIndex) []string {
	var out []string
	out = append(out, index.EntryPoints...)
	out = append(out, Limit(index.Routes, 20)...)
	out = append(out, Limit(index.Pages, 20)...)
	out = append(out, index.ConfigFiles...)
	out = append(out, Limit(index.SourceDirs, 20)...)
	out = append(out, Limit(index.TestDirs, 20)...)
	return Unique(out)
}

func suggestAnchor(index SurfaceIndex, anchor string) string {
	base := strings.ToLower(filepath.Base(anchor))
	for _, candidate := range importantSurfaces(index) {
		if strings.ToLower(filepath.Base(candidate)) == base {
			return candidate
		}
	}
	for _, candidate := range importantSurfaces(index) {
		if strings.Contains(strings.ToLower(candidate), strings.TrimSuffix(base, filepath.Ext(base))) {
			return candidate
		}
	}
	return ""
}

func recommendations(report DriftReport) []string {
	var out []string
	if len(report.StaleAnchors) > 0 {
		out = append(out, "Update generated agent/skill file anchors from current surface index.")
	}
	if len(report.MissingSurfaces) > 0 {
		out = append(out, "Add or update repo-specific agents/skills for missing important surfaces.")
	}
	if len(report.RuntimeIssues) > 0 {
		out = append(out, "Fix runtime-specific metadata drift before relying on generated agents/skills.")
	}
	if len(out) == 0 {
		out = append(out, "No obvious metadata drift detected.")
	}
	return out
}

func detectedRuntimeMetadataIDs(root string) []string {
	var out []string
	if Exists(filepath.Join(root, "opencode.json")) || Exists(filepath.Join(root, "opencode.jsonc")) || Exists(filepath.Join(root, ".opencode")) {
		out = append(out, RuntimeOpenCode)
	}
	if Exists(filepath.Join(root, ".codex")) || Exists(filepath.Join(root, ".agents", "skills")) {
		out = append(out, RuntimeCodex)
	}
	if Exists(filepath.Join(root, "CLAUDE.md")) || Exists(filepath.Join(root, ".claude")) {
		out = append(out, RuntimeClaude)
	}
	return Unique(out)
}
