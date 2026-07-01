package aitools

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DetectPackages identifies dependency roles from supported package manifests.
func DetectPackages(root string, files []string) []PackageInsight {
	packages, _ := DetectPackagesWithWarnings(root, files)
	return packages
}

// DetectPackagesWithWarnings returns dependency roles plus manifest parse warnings.
func DetectPackagesWithWarnings(root string, files []string) ([]PackageInsight, []string) {
	var out []PackageInsight
	var warnings []string
	for _, manifest := range packageJSONManifestPaths(files) {
		pkg := readPackage(filepath.Join(root, manifest))
		scopePrefix := packageScopePrefix(manifest)
		for name, version := range pkg.Dependencies {
			out = append(out, packageInsight("npm", name, version, scopePrefix+"dependencies"))
		}
		for name, version := range pkg.DevDependencies {
			out = append(out, packageInsight("npm", name, version, scopePrefix+"devDependencies"))
		}
	}
	if containsExact(files, "go.mod") {
		out = append(out, goModPackages(filepath.Join(root, "go.mod"))...)
	}
	if containsExact(files, "build.gradle") || containsExact(files, "build.gradle.kts") {
		path := "build.gradle"
		if containsExact(files, "build.gradle.kts") {
			path = "build.gradle.kts"
		}
		out = append(out, gradlePackages(filepath.Join(root, path))...)
	}
	if containsExact(files, "requirements.txt") {
		packages, packageWarnings := requirementsPackages(filepath.Join(root, "requirements.txt"))
		out = append(out, packages...)
		warnings = append(warnings, packageWarnings...)
	}
	return sortPackages(UniquePackages(out)), warnings
}

func packageJSONManifestPaths(files []string) []string {
	var manifests []string
	for _, file := range files {
		if filepath.Base(filepath.ToSlash(file)) == "package.json" {
			manifests = append(manifests, filepath.ToSlash(file))
		}
	}
	return Unique(manifests)
}

func packageScopePrefix(manifest string) string {
	dir := filepath.ToSlash(filepath.Dir(manifest))
	if dir == "." || dir == "" {
		return ""
	}
	return dir + ":"
}

// BuildToolchain derives tooling metadata from stack signals and dependencies.
func BuildToolchain(stack StackInfo, packages []PackageInsight, commands []string) ToolchainInfo {
	tools := ToolchainInfo{
		Languages:                    stack.Languages,
		Frameworks:                   stack.Frameworks,
		RecommendedCommands:          commands,
		RecommendedCommandsBySurface: commandsBySurface(commands),
	}
	if stack.PackageManager != "" {
		tools.PackageManagers = []string{stack.PackageManager}
	}
	tools = addCommandDerivedTooling(tools, commands)
	for _, pkg := range packages {
		switch pkg.Role {
		case "test-tool":
			tools.TestTools = append(tools.TestTools, "package-level test dependency")
		case "linter":
			tools.Linters = append(tools.Linters, "package-level lint dependency")
		case "formatter":
			tools.Formatters = append(tools.Formatters, "package-level format dependency")
		case "build-tool":
			tools.BuildTools = append(tools.BuildTools, "package-level build dependency")
		case "frontend-framework", "api-framework", "state-management", "orm", "validation", "api-client", "rpc-framework":
			tools.Frameworks = append(tools.Frameworks, pkg.Role)
		}
	}
	tools.Languages = Unique(tools.Languages)
	tools.PackageManagers = Unique(tools.PackageManagers)
	tools.Frameworks = Unique(tools.Frameworks)
	tools.TestTools = Unique(tools.TestTools)
	tools.Linters = Unique(tools.Linters)
	tools.Formatters = Unique(tools.Formatters)
	tools.BuildTools = Unique(tools.BuildTools)
	tools.RecommendedCommands = Unique(tools.RecommendedCommands)
	tools.RecommendedCommandsBySurface = uniqueCommandSurfaceMap(tools.RecommendedCommandsBySurface)
	return tools
}

func addCommandDerivedTooling(tools ToolchainInfo, commands []string) ToolchainInfo {
	for _, command := range commands {
		lower := strings.ToLower(command)
		switch {
		case strings.Contains(lower, " test") || strings.Contains(lower, "run test"):
			tools.TestTools = append(tools.TestTools, "repository test command")
		case strings.Contains(lower, " lint") || strings.Contains(lower, "run lint"):
			tools.Linters = append(tools.Linters, "repository lint command")
		case strings.Contains(lower, " format") || strings.Contains(lower, "run format"):
			tools.Formatters = append(tools.Formatters, "repository format command")
		}
		if strings.Contains(lower, " build") ||
			strings.Contains(lower, "run build") ||
			strings.Contains(lower, " typecheck") ||
			strings.Contains(lower, "run typecheck") {
			tools.BuildTools = append(tools.BuildTools, "repository build/typecheck command")
		}
	}
	return tools
}

func commandsBySurface(commands []string) map[string][]string {
	out := map[string][]string{}
	for _, command := range commands {
		surface := commandSurface(command)
		out[surface] = append(out[surface], command)
	}
	return out
}

