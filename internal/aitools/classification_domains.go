package aitools

import (
	"path/filepath"
	"strings"
)

func domainDescription(name string) string {
	return "Owns the " + name + " domain surface, related contracts, tests, and verification evidence"
}

func domainKind(name string) string {
	switch name {
	case "auth":
		return "auth"
	default:
		return "domain"
	}
}

func domainAgentName(name string) string {
	return sanitizeID(name) + "-domain-agent"
}

func sanitizeID(value string) string {
	value = strings.ToLower(filepath.ToSlash(value))
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "_", "-")
	var b strings.Builder
	lastDash := false
	for _, ch := range value {
		ok := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if ok {
			b.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
