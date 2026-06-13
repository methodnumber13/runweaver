package statepath

import "testing"

func TestStatePathsAreRepoLocalAndStable(t *testing.T) {
	root := "/repo"

	if got := TmpPath(root, "index", "repo-index.json"); got != "/repo/.runweaver/tmp/index/repo-index.json" {
		t.Fatalf("TmpPath() = %q", got)
	}
	if got := TmpRel("swarm-runs", "latest.json"); got != ".runweaver/tmp/swarm-runs/latest.json" {
		t.Fatalf("TmpRel() = %q", got)
	}
	if got := WorkflowPath(root, "feature.json"); got != "/repo/.runweaver/workflows/feature.json" {
		t.Fatalf("WorkflowPath() = %q", got)
	}
	if got := WorkflowLatestPath(root); got != "/repo/.runweaver/tmp/swarm-runs/latest.json" {
		t.Fatalf("WorkflowLatestPath() = %q", got)
	}
	if got := IndexRootPath(root); got != "/repo/.runweaver/tmp/index" {
		t.Fatalf("IndexRootPath() = %q", got)
	}
	if got := CacheRootPath(root); got != "/repo/.runweaver/tmp/cache" {
		t.Fatalf("CacheRootPath() = %q", got)
	}
}
