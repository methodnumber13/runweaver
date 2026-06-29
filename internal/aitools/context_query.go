package aitools

import (
	"fmt"
	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
	"path/filepath"
	"sort"
	"strings"
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
	files := rankedContextFiles(index, tokens, opts.IncludeGenerated, limit)
	selectedFiles := contextFileSet(files)
	routes := rankedContextEdges(index.Edges, "declares-route", tokens, selectedFiles, limit)
	tests := rankedContextEdges(index.Edges, "tests", tokens, selectedFiles, limit)
	symbols := rankedContextSymbols(index.Symbols, tokens, selectedFiles, limit)
	return ContextQueryResult{
		Status:   "success",
		RepoRoot: root,
		Task:     task,
		Index:    rel(root, indexPath),
		Limit:    limit,
		Files:    files,
		Symbols:  symbols,
		Routes:   routes,
		Tests:    tests,
		Commands: contextCommands(index.Tools.RecommendedCommands, tokens, limit),
		Warnings: Limit(index.Warnings, 5),
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

func rankedContextFiles(index RepoIndex, tokens map[string]bool, includeGenerated bool, limit int) []ContextFileHit {
	var hits []ContextFileHit
	for _, file := range index.Files {
		if file.Generated && !includeGenerated {
			continue
		}
		score, rationale := scoreContextText(file.Path+" "+file.Category+" "+file.Language, tokens)
		score += contextCategoryBoost(file.Category)
		if score > 0 {
			hits = append(hits, ContextFileHit{
				Path:      file.Path,
				Category:  file.Category,
				Language:  file.Language,
				Score:     score,
				Generated: file.Generated,
				Rationale: rationale,
			})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].Path < hits[j].Path
		}
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > limit {
		return hits[:limit]
	}
	return hits
}

func rankedContextSymbols(symbols []SymbolInfo, tokens map[string]bool, selectedFiles map[string]bool, limit int) []SymbolInfo {
	type symbolHit struct {
		symbol SymbolInfo
		score  int
	}
	var hits []symbolHit
	for _, symbol := range symbols {
		score, _ := scoreContextText(symbol.Name+" "+symbol.Kind+" "+symbol.Path, tokens)
		if selectedFiles[symbol.Path] {
			score += 6
		}
		if score > 0 {
			hits = append(hits, symbolHit{symbol: symbol, score: score})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score == hits[j].score {
			return hits[i].symbol.Path < hits[j].symbol.Path
		}
		return hits[i].score > hits[j].score
	})
	out := make([]SymbolInfo, 0, minInt(len(hits), limit))
	for _, hit := range hits {
		out = append(out, hit.symbol)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func rankedContextEdges(edges []IndexEdge, kind string, tokens map[string]bool, selectedFiles map[string]bool, limit int) []IndexEdge {
	type edgeHit struct {
		edge  IndexEdge
		score int
	}
	var hits []edgeHit
	for _, edge := range edges {
		if edge.Kind != kind {
			continue
		}
		score, _ := scoreContextText(edge.From+" "+edge.To+" "+edge.Reason, tokens)
		if selectedFiles[edge.From] || selectedFiles[edge.To] {
			score += 6
		}
		if score > 0 {
			hits = append(hits, edgeHit{edge: edge, score: score})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score == hits[j].score {
			return hits[i].edge.From < hits[j].edge.From
		}
		return hits[i].score > hits[j].score
	})
	out := make([]IndexEdge, 0, minInt(len(hits), limit))
	for _, hit := range hits {
		out = append(out, hit.edge)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func contextCommands(commands []string, tokens map[string]bool, limit int) []string {
	type commandHit struct {
		command string
		score   int
	}
	var hits []commandHit
	for _, command := range commands {
		score, _ := scoreContextText(command, tokens)
		if score == 0 && len(hits) < 2 {
			score = 1
		}
		if score > 0 {
			hits = append(hits, commandHit{command: command, score: score})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score == hits[j].score {
			return hits[i].command < hits[j].command
		}
		return hits[i].score > hits[j].score
	})
	out := make([]string, 0, minInt(len(hits), limit))
	for _, hit := range hits {
		out = append(out, hit.command)
		if len(out) >= minInt(limit, 5) {
			break
		}
	}
	return out
}

func scoreContextText(text string, tokens map[string]bool) (int, []string) {
	textTokens := tokenizeSelectionText(text)
	score := 0
	var rationale []string
	for token := range tokens {
		if textTokens[token] {
			score += 10
			rationale = append(rationale, "matched token: "+token)
		}
	}
	return score, Unique(rationale)
}

func contextCategoryBoost(category string) int {
	switch category {
	case "test":
		return 5
	case "route", "service", "contract", "entrypoint":
		return 4
	case "source", "module":
		return 3
	case "config":
		return 2
	default:
		return 0
	}
}

func contextFileSet(files []ContextFileHit) map[string]bool {
	out := map[string]bool{}
	for _, file := range files {
		out[file.Path] = true
	}
	return out
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