func commandSurface(command string) string {
	fields := strings.Fields(command)
	for index, field := range fields {
		clean := trimCommandToken(field)
		switch {
		case clean == "cd" && index+1 < len(fields):
			return normalizeCommandSurface(fields[index+1])
		case clean == "--prefix" && index+1 < len(fields):
			return normalizeCommandSurface(fields[index+1])
		case strings.HasPrefix(clean, "--prefix="):
			return normalizeCommandSurface(strings.TrimPrefix(clean, "--prefix="))
		case clean == "--cwd" && index+1 < len(fields):
			return normalizeCommandSurface(fields[index+1])
		case strings.HasPrefix(clean, "--cwd="):
			return normalizeCommandSurface(strings.TrimPrefix(clean, "--cwd="))
		}
	}
	return "root"
}

func normalizeCommandSurface(value string) string {
	value = trimCommandToken(value)
	if value == "" || value == "." || strings.HasPrefix(value, "-") {
		return "root"
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return "root"
	}
	return clean
}

func trimCommandToken(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"'`)
}

func uniqueCommandSurfaceMap(commands map[string][]string) map[string][]string {
	if len(commands) == 0 {
		return nil
	}
	out := map[string][]string{}
	for surface, values := range commands {
		unique := Unique(values)
		if len(unique) > 0 {
			out[surface] = unique
		}
	}
	return out
}

func packageInsight(ecosystem, name, version, scope string) PackageInsight {
	role, action := packageRoleAction(ecosystem, name)
	return PackageInsight{Ecosystem: ecosystem, Name: name, Version: version, Scope: scope, Role: role, Action: action}
}

func packageRoleAction(ecosystem, name string) (string, string) {
	key := strings.ToLower(name)
	switch {
	case packageNameContainsAny(key, "state", "store"):
		return "state-management", "Create state-flow skills covering selectors, async actions, cache invalidation, and UI states."
	case packageNameContainsAny(key, "test", "spec", "mock", "assert"):
		return "test-tool", "Use repository-native test commands and create test-quality reviewer skills."
	case packageNameContainsAny(key, "lint"):
		return "linter", "Use configured lint rules as hard constraints for generated agents."
	case packageNameContainsAny(key, "format", "fmt"):
		return "formatter", "Use configured formatter before final verification."
	case packageNameContainsAny(key, "build", "bundle", "compiler", "typecheck"):
		return "build-tool", "Use build/typecheck outputs as quality gates."
	case packageNameContainsAny(key, "schema", "valid"):
		return "validation", "Create validation contract checks for request/response or form schemas."
	case packageNameContainsAny(key, "client", "http", "request", "fetch", "graphql"):
		return "api-client", "Create client-contract skills linking callers to API/server contracts."
	case packageNameContainsAny(key, "orm", "sql", "mongo", "database", "persistence"):
		return "orm", "Create persistence boundary skills covering migrations, queries, and data contracts."
	case packageNameContainsAny(key, "grpc", "rpc"):
		return "rpc-framework", "Create RPC contract and protobuf/schema review agents."
	case packageNameContainsAny(key, "cli", "config"):
		return "cli-config", "Create CLI/config boundary skills."
	default:
		return "library", "Record package as context; create specific agents only if source usage proves it is task-relevant."
	}
}

func packageNameContainsAny(name string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(name, needle) {
			return true
		}
	}
	return false
}

func goModPackages(path string) []PackageInsight {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []PackageInsight
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "module ") || strings.HasPrefix(line, "go ") || line == "require (" || line == ")" {
			continue
		}
		if strings.HasPrefix(line, "require ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "require "))
		}
		parts := strings.Fields(line)
		if len(parts) >= 1 && strings.Contains(parts[0], "/") {
			version := ""
			if len(parts) >= 2 {
				version = parts[1]
			}
			out = append(out, packageInsight("go", parts[0], version, "require"))
		}
	}
	return out
}

func gradlePackages(path string) []PackageInsight {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	text := strings.ToLower(string(data))
	var out []PackageInsight
	if strings.Contains(text, "dependencies") {
		out = append(out, PackageInsight{
			Ecosystem: "gradle",
			Name:      "gradle-dependencies",
			Scope:     "build",
			Role:      "library",
			Action:    "Record Gradle dependencies as context; create specific agents only if source usage proves them task-relevant.",
		})
	}
	return out
}

func requirementsPackages(path string) ([]PackageInsight, []string) {
	file, err := os.Open(path)
	if err != nil {
		return nil, []string{"cannot read requirements.txt: " + err.Error()}
	}
	defer file.Close()
	var out []PackageInsight
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name := strings.FieldsFunc(line, func(r rune) bool { return r == '=' || r == '<' || r == '>' || r == '~' || r == '!' })[0]
		out = append(out, packageInsight("python", name, "", "requirements"))
	}
	if err := scanner.Err(); err != nil {
		return out, []string{"requirements.txt scan warning: " + err.Error()}
	}
	return out, nil
}

func sortPackages(items []PackageInsight) []PackageInsight {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Ecosystem == items[j].Ecosystem {
			return items[i].Name < items[j].Name
		}
		return items[i].Ecosystem < items[j].Ecosystem
	})
	return items
}

// UniquePackages de-duplicates package insights by ecosystem, name, scope, and role.
func UniquePackages(items []PackageInsight) []PackageInsight {
	seen := map[string]bool{}
	var out []PackageInsight
	for _, item := range items {
		key := item.Ecosystem + ":" + item.Name
		if item.Name == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}
