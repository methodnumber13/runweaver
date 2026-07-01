package aitools

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
)

// QueryContext returns a small task-scoped slice of the repo index.
func QueryContext(repoPath string, opts ContextQueryOptions) (ContextQueryResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return ContextQueryResult{}, err
	}
	task := strings.TrimSpace(opts.Task)
	if task == "" {
		return ContextQueryResult{}, fmt.Errorf("context task is required")
	}
	limit := normalizeContextLimit(opts.Limit)
	indexPath := filepath.Join(root, statepath.TmpRel("index", "repo-index.json"))
	var index RepoIndex
	if err := ReadJSON(indexPath, &index); err != nil {
		return ContextQueryResult{}, fmt.Errorf("read repo index for context query: %w", err)
	}
	tokens := tokenizeSelectionText(task)
	scope := inferContextTaskScope(task, tokens)
	scoringTokens := contextScoringTokens(tokens, scope)
	sortedTokens := sortedSelectionTokens(scoringTokens)
	files := rankedContextFiles(index, scoringTokens, scope, opts.IncludeGenerated, limit)
	selectedFiles := contextSpecificFileSet(files)
	routes := rankedContextEdges(index.Edges, "declares-route", scoringTokens, selectedFiles, scope, limit)
	tests := rankedContextEdges(index.Edges, "tests", scoringTokens, selectedFiles, scope, limit)
	symbols := rankedContextSymbols(index.Symbols, scoringTokens, selectedFiles, scope, limit)
	commands := contextCommands(index.Tools.RecommendedCommands, scoringTokens, scope, limit)
	return ContextQueryResult{
		Status:      "success",
		RepoRoot:    root,
		Task:        task,
		Index:       rel(root, indexPath),
		Limit:       limit,
		Tokens:      sortedTokens,
		Explanation: contextExplanation(sortedTokens, files, symbols, routes, tests, commands),
		Files:       files,
		Symbols:     symbols,
		Routes:      routes,
		Tests:       tests,
		Commands:    commands,
		Warnings:    Limit(index.Warnings, 5),
	}, nil
}

func normalizeContextLimit(limit int) int {
	switch {
	case limit <= 0:
		return 12
	case limit < 5:
		return 5
	case limit > 20:
		return 20
	default:
		return limit
	}
}

func contextExplanation(tokens []string, files []ContextFileHit, symbols []SymbolInfo, routes []IndexEdge, tests []IndexEdge, commands []string) []string {
	var out []string
	if len(tokens) > 0 {
		out = append(out, "task tokens: "+strings.Join(tokens, ", "))
	}
	out = append(out, fmt.Sprintf("selected %d file(s), %d symbol(s), %d route edge(s), %d test edge(s)", len(files), len(symbols), len(routes), len(tests)))
	if len(files) == 0 {
		out = append(out, "no indexed file matched task-specific tokens")
	}
	if len(commands) == 0 {
		out = append(out, "no recommended command matched task tokens")
	}
	return out
}
