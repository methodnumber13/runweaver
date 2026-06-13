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
	if containsExact(files, "package.json") {
		pkg := readPackage(filepath.Join(root, "package.json"))
		for name, version := range pkg.Dependencies {
			out = append(out, packageInsight("npm", name, version, "dependencies"))
		}
		for name, version := range pkg.DevDependencies {
			out = append(out, packageInsight("npm", name, version, "devDependencies"))
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

// BuildToolchain derives tooling metadata from stack signals and dependencies.
func BuildToolchain(stack StackInfo, packages []PackageInsight, commands []string) ToolchainInfo {
	tools := ToolchainInfo{
		Languages:           stack.Languages,
		Frameworks:          stack.Frameworks,
		RecommendedCommands: commands,
	}
	if stack.PackageManager != "" {
		tools.PackageManagers = []string{stack.PackageManager}
	}
	for _, pkg := range packages {
		switch pkg.Role {
		case "test-tool":
			tools.TestTools = append(tools.TestTools, pkg.Name)
		case "linter":
			tools.Linters = append(tools.Linters, pkg.Name)
		case "formatter":
			tools.Formatters = append(tools.Formatters, pkg.Name)
		case "build-tool":
			tools.BuildTools = append(tools.BuildTools, pkg.Name)
		case "frontend-framework", "api-framework", "state-management", "orm", "validation":
			tools.Frameworks = append(tools.Frameworks, pkg.Name)
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
	return tools
}

func packageInsight(ecosystem, name, version, scope string) PackageInsight {
	role, action := packageRoleAction(ecosystem, name)
	return PackageInsight{Ecosystem: ecosystem, Name: name, Version: version, Scope: scope, Role: role, Action: action}
}

func packageRoleAction(ecosystem, name string) (string, string) {
	key := strings.ToLower(name)
	switch {
	case key == "react" || key == "next" || key == "vue" || key == "svelte":
		return "frontend-framework", "Create frontend agents for pages, components, state, API clients, accessibility, and visual regression."
	case key == "express" || key == "fastify" || key == "koa" || key == "@nestjs/core" || strings.Contains(key, "spring-boot"):
		return "api-framework", "Create API route/controller agents for validation, auth, contracts, and integration tests."
	case key == "redux" || key == "@reduxjs/toolkit" || strings.Contains(key, "zustand"):
		return "state-management", "Create state-flow skills covering selectors, async actions, cache invalidation, and UI states."
	case key == "jest" || key == "vitest" || key == "mocha" || key == "junit" || key == "kotest" || key == "pytest" || key == "testify":
		return "test-tool", "Use repository-native test commands and create test-quality reviewer skills."
	case key == "playwright" || key == "cypress":
		return "test-tool", "Create E2E workflow guidance and browser-state verification skills."
	case key == "eslint" || key == "ruff" || key == "golangci-lint":
		return "linter", "Use configured lint rules as hard constraints for generated agents."
	case key == "prettier" || key == "black" || key == "gofmt":
		return "formatter", "Use configured formatter before final verification."
	case key == "typescript" || key == "vite" || key == "webpack" || key == "rollup" || key == "gradle" || key == "maven":
		return "build-tool", "Use build/typecheck outputs as quality gates."
	case key == "zod" || key == "yup" || strings.Contains(key, "validator"):
		return "validation", "Create validation contract checks for request/response or form schemas."
	case key == "axios" || key == "graphql" || strings.Contains(key, "apollo"):
		return "api-client", "Create client-contract skills linking callers to API/server contracts."
	case key == "prisma" || key == "typeorm" || key == "sequelize" || key == "mongoose" || key == "gorm":
		return "orm", "Create persistence boundary skills covering migrations, queries, and data contracts."
	case key == "gin-gonic/gin" || key == "go-chi/chi" || key == "labstack/echo":
		return "api-framework", "Create Go HTTP route and middleware agents."
	case strings.Contains(key, "grpc"):
		return "rpc-framework", "Create RPC contract and protobuf/schema review agents."
	case key == "cobra" || key == "viper":
		return "cli-config", "Create CLI/config boundary skills."
	default:
		return "library", "Record package as context; create specific agents only if source usage proves it is task-relevant."
	}
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
	for _, name := range []string{"org.springframework.boot", "kotlin", "junit", "kotest", "mockk", "gradle"} {
		if strings.Contains(text, strings.ToLower(name)) {
			out = append(out, packageInsight("gradle", name, "", "build"))
		}
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
