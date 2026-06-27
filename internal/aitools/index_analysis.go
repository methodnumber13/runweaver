package aitools

import (
	"path/filepath"
	"regexp"
	"strings"
)

func languageFor(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	case ".go":
		return "go"
	case ".kt", ".kts":
		return "kotlin"
	case ".java":
		return "java"
	case ".py":
		return "python"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".md":
		return "markdown"
	default:
		return ""
	}
}

func categoryFor(path string) string {
	lower := strings.ToLower(path)
	switch {
	case isTestPath(lower):
		return "test"
	case lower == "src/main.ts" || lower == "src/main.js" || strings.HasSuffix(lower, "/main.go"):
		return "entrypoint"
	case strings.HasSuffix(lower, ".module.ts") || strings.HasSuffix(lower, ".module.js") || strings.HasSuffix(lower, "module.kt") || strings.HasSuffix(lower, "module.java"):
		return "module"
	case isRouteSurfacePath(lower):
		return "route"
	case strings.Contains(lower, "/dto/") || strings.HasSuffix(lower, ".dto.ts") || strings.HasSuffix(lower, ".dto.js") || strings.Contains(lower, "/types/") || strings.HasSuffix(lower, ".types.ts"):
		return "contract"
	case strings.Contains(lower, "/pages/") || strings.Contains(lower, "/components/") || strings.Contains(lower, "/containers/"):
		return "ui"
	case strings.HasSuffix(lower, ".service.ts") || strings.HasSuffix(lower, ".service.js") || strings.Contains(lower, "/services/"):
		return "service"
	case lower == "prisma/schema.prisma" || strings.HasPrefix(lower, "prisma/migrations/") || strings.HasSuffix(lower, ".repository.ts") || strings.Contains(lower, "/repository") || strings.Contains(lower, "repository"):
		return "persistence"
	case isConfigPath(lower):
		return "config"
	default:
		return "source"
	}
}

func isConfigPath(path string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(lower)
	return strings.Contains(lower, "config") ||
		strings.HasPrefix(base, ".") ||
		base == "package.json" ||
		base == "package-lock.json" ||
		base == "go.mod" ||
		base == "dockerfile" ||
		base == "jenkinsfile" ||
		base == "makefile" ||
		base == "nest-cli.json" ||
		strings.HasPrefix(lower, "helm/") ||
		strings.HasPrefix(lower, "test/jest-") ||
		strings.HasPrefix(base, "tsconfig") ||
		strings.HasPrefix(base, "jest.config") ||
		strings.HasPrefix(base, "docker-compose") ||
		base == "sonar-project.properties" ||
		base == "readme.md"
}

func isGeneratedFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "generated") || strings.Contains(lower, ".gen.") || strings.HasSuffix(lower, ".pb.go") || strings.HasSuffix(lower, "package-lock.json") || strings.HasSuffix(lower, "yarn.lock") || strings.HasSuffix(lower, "pnpm-lock.yaml")
}

var (
	jsImportPattern            = regexp.MustCompile(`(?:(?:import|export)\s+.*?\s+from\s+|require\()\s*['"]([^'"]+)['"]`)
	jsExportPattern            = regexp.MustCompile(`export\s+(?:default\s+)?(?:class|function|const|let|var|interface|type)?\s*([A-Za-z0-9_]+)?`)
	jsFunctionPattern          = regexp.MustCompile(`(?:export\s+)?(?:async\s+)?function\s+([A-Za-z0-9_]+)|(?:const|let|var)\s+([A-Za-z0-9_]+)\s*=\s*(?:async\s*)?\(`)
	classPattern               = regexp.MustCompile(`(?:export\s+)?(?:class|interface|type)\s+([A-Za-z0-9_]+)`)
	goFuncPattern              = regexp.MustCompile(`func\s+(?:\([^)]+\)\s*)?([A-Za-z0-9_]+)\s*\(`)
	javaKotlinPattern          = regexp.MustCompile(`(?:class|interface|object|fun)\s+([A-Za-z0-9_]+)`)
	expressRoutePattern        = regexp.MustCompile(`(?:router|app)\.(get|post|put|patch|delete|options|head|all)\s*\(\s*['"]([^'"]*)['"]`)
	mappingPattern             = regexp.MustCompile(`@(Get|Post|Put|Patch|Delete|Request)Mapping(?:\s*\(\s*['"]([^'"]*)['"])?`)
	nestControllerPattern      = regexp.MustCompile(`@Controller\s*\(\s*(?:\{\s*path\s*:\s*)?['"]([^'"]*)['"]`)
	nestControllerEmptyPattern = regexp.MustCompile(`@Controller\s*\(\s*\)`)
	nestMethodPattern          = regexp.MustCompile(`@(Get|Post|Put|Patch|Delete|Options|Head|All)\s*\(\s*(?:['"]([^'"]*)['"])?`)
)

