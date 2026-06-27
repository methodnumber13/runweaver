package aitools

import (
	"path/filepath"
	"strings"
)

func domainDescription(name string) string {
	switch name {
	case "auth":
		return "Owns auth guards, identity token introspection/userinfo, x-api-key behavior, decorators, DTOs, and auth tests"
	case "scm":
		return "Owns source control adapter request/response contracts, groups, member roles, variables, triggers, and integration tests"
	case "kubernetes":
		return "Owns Kubernetes cluster endpoints, product/microfront cluster contracts, DTOs, and tests"
	case "templates":
		return "Owns template creation orchestration across catalog, platform, deployment, source control, persistence, and messaging services"
	case "object-storage":
		return "Owns object storage sharing, bucket policy, static keys, private-key handling, and tests"
	case "devops":
		return "Owns platform automation endpoints for secret storage, package repositories, pipeline notification, and messaging side effects"
	case "catalog-service":
		return "Owns catalog product/component/team contracts, pagination utilities, and infrastructure mapping"
	case "deployment-service":
		return "Owns deployment project configuration, request polling, secret/resource extraction, and external API contracts"
	case "messaging":
		return "Owns messaging channel lookup, batching, filtering, and API contract tests"
	case "mail":
		return "Owns mail module configuration, notification recipients, and MailerService calls"
	case "cache":
		return "Owns cache module, Redis-backed cache behavior, and cache test endpoint"
	case "observability":
		return "Owns observability setup, interceptor/filter, tracing test endpoints, and telemetry boundaries"
	default:
		return "Owns the " + name + " domain surface"
	}
}

func domainKind(name string) string {
	switch name {
	case "auth":
		return "auth"
	case "scm", "kubernetes", "devops", "catalog-service", "deployment-service", "messaging", "mail", "observability", "object-storage", "cache":
		return "external-integration"
	case "templates":
		return "orchestration"
	case "prisma":
		return "persistence"
	default:
		return "domain"
	}
}

func isBFFDomain(name string) bool {
	switch name {
	case "auth", "scm", "kubernetes", "templates", "object-storage", "devops", "catalog-service", "deployment-service", "messaging", "mail", "cache", "observability":
		return true
	default:
		return false
	}
}

func domainAgentName(name string) string {
	switch name {
	case "auth":
		return "identity-access-agent"
	case "templates":
		return "templates-orchestration-agent"
	case "object-storage":
		return "object-storage-agent"
	case "cache":
		return "cache-agent"
	case "mail":
		return "mail-notification-agent"
	case "observability":
		return "observability-agent"
	default:
		return sanitizeID(name) + "-integration-agent"
	}
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
