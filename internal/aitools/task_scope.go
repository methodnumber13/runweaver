package aitools

import "strings"

func taskForbidsBackendChanges(task string) bool {
	lower := strings.ToLower(task)
	if taskUsesFrontendOnlyScope(lower) {
		return true
	}
	if taskContainsDirectBackendForbidPhrase(lower) {
		return true
	}
	tokens := strings.Fields(normalizeTaskConstraintText(lower))
	for index := 0; index < len(tokens); index++ {
		windowStart := index + 1
		if tokens[index] == "do" && index+1 < len(tokens) && tokens[index+1] == "not" {
			windowStart = index + 2
		} else if tokens[index] == "don" && index+1 < len(tokens) && tokens[index+1] == "t" {
			windowStart = index + 2
		} else if tokens[index] == "dont" {
			windowStart = index + 1
		} else if !taskConstraintNegationToken(tokens[index]) {
			continue
		}
		if taskConstraintWindowContainsBackendTarget(tokens, windowStart, 12) {
			return true
		}
	}
	return false
}

func taskNegatedBackendTargetTokens(task string) map[string]bool {
	tokens := strings.Fields(normalizeTaskConstraintText(strings.ToLower(task)))
	out := map[string]bool{}
	for index := 0; index < len(tokens); index++ {
		windowStart := index + 1
		if tokens[index] == "do" && index+1 < len(tokens) && tokens[index+1] == "not" {
			windowStart = index + 2
		} else if tokens[index] == "don" && index+1 < len(tokens) && tokens[index+1] == "t" {
			windowStart = index + 2
		} else if tokens[index] == "dont" {
			windowStart = index + 1
		} else if !taskConstraintNegationToken(tokens[index]) {
			continue
		}
		for _, token := range taskConstraintBackendTargetsInWindow(tokens, windowStart, 12) {
			out[token] = true
		}
	}
	return out
}

func taskRequestsClientBoundaryWork(task string, taskTokens map[string]bool) bool {
	if taskForbidsClientBoundaryChanges(task) {
		return false
	}
	clientSignal := hasAnySelectionToken(taskTokens, []string{
		"adapter", "caller", "client", "consumer",
	})
	boundarySignal := hasAnySelectionToken(taskTokens, []string{
		"compatibility", "contract", "contracts", "mapping", "mappings", "request", "requests", "response", "responses",
	})
	return clientSignal && boundarySignal
}

func taskForbidsClientBoundaryChanges(task string) bool {
	lower := strings.ToLower(task)
	for _, clause := range strings.FieldsFunc(lower, taskConstraintClauseSeparator) {
		tokens := strings.Fields(normalizeTaskConstraintText(clause))
		for index := 0; index < len(tokens); index++ {
			windowStart := index + 1
			if tokens[index] == "do" && index+1 < len(tokens) && tokens[index+1] == "not" {
				windowStart = index + 2
			} else if tokens[index] == "don" && index+1 < len(tokens) && tokens[index+1] == "t" {
				windowStart = index + 2
			} else if tokens[index] == "dont" {
				windowStart = index + 1
			} else if !taskConstraintNegationToken(tokens[index]) {
				continue
			}
			if taskConstraintWindowContainsClientBoundaryTarget(tokens, windowStart, 12) {
				return true
			}
		}
	}
	return false
}

func taskConstraintClauseSeparator(r rune) bool {
	switch r {
	case '.', ';', '\n', '\r':
		return true
	default:
		return false
	}
}

func taskUsesFrontendOnlyScope(lowerTask string) bool {
	for _, phrase := range []string{
		"frontend-only",
		"frontend only",
		"ui-only",
		"ui only",
		"client-only",
		"client only",
		"work only in the frontend",
		"work only in frontend",
		"only in the frontend",
		"only in frontend",
		"only touch frontend",
		"modify only frontend",
	} {
		if strings.Contains(lowerTask, phrase) {
			return true
		}
	}
	return false
}

func taskContainsDirectBackendForbidPhrase(lowerTask string) bool {
	for _, phrase := range []string{
		"do not touch backend",
		"do not modify backend",
		"do not change backend",
		"don't touch backend",
		"don't modify backend",
		"don't change backend",
		"dont touch backend",
		"dont modify backend",
		"dont change backend",
		"without backend",
		"no backend",
		"backend should remain untouched",
		"backend must remain untouched",
		"backend remains untouched",
		"backend unchanged",
	} {
		if strings.Contains(lowerTask, phrase) {
			return true
		}
	}
	return false
}

func normalizeTaskConstraintText(value string) string {
	replacer := strings.NewReplacer(
		"/", " ",
		"\\", " ",
		"-", " ",
		"_", " ",
		".", " ",
		",", " ",
		";", " ",
		":", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
		"'", " ",
		"’", " ",
		"`", " ",
		"\"", " ",
	)
	return replacer.Replace(value)
}

func taskConstraintNegationToken(token string) bool {
	switch token {
	case "avoid", "exclude", "excluding", "except", "never", "no", "without":
		return true
	default:
		return false
	}
}

func taskConstraintWindowContainsBackendTarget(tokens []string, start, size int) bool {
	return len(taskConstraintBackendTargetsInWindow(tokens, start, size)) > 0
}

func taskConstraintWindowContainsClientBoundaryTarget(tokens []string, start, size int) bool {
	if start < 0 {
		start = 0
	}
	end := start + size
	if end > len(tokens) {
		end = len(tokens)
	}
	hasRequest := false
	hasResponse := false
	for _, token := range tokens[start:end] {
		switch token {
		case "request", "requests":
			hasRequest = true
		case "response", "responses":
			hasResponse = true
		}
		if taskConstraintExplicitClientBoundaryTargetToken(token) {
			return true
		}
	}
	return hasRequest && hasResponse
}

func taskConstraintBackendTargetsInWindow(tokens []string, start, size int) []string {
	if start < 0 {
		start = 0
	}
	end := start + size
	if end > len(tokens) {
		end = len(tokens)
	}
	var out []string
	for _, token := range tokens[start:end] {
		if taskConstraintBackendTargetToken(token) {
			out = append(out, token)
		}
	}
	return Unique(out)
}

func taskConstraintBackendTargetToken(token string) bool {
	switch token {
	case "api", "backend", "controller", "controllers", "database", "databases", "db", "endpoint", "endpoints", "handler", "handlers", "migration", "migrations", "resolver", "resolvers", "route", "routes", "runtime", "server", "service", "services", "worker", "workers":
		return true
	default:
		return false
	}
}

func taskConstraintExplicitClientBoundaryTargetToken(token string) bool {
	switch token {
	case "adapter", "adapters", "caller", "callers", "client", "clients", "compatibility", "consumer", "consumers", "contract", "contracts", "mapping", "mappings":
		return true
	default:
		return false
	}
}