func importsFromLine(line, language string) []string {
	switch language {
	case "typescript", "javascript":
		matches := jsImportPattern.FindAllStringSubmatch(line, -1)
		var out []string
		for _, match := range matches {
			if len(match) > 1 {
				out = append(out, match[1])
			}
		}
		return out
	case "go", "kotlin", "java", "python":
		if strings.HasPrefix(line, "import ") {
			return []string{strings.Trim(strings.TrimPrefix(line, "import "), `" ;`)}
		}
	}
	return nil
}

func exportsFromLine(line, language string) []string {
	if language != "typescript" && language != "javascript" {
		return nil
	}
	match := jsExportPattern.FindStringSubmatch(line)
	if len(match) > 1 && match[1] != "" {
		return []string{match[1]}
	}
	return nil
}

func symbolsFromLine(line, language, path string, lineNo int) []SymbolInfo {
	var out []SymbolInfo
	switch language {
	case "typescript", "javascript":
		for _, match := range jsFunctionPattern.FindAllStringSubmatch(line, -1) {
			name := firstNonEmpty(match[1:]...)
			if name != "" {
				out = append(out, SymbolInfo{Kind: "function", Name: name, Path: path, Line: lineNo})
			}
		}
		if match := classPattern.FindStringSubmatch(line); len(match) > 1 {
			out = append(out, SymbolInfo{Kind: "type", Name: match[1], Path: path, Line: lineNo})
		}
	case "go":
		if match := goFuncPattern.FindStringSubmatch(line); len(match) > 1 {
			out = append(out, SymbolInfo{Kind: "function", Name: match[1], Path: path, Line: lineNo})
		}
	case "kotlin", "java":
		if match := javaKotlinPattern.FindStringSubmatch(line); len(match) > 1 {
			out = append(out, SymbolInfo{Kind: "symbol", Name: match[1], Path: path, Line: lineNo})
		}
	}
	return out
}

func routesFromLine(line, language, path string, lineNo int) []RouteInfo {
	var out []RouteInfo
	if language == "typescript" || language == "javascript" {
		for _, match := range expressRoutePattern.FindAllStringSubmatch(line, -1) {
			out = append(out, RouteInfo{Method: strings.ToUpper(match[1]), Path: match[2], File: path, Line: lineNo})
		}
	}
	if language == "kotlin" || language == "java" {
		if match := mappingPattern.FindStringSubmatch(line); len(match) > 1 {
			routePath := ""
			if len(match) > 2 {
				routePath = match[2]
			}
			out = append(out, RouteInfo{Method: strings.ToUpper(strings.TrimSuffix(match[1], "Mapping")), Path: routePath, File: path, Line: lineNo})
		}
	}
	return out
}

func nestControllerPathFromLine(line string) (string, bool) {
	if nestControllerEmptyPattern.MatchString(line) {
		return "", true
	}
	if match := nestControllerPattern.FindStringSubmatch(line); len(match) > 1 {
		return strings.Trim(match[1], "/"), true
	}
	return "", false
}

func nestRoutesFromLine(line, controllerPath, language, path string, lineNo int) []RouteInfo {
	if language != "typescript" && language != "javascript" {
		return nil
	}
	match := nestMethodPattern.FindStringSubmatch(line)
	if len(match) == 0 {
		return nil
	}
	method := strings.ToUpper(match[1])
	routePath := ""
	if len(match) > 2 {
		routePath = match[2]
	}
	fullPath := joinRoutePath(controllerPath, routePath)
	return []RouteInfo{{Method: method, Path: fullPath, File: path, Line: lineNo}}
}

func joinRoutePath(parts ...string) string {
	var cleaned []string
	for _, part := range parts {
		part = strings.Trim(part, "/")
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return strings.Join(cleaned, "/")
}

func firstNonEmpty(items ...string) string {
	for _, item := range items {
		if item != "" {
			return item
		}
	}
	return ""
}
