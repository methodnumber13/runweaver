package statepath

import "path/filepath"

const (
	// RootDir is the repo-local RunWeaver metadata root.
	RootDir = ".runweaver"
	// TmpDir is the repo-local generated temporary state directory.
	TmpDir = ".runweaver/tmp"
	// WorkflowDir is the canonical workflow template directory.
	WorkflowDir = ".runweaver/workflows"
	// LegacyOpenCodeWorkflowDir is read only for old workflow templates.
	LegacyOpenCodeWorkflowDir = ".opencode/workflows"
)

// TmpPath builds an absolute path under .runweaver/tmp.
func TmpPath(root string, parts ...string) string {
	all := append([]string{root, RootDir, "tmp"}, parts...)
	return filepath.Join(all...)
}

// TmpRel builds a repo-relative path under .runweaver/tmp.
func TmpRel(parts ...string) string {
	all := append([]string{RootDir, "tmp"}, parts...)
	return filepath.ToSlash(filepath.Join(all...))
}

// WorkflowPath builds an absolute path to a canonical workflow template.
func WorkflowPath(root, name string) string {
	return filepath.Join(root, RootDir, "workflows", name)
}

// LegacyOpenCodeWorkflowPath builds an absolute path to a legacy workflow template.
func LegacyOpenCodeWorkflowPath(root, name string) string {
	return filepath.Join(root, ".opencode", "workflows", name)
}

// WorkflowRunsRoot returns the canonical workflow run state directory.
func WorkflowRunsRoot(root string) string {
	return TmpPath(root, "swarm-runs")
}

// WorkflowLatestPath returns the canonical latest workflow pointer path.
func WorkflowLatestPath(root string) string {
	return filepath.Join(WorkflowRunsRoot(root), "latest.json")
}

// IndexRootPath returns the canonical index artifact directory.
func IndexRootPath(root string) string {
	return TmpPath(root, "index")
}

// CacheRootPath returns the content-hash cache directory.
func CacheRootPath(root string) string {
	return TmpPath(root, "cache")
}
