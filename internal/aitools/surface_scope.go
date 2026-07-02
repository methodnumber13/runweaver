package aitools

import (
	"path/filepath"
	"strings"
)

type surfaceScope struct {
	Frontend bool
	Backend  bool
	Test     bool
}

func taskScopedVerificationGuidance() []string {
	return []string{
		"Select verification through runweaver start/context query for the current task scope; do not copy unrelated package commands across surfaces.",
	}
}

func inferSurfaceScope(name, description string, focusFiles []string) surfaceScope {
	text := strings.ToLower(name + " " + description + " " + strings.Join(focusFiles, " "))
	scope := surfaceScope{
		Frontend: textLooksFrontendSurface(text),
		Backend:  textLooksBackendSurface(text),
		Test:     textLooksTestSurface(text),
	}
	for _, file := range focusFiles {
		path := strings.ToLower(filepath.ToSlash(file))
		if pathLooksFrontendSurface(path) {
			scope.Frontend = true
		}
		if pathLooksBackendSurface(path) {
			scope.Backend = true
		}
		if pathLooksTestSurface(path) {
			scope.Test = true
		}
	}
	return scope
}

func skillUsageLabel(skill SkillProfile, repo RepoProfile) string {
	if isGeneralVerificationSkill(skill.Name) {
		return "task-scoped repository verification"
	}
	scope := inferSurfaceScope(skill.Name, skill.Description, skill.FocusFiles)
	switch {
	case scope.Frontend && !scope.Backend:
		return "frontend UI/component"
	case scope.Backend && !scope.Frontend:
		return "backend/API"
	case scope.Test && !scope.Frontend && !scope.Backend:
		return "test/verification"
	case strings.TrimSpace(repo.Kind) != "":
		return repo.Kind
	default:
		return "repository"
	}
}

func isGeneralVerificationSkill(name string) bool {
	switch strings.TrimSpace(name) {
	case "repo-test-tooling", "repo-style-tooling", "repo-quality-gates":
		return true
	default:
		return false
	}
}

func verificationForSkill(skill SkillProfile, repo RepoProfile) []string {
	return verificationForSurface(skill.Verification, inferSurfaceScope(skill.Name, skill.Description, skill.FocusFiles), repo.Kind)
}

func verificationForAgent(agent AgentProfile, repo RepoProfile) []string {
	return verificationForSurface(agent.Verification, inferSurfaceScope(agent.Name, agent.Description, agent.FocusFiles), repo.Kind)
}

func verificationForFocusFiles(commands []string, name, description string, focusFiles []string, repoKind string) []string {
	return verificationForSurface(commands, inferSurfaceScope(name, description, focusFiles), repoKind)
}

func renderedAgentVerification(agent AgentProfile, repo RepoProfile) []string {
	verification := verificationForAgent(agent, repo)
	if len(verification) > 0 {
		return verification
	}
	return taskScopedVerificationGuidance()
}

func verificationForSurface(commands []string, scope surfaceScope, repoKind string) []string {
	if len(commands) == 0 {
		return nil
	}
	if scope.Frontend && !scope.Backend {
		var out []string
		for _, command := range commands {
			if commandRelevantToFrontend(command, repoKind) {
				out = append(out, command)
			}
		}
		if len(out) > 0 {
			return Unique(out)
		}
		for _, command := range commands {
			if !contextCommandLooksBackend(command) && !verificationTargetsNonFrontendSurface(command) {
				out = append(out, command)
			}
		}
		return Unique(out)
	}
	if scope.Backend && !scope.Frontend {
		var out []string
		for _, command := range commands {
			if !commandLooksFrontendOnly(command) {
				out = append(out, command)
			}
		}
		return Unique(out)
	}
	return Unique(commands)
}

func commandRelevantToFrontend(command, repoKind string) bool {
	surface := commandSurface(command)
	if surface != "root" && pathLooksFrontendSurface(surface+"/") {
		return true
	}
	return repoKind == "frontend-app" && !contextCommandLooksBackend(command) && !verificationTargetsNonFrontendSurface(command)
}

func commandLooksFrontendOnly(command string) bool {
	surface := commandSurface(command)
	return surface != "root" && pathLooksFrontendSurface(surface+"/")
}

func textLooksFrontendSurface(text string) bool {
	return strings.Contains(text, "frontend") ||
		strings.Contains(text, "ui") ||
		strings.Contains(text, "component") ||
		strings.Contains(text, "dialog") ||
		strings.Contains(text, "modal") ||
		strings.Contains(text, "page") ||
		strings.Contains(text, "screen") ||
		strings.Contains(text, "view")
}

func textLooksBackendSurface(text string) bool {
	if textLooksClientBoundarySurface(text) {
		return false
	}
	return strings.Contains(text, "backend") ||
		strings.Contains(text, "api route") ||
		strings.Contains(text, "api-route") ||
		strings.Contains(text, "api-security") ||
		strings.Contains(text, "api code") ||
		strings.Contains(text, "api boundary") ||
		strings.Contains(text, "controller") ||
		strings.Contains(text, "endpoint") ||
		strings.Contains(text, "handler") ||
		strings.Contains(text, "middleware") ||
		strings.Contains(text, "runtime") ||
		strings.Contains(text, "server") ||
		strings.Contains(text, "database") ||
		strings.Contains(text, "worker")
}

func textLooksClientBoundarySurface(text string) bool {
	text = strings.ToLower(text)
	for _, phrase := range []string{
		"api-client",
		"api client",
		"frontend api client",
		"client contract",
		"request/response contract",
		"request response contract",
	} {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return strings.Contains(text, "client") &&
		(strings.Contains(text, "contract") ||
			strings.Contains(text, "request") ||
			strings.Contains(text, "response") ||
			strings.Contains(text, "compatibility") ||
			strings.Contains(text, "mapping"))
}

func textLooksTestSurface(text string) bool {
	return strings.Contains(text, "test") ||
		strings.Contains(text, "spec") ||
		strings.Contains(text, "fixture") ||
		strings.Contains(text, "mock") ||
		strings.Contains(text, "verification")
}

func pathLooksFrontendSurface(path string) bool {
	return strings.HasPrefix(path, "frontend/") ||
		strings.Contains(path, "/frontend/") ||
		strings.Contains(path, "/components/") ||
		strings.Contains(path, "/pages/") ||
		strings.Contains(path, "/app/") ||
		strings.Contains(path, "/views/")
}

func pathLooksBackendSurface(path string) bool {
	return strings.HasPrefix(path, "backend/") ||
		strings.Contains(path, "/backend/") ||
		strings.HasPrefix(path, "server/") ||
		strings.Contains(path, "/server/") ||
		strings.Contains(path, "/api/") ||
		strings.Contains(path, "/controllers/") ||
		strings.Contains(path, "/middleware/") ||
		strings.Contains(path, "/workers/")
}

func pathLooksTestSurface(path string) bool {
	base := filepath.Base(path)
	return strings.Contains(base, ".test.") ||
		strings.Contains(base, ".spec.") ||
		strings.Contains(path, "/test/") ||
		strings.Contains(path, "/tests/") ||
		strings.Contains(path, "__tests__")
}
