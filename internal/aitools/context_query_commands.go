package aitools

import (
	"sort"
	"strings"
)

var contextBackendCommandMarkers = []string{
	" backend",
	"cd backend",
	" server",
	"cd server",
	"src/api",
	"/api/",
	"-api",
	"api:",
	"database",
	"migration",
	"worker",
}

func contextCommands(commands []string, tokens map[string]bool, scope contextTaskScope, limit int) []string {
	type commandHit struct {
		command string
		score   int
	}
	var hits []commandHit
	var frontendHits []commandHit
	for _, command := range commands {
		if scope.FrontendOnly && scope.BackendForbidden && contextCommandLooksBackend(command) {
			continue
		}
		score, _ := scoreContextText(command, tokens)
		if score > 0 {
			hit := commandHit{command: command, score: score}
			hits = append(hits, hit)
			if scope.FrontendOnly && commandRelevantToFrontend(command, "") {
				frontendHits = append(frontendHits, hit)
			}
		}
	}
	if scope.FrontendOnly && len(frontendHits) > 0 {
		hits = frontendHits
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

func contextCommandLooksBackend(command string) bool {
	lower := strings.ToLower(command)
	for _, marker := range contextBackendCommandMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
