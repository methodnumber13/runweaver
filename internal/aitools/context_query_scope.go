package aitools

import (
	"path/filepath"
	"strings"
)

var (
	contextFrontendTaskTokens       = []string{"card", "client", "component", "dialog", "frontend", "page", "screen", "ui", "view", "widget"}
	contextBackendTaskTokens        = []string{"api", "backend", "controller", "database", "db", "endpoint", "resolver", "route", "server"}
	contextImplementationTaskTokens = []string{
		"add", "build", "card", "component", "create", "dialog", "feature", "frontend", "implement", "implementation", "launch", "launcher", "new", "page", "screen", "ui", "variant", "variants", "view",
	}
	contextDocumentationTaskTokens = []string{"adr", "docs", "documentation", "markdown", "readme", "task", "tasks"}
	contextTestTaskTokens          = []string{"coverage", "e2e", "spec", "test", "tests"}
	contextEnvironmentTaskTokens   = []string{"env", "environment"}
	contextBackendPathSegments     = []string{"api", "backend", "controller", "controllers", "database", "databases", "endpoint", "endpoints", "handler", "handlers", "middleware", "resolver", "resolvers", "route", "routes", "runtime", "server", "worker", "workers"}
	contextBackendFileStems        = []string{"controller", "handler", "middleware", "resolver", "route", "routes", "server", "worker"}
)

type contextTaskScope struct {
	FrontendOnly     bool
	BackendForbidden bool
	SuppressedTokens map[string]bool
}

func contextFileExcludedByScope(file FileInventoryItem, tokens map[string]bool, scope contextTaskScope) bool {
	return contextPathExcludedByScopeWithCategory(file.Path, file.Category, tokens, scope)
}

func contextPathExcludedByScope(path string, tokens map[string]bool, scope contextTaskScope) bool {
	return contextPathExcludedByScopeWithCategory(path, "", tokens, scope)
}

func contextPathExcludedByScopeWithCategory(path, category string, tokens map[string]bool, scope contextTaskScope) bool {
	if !scope.FrontendOnly || !scope.BackendForbidden {
		return false
	}
	file := FileInventoryItem{Path: path, Category: category}
	if contextFileLooksBackend(file) {
		return true
	}
	return !contextTaskWantsEnvironment(tokens) && contextFileLooksEnvironmentConfig(file)
}

func contextFileScopeAdjustment(file FileInventoryItem, tokens map[string]bool, scope contextTaskScope) (int, []string) {
	path := strings.ToLower(filepath.ToSlash(file.Path))
	wantsDocs := contextTaskWantsDocumentation(tokens)
	implementationTask := contextTaskWantsImplementation(tokens)
	score := 0
	var rationale []string
	if scope.FrontendOnly && contextFileLooksBackend(file) {
		score -= 80
		rationale = append(rationale, "penalized backend/server surface for frontend-only task")
	}
	if scope.FrontendOnly && !contextTaskWantsEnvironment(tokens) && contextFileLooksEnvironmentConfig(file) {
		score -= 60
		rationale = append(rationale, "penalized env/runtime config for frontend-only task")
	}
	if implementationTask {
		if file.Category != "test" && (file.Category == "ui" || pathLooksFrontendSurface(path)) {
			score += 8
			rationale = append(rationale, "implementation task prefers source UI surface")
		}
		if file.Category == "test" {
			if contextTaskWantsTests(tokens) {
				score -= 10
				rationale = append(rationale, "implementation task keeps verification files below source targets")
			} else {
				score -= 8
				rationale = append(rationale, "implementation task keeps tests below source targets")
			}
		}
	}
	if !wantsDocs {
		if strings.HasPrefix(path, "docs/tasks/") {
			score -= 22
			rationale = append(rationale, "penalized docs/tasks for implementation context")
		} else if strings.HasPrefix(path, "docs/") || strings.HasSuffix(path, ".md") {
			score -= 10
			rationale = append(rationale, "penalized documentation for non-documentation task")
		}
	}
	return score, rationale
}

func inferContextTaskScope(task string, tokens map[string]bool) contextTaskScope {
	frontend := hasAnySelectionToken(tokens, contextFrontendTaskTokens)
	backend := hasAnySelectionToken(tokens, contextBackendTaskTokens)
	negatesBackend := taskForbidsBackendChanges(task)
	suppressedTokens := taskNegatedBackendTargetTokens(task)
	if negatesBackend && len(suppressedTokens) == 0 {
		suppressedTokens["backend"] = true
	}
	return contextTaskScope{
		FrontendOnly:     frontend && (negatesBackend || !backend),
		BackendForbidden: negatesBackend,
		SuppressedTokens: suppressedTokens,
	}
}

func contextScoringTokens(tokens map[string]bool, scope contextTaskScope) map[string]bool {
	if len(scope.SuppressedTokens) == 0 {
		return tokens
	}
	out := make(map[string]bool, len(tokens))
	for token, enabled := range tokens {
		if enabled && !scope.SuppressedTokens[token] {
			out[token] = true
		}
	}
	return out
}

func contextFileLooksBackend(file FileInventoryItem) bool {
	path := strings.ToLower(filepath.ToSlash(file.Path))
	category := strings.ToLower(file.Category)
	if contextPathUnderFrontendRoot(path) {
		return false
	}
	if category == "route" || category == "service" || category == "contract" || category == "entrypoint" {
		return true
	}
	for _, segment := range strings.Split(path, "/") {
		if contextPathSegmentLooksBackend(segment) {
			return true
		}
	}
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if containsExact(contextBackendFileStems, strings.ToLower(stem)) {
		return true
	}
	return false
}

func contextPathUnderFrontendRoot(path string) bool {
	path = strings.ToLower(filepath.ToSlash(path))
	return strings.HasPrefix(path, "frontend/") || strings.Contains(path, "/frontend/")
}

func contextPathSegmentLooksBackend(segment string) bool {
	segment = strings.TrimSpace(strings.ToLower(segment))
	if segment == "" {
		return false
	}
	for _, token := range strings.Fields(normalizeTaskConstraintText(segment)) {
		if containsExact(contextBackendPathSegments, token) {
			return true
		}
	}
	return containsExact(contextBackendPathSegments, segment)
}

func contextFileLooksEnvironmentConfig(file FileInventoryItem) bool {
	path := strings.ToLower(filepath.ToSlash(file.Path))
	base := filepath.Base(path)
	if strings.Contains(path, ".env") || base == "env" || strings.HasPrefix(base, "env.") {
		return true
	}
	return strings.Contains(path, "/config/env.") ||
		strings.Contains(path, "/runtime/") ||
		strings.Contains(path, "/worker/") ||
		strings.Contains(path, "/workers/")
}

func contextTaskWantsImplementation(tokens map[string]bool) bool {
	return hasAnySelectionToken(tokens, contextImplementationTaskTokens)
}

func contextTaskWantsDocumentation(tokens map[string]bool) bool {
	return hasAnySelectionToken(tokens, contextDocumentationTaskTokens)
}

func contextTaskWantsTests(tokens map[string]bool) bool {
	return hasAnySelectionToken(tokens, contextTestTaskTokens)
}

func contextTaskWantsEnvironment(tokens map[string]bool) bool {
	return hasAnySelectionToken(tokens, contextEnvironmentTaskTokens)
}
