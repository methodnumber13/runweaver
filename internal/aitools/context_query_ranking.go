package aitools

import (
	"path/filepath"
	"sort"
	"strings"
)

func rankedContextFiles(index RepoIndex, tokens map[string]bool, scope contextTaskScope, includeGenerated bool, limit int) []ContextFileHit {
	var hits []ContextFileHit
	for _, file := range index.Files {
		if file.Generated && !includeGenerated {
			continue
		}
		if contextFileExcludedByScope(file, tokens, scope) {
			continue
		}
		score, rationale := scoreContextText(file.Path+" "+file.Category+" "+file.Language, tokens)
		pathScore, pathRationale := scoreContextFilePath(file, tokens)
		score += pathScore
		rationale = append(rationale, pathRationale...)
		if score > 0 {
			score += contextCategoryBoost(file.Category)
			adjustment, adjustmentRationale := contextFileScopeAdjustment(file, tokens, scope)
			score += adjustment
			rationale = append(rationale, adjustmentRationale...)
		}
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

func rankedContextSymbols(symbols []SymbolInfo, tokens map[string]bool, selectedFiles map[string]bool, scope contextTaskScope, limit int) []SymbolInfo {
	type symbolHit struct {
		symbol SymbolInfo
		score  int
	}
	var hits []symbolHit
	for _, symbol := range symbols {
		if contextPathExcludedByScope(symbol.Path, tokens, scope) {
			continue
		}
		if !selectedFiles[symbol.Path] && contextPathLooksGenerated(symbol.Path) {
			continue
		}
		score, rationale := scoreContextText(symbol.Name+" "+symbol.Kind+" "+symbol.Path, tokens)
		if selectedFiles[symbol.Path] {
			score += 6
		}
		if score > 0 && (selectedFiles[symbol.Path] || hasStrongSymbolContextMatch(score, rationale)) {
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

func hasStrongSymbolContextMatch(score int, rationale []string) bool {
	if score < 20 {
		return false
	}
	return hasSpecificContextMatch(rationale)
}

func rankedContextEdges(edges []IndexEdge, kind string, tokens map[string]bool, selectedFiles map[string]bool, scope contextTaskScope, limit int) []IndexEdge {
	type edgeHit struct {
		edge  IndexEdge
		score int
	}
	var hits []edgeHit
	for _, edge := range edges {
		if edge.Kind != kind {
			continue
		}
		if contextPathExcludedByScope(edge.From, tokens, scope) || contextPathExcludedByScope(edge.To, tokens, scope) {
			continue
		}
		score, rationale := scoreContextText(edge.From+" "+edge.To+" "+edge.Reason, tokens)
		selectedFileEdge := selectedFiles[edge.From] || selectedFiles[edge.To]
		if selectedFileEdge {
			score += 6
		}
		if score > 0 && (selectedFileEdge || hasSpecificContextMatch(rationale)) {
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

func contextPathLooksGenerated(path string) bool {
	clean := strings.ToLower(filepath.ToSlash(path))
	return strings.HasPrefix(clean, "build/") ||
		strings.HasPrefix(clean, "build-secure/") ||
		strings.HasPrefix(clean, "dist/") ||
		strings.HasPrefix(clean, "coverage/") ||
		strings.Contains(clean, "/node_modules/") ||
		strings.Contains(clean, "/dist/") ||
		strings.Contains(clean, "/build/")
}

func scoreContextText(text string, tokens map[string]bool) (int, []string) {
	textTokens := tokenizeSelectionText(text)
	score := 0
	var rationale []string
	for _, token := range sortedSelectionTokens(tokens) {
		if textTokens[token] {
			score += contextTokenWeight(token)
			rationale = append(rationale, "matched token: "+token)
		}
	}
	return score, Unique(rationale)
}

func sortedSelectionTokens(tokens map[string]bool) []string {
	out := make([]string, 0, len(tokens))
	for token := range tokens {
		out = append(out, token)
	}
	sort.Strings(out)
	return out
}

func contextTokenWeight(token string) int {
	switch token {
	case "add", "bug", "bugs", "change", "changes", "fix", "fixes", "fixed", "failing", "failure", "implement", "issue", "issues", "regression", "test", "tests", "update":
		return 3
	default:
		return 10
	}
}

func hasSpecificContextMatch(rationale []string) bool {
	for _, item := range rationale {
		token, ok := strings.CutPrefix(item, "matched token: ")
		if ok && contextTokenWeight(token) > 3 {
			return true
		}
	}
	return false
}

func contextCategoryBoost(category string) int {
	switch category {
	case "test":
		return 5
	case "ui":
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

func scoreContextFilePath(file FileInventoryItem, tokens map[string]bool) (int, []string) {
	base := filepath.Base(file.Path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	compactStem := compactContextToken(stem)
	compactPath := compactContextToken(file.Path)
	score := 0
	var rationale []string
	for _, token := range sortedSelectionTokens(tokens) {
		if token == compactStem {
			score += 40
			rationale = append(rationale, "exact filename match: "+token)
			continue
		}
		if len(token) >= 8 && strings.Contains(compactStem, token) {
			score += 16
			rationale = append(rationale, "filename contains specific token: "+token)
			continue
		}
		if len(token) >= 12 && strings.Contains(compactPath, token) {
			score += 10
			rationale = append(rationale, "path contains specific token: "+token)
		}
	}
	return score, rationale
}

func compactContextToken(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func contextSpecificFileSet(files []ContextFileHit) map[string]bool {
	out := map[string]bool{}
	for _, file := range files {
		if hasSpecificContextMatch(file.Rationale) {
			out[file.Path] = true
		}
	}
	return out
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
