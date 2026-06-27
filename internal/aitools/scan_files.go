package aitools

import (
	"path/filepath"
	"strings"
)

func configFiles(files []string) []string {
	var out []string
	names := []string{
		"package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock", "go.mod", "go.sum",
		"build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts", "pom.xml",
		"pyproject.toml", "requirements.txt", "Dockerfile", "Jenkinsfile", "Makefile", "README.md",
		"docker-compose.yml", "docker-compose.yaml", "helm/values.yaml", "sonar-project.properties",
		"tsconfig.json", "tsconfig.build.json", "nest-cli.json", "jest.config.js", "jest.config.ts",
		"test/jest-unit.json", "test/jest-e2e.json", ".eslintrc.js", "eslint.config.ts", ".prettierrc", ".gitignore",
		"prisma/schema.prisma",
	}
	for _, name := range names {
		if containsExact(files, name) {
			out = append(out, name)
		}
	}
	return out
}

func sourceDirs(files []string) []string {
	candidates := []string{"src", "app", "routes", "internal", "cmd", "pkg", "src/main/kotlin", "src/main/java", "src/main/resources"}
	return existingDirs(files, candidates)
}

func testDirs(files []string) []string {
	candidates := []string{"test", "tests", "src/test", "src/test/kotlin", "src/test/java", "internal"}
	var out []string
	for _, candidate := range candidates {
		if containsPath(files, candidate) && hasTestBelow(files, candidate) {
			out = append(out, candidate)
		}
	}
	return Unique(out)
}

func entryPoints(files []string, stack StackInfo) []string {
	var out []string
	if stack.Kind == "node-api" {
		for _, file := range []string{"src/main.ts", "src/main.js", "src/app.module.ts", "src/app.module.js", "server.ts", "server.js"} {
			if containsExact(files, file) {
				out = append(out, file)
			}
		}
	}
	for _, file := range files {
		base := filepath.Base(file)
		if base == "main.go" || base == "index.tsx" || base == "app.tsx" || base == "server.ts" || base == "Application.kt" || base == "Application.java" {
			out = append(out, file)
		}
		if stack.Kind != "node-api" && (base == "index.ts" || base == "index.js") {
			out = append(out, file)
		}
	}
	return Limit(Unique(out), 40)
}

func routeFiles(files []string) []string {
	var out []string
	for _, file := range files {
		lower := strings.ToLower(file)
		if isRouteSurfacePath(lower) {
			if isSourceFile(lower) && !isTestPath(lower) {
				out = append(out, file)
			}
		}
	}
	return Limit(Unique(out), 80)
}

func pageFiles(files []string) []string {
	var out []string
	for _, file := range files {
		lower := strings.ToLower(file)
		if strings.Contains(lower, "/pages/") || strings.Contains(lower, "/containers/") || strings.Contains(lower, "/components/") {
			if isSourceFile(lower) {
				out = append(out, file)
			}
		}
	}
	return Limit(Unique(out), 80)
}
