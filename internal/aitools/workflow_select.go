package aitools

import (
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// SelectWorkflow deterministically chooses the best workflow template for a task.
func SelectWorkflow(repoPath string, opts WorkflowSelectOptions) (WorkflowSelectResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return WorkflowSelectResult{}, err
	}
	task := strings.TrimSpace(opts.Task)
	if task == "" {
		return WorkflowSelectResult{}, fmt.Errorf("workflow task is required")
	}
	if strings.TrimSpace(opts.Workflow) != "" {
		candidate, err := explicitWorkflowCandidate(root, opts.Workflow)
		if err != nil {
			return WorkflowSelectResult{}, err
		}
		candidate.Explicit = true
		candidate.Rationale = append(candidate.Rationale, "explicit workflow requested")
		return WorkflowSelectResult{
			Status:       "success",
			RepoRoot:     root,
			Task:         task,
			WorkflowPath: candidate.Path,
			Selected:     candidate,
			Candidates:   []WorkflowSelectionCandidate{candidate},
		}, nil
	}
	workflows, err := loadWorkflowCandidates(root)
	if err != nil {
		return WorkflowSelectResult{}, err
	}
	if len(workflows) == 0 {
		return WorkflowSelectResult{}, fmt.Errorf("no workflow templates found under %s", statepath.WorkflowDir)
	}
	taskTokens := tokenizeSelectionText(task)
	for index := range workflows {
		scoreWorkflowCandidate(&workflows[index], taskTokens)
	}
	sort.SliceStable(workflows, func(i, j int) bool {
		if workflows[i].Score == workflows[j].Score {
			return workflows[i].ID < workflows[j].ID
		}
		return workflows[i].Score > workflows[j].Score
	})
	selected := workflows[0]
	return WorkflowSelectResult{
		Status:       "success",
		RepoRoot:     root,
		Task:         task,
		WorkflowPath: selected.Path,
		Selected:     selected,
		Candidates:   workflows,
	}, nil
}

func explicitWorkflowCandidate(root, workflowPath string) (WorkflowSelectionCandidate, error) {
	path := workflowPath
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	var workflow WorkflowSpec
	if err := ReadJSON(path, &workflow); err != nil {
		return WorkflowSelectionCandidate{}, fmt.Errorf("load workflow %s: %w", workflowPath, err)
	}
	if workflow.ID == "" {
		return WorkflowSelectionCandidate{}, fmt.Errorf("workflow id is required in %s", workflowPath)
	}
	if err := validateWorkflowSpec(workflow); err != nil {
		return WorkflowSelectionCandidate{}, fmt.Errorf("validate workflow %s: %w", workflowPath, err)
	}
	return workflowCandidate(root, path, workflow), nil
}

func loadWorkflowCandidates(root string) ([]WorkflowSelectionCandidate, error) {
	paths, err := workflowTemplatePaths(root)
	if err != nil {
		return nil, err
	}
	candidates := make([]WorkflowSelectionCandidate, 0, len(paths))
	for _, path := range paths {
		var workflow WorkflowSpec
		if err := ReadJSON(path, &workflow); err != nil {
			return nil, fmt.Errorf("load workflow %s: %w", rel(root, path), err)
		}
		if workflow.ID == "" {
			return nil, fmt.Errorf("workflow id is required in %s", rel(root, path))
		}
		if err := validateWorkflowSpec(workflow); err != nil {
			return nil, fmt.Errorf("validate workflow %s: %w", rel(root, path), err)
		}
		candidates = append(candidates, workflowCandidate(root, path, workflow))
	}
	return candidates, nil
}

func workflowTemplatePaths(root string) ([]string, error) {
	seen := map[string]bool{}
	var paths []string
	for _, dir := range []string{
		filepath.Join(root, statepath.WorkflowDir),
		filepath.Join(root, statepath.LegacyOpenCodeWorkflowDir),
	} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read workflow directory %s: %w", rel(root, dir), err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			if seen[path] {
				continue
			}
			seen[path] = true
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func workflowCandidate(root, path string, workflow WorkflowSpec) WorkflowSelectionCandidate {
	return WorkflowSelectionCandidate{
		ID:          workflow.ID,
		Name:        workflow.Name,
		Description: workflow.Description,
		Path:        rel(root, path),
	}
}

func scoreWorkflowCandidate(candidate *WorkflowSelectionCandidate, taskTokens map[string]bool) {
	score := 0
	var rationale []string
	text := strings.Join([]string{candidate.ID, candidate.Name, candidate.Description}, " ")
	workflowTokens := tokenizeSelectionText(text)
	for token := range taskTokens {
		if workflowTokens[token] {
			score += 8
			rationale = append(rationale, "task token matches workflow text: "+token)
		}
	}
	for _, rule := range workflowSelectionRules() {
		if candidate.ID != rule.WorkflowID {
			continue
		}
		for _, keyword := range rule.Keywords {
			if taskTokens[keyword] {
				score += rule.Weight
				rationale = append(rationale, rule.Reason+": "+keyword)
			}
		}
	}
	if candidate.ID == "feature-delivery-swarm" || candidate.ID == "feature-swarm" {
		score += 1
		rationale = append(rationale, "default feature workflow fallback")
	}
	candidate.Score = score
	candidate.Rationale = Unique(rationale)
}

type workflowSelectionRule struct {
	WorkflowID string
	Keywords   []string
	Weight     int
	Reason     string
}

func workflowSelectionRules() []workflowSelectionRule {
	return []workflowSelectionRule{
		{WorkflowID: "bugfix-swarm", Weight: 12, Reason: "bugfix signal", Keywords: []string{
			"bug", "bugfix", "defect", "error", "failure", "failing", "fails", "fix", "hotfix", "regression", "reproduce", "crash", "broken",
		}},
		{WorkflowID: "test-hardening-swarm", Weight: 11, Reason: "test-hardening signal", Keywords: []string{
			"test", "tests", "coverage", "flake", "flaky", "jest", "vitest", "playwright", "e2e", "spec", "regression-test",
		}},
		{WorkflowID: "refactor-swarm", Weight: 11, Reason: "refactor signal", Keywords: []string{
			"refactor", "cleanup", "clean", "split", "extract", "rename", "dry", "solid", "architecture", "restructure",
		}},
		{WorkflowID: "metadata-refresh-swarm", Weight: 11, Reason: "metadata refresh signal", Keywords: []string{
			"metadata", "agent", "agents", "skill", "skills", "stale", "anchor", "anchors", "refresh", "index", "profile", "workflow",
		}},
		{WorkflowID: "repo-onboarding-swarm", Weight: 10, Reason: "onboarding signal", Keywords: []string{
			"onboard", "onboarding", "explore", "study", "analyze", "analyse", "map", "audit", "review", "understand", "overview",
		}},
		{WorkflowID: "feature-delivery-swarm", Weight: 7, Reason: "feature delivery signal", Keywords: []string{
			"add", "build", "create", "deliver", "feature", "implement", "new", "ship", "support",
		}},
	}
}

func tokenizeSelectionText(value string) map[string]bool {
	tokens := map[string]bool{}
	var current strings.Builder
	flush := func() {
		token := strings.TrimSpace(current.String())
		current.Reset()
		if len(token) < 2 {
			return
		}
		tokens[token] = true
	}
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			current.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return tokens
}
