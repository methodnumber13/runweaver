package aitools

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func detectStack(root string, files []string) StackInfo {
	stack := StackInfo{Kind: "generic", Languages: []string{}, Frameworks: []string{}}
	has := func(name string) bool {
		for _, file := range files {
			if file == name {
				return true
			}
		}
		return false
	}

	if has("package.json") {
		stack.Languages = append(stack.Languages, "typescript/javascript")
		stack.PackageManager = packageManager(files)
		pkg := readPackage(filepath.Join(root, "package.json"))
		deps := map[string]bool{}
		for key := range pkg.Dependencies {
			deps[key] = true
		}
		for key := range pkg.DevDependencies {
			deps[key] = true
		}
		if deps["react"] {
			stack.Frameworks = append(stack.Frameworks, "react")
			stack.Kind = "frontend-react"
		}
		if deps["express"] || deps["@nestjs/core"] || deps["koa"] || deps["fastify"] {
			stack.Frameworks = append(stack.Frameworks, "node-api")
			if stack.Kind == "generic" {
				stack.Kind = "node-api"
			}
		}
		if deps["redux"] || deps["@reduxjs/toolkit"] {
			stack.Frameworks = append(stack.Frameworks, "redux")
		}
	}
	if has("go.mod") {
		stack.Languages = append(stack.Languages, "go")
		stack.Frameworks = append(stack.Frameworks, "go")
		stack.Kind = "go-service"
	}
	if has("build.gradle") || has("build.gradle.kts") || has("settings.gradle") || has("settings.gradle.kts") {
		stack.Languages = append(stack.Languages, "kotlin/java")
		stack.Frameworks = append(stack.Frameworks, "gradle")
		stack.Kind = "jvm-service"
		if containsPath(files, "src/main/kotlin") {
			stack.Frameworks = append(stack.Frameworks, "kotlin")
		}
	}
	if has("pom.xml") {
		stack.Languages = append(stack.Languages, "java")
		stack.Frameworks = append(stack.Frameworks, "maven")
		stack.Kind = "jvm-service"
	}
	if has("pyproject.toml") || has("requirements.txt") {
		stack.Languages = append(stack.Languages, "python")
		stack.Frameworks = append(stack.Frameworks, "python")
		if stack.Kind == "generic" {
			stack.Kind = "python-service"
		}
	}
	stack.Languages = Unique(stack.Languages)
	stack.Frameworks = Unique(stack.Frameworks)
	return stack
}

func readPackage(path string) packageJSON {
	var pkg packageJSON
	data, err := os.ReadFile(path)
	if err != nil {
		return pkg
	}
	_ = json.Unmarshal(data, &pkg)
	return pkg
}

func packageManager(files []string) string {
	switch {
	case containsExact(files, "pnpm-lock.yaml"):
		return "pnpm"
	case containsExact(files, "yarn.lock"):
		return "yarn"
	case containsExact(files, "bun.lockb") || containsExact(files, "bun.lock"):
		return "bun"
	case containsExact(files, "package-lock.json"):
		return "npm"
	default:
		return "npm"
	}
}
