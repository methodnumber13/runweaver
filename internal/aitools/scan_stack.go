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

	manifests := packageJSONManifestPaths(files)
	if len(manifests) > 0 {
		stack.Languages = append(stack.Languages, "typescript/javascript")
		stack.PackageManager = packageManagerForDir(files, ".")
		if stack.PackageManager == "" {
			stack.PackageManager = packageManagerForDir(files, filepath.Dir(manifests[0]))
		}
		hasUISurface := hasAnyUISurface(files)
		hasAPISurface := hasAnyAPISurface(files)
		switch {
		case hasUISurface && !hasAPISurface:
			stack.Kind = "frontend-app"
			stack.Frameworks = append(stack.Frameworks, "frontend-app")
		case hasAPISurface && !hasUISurface:
			stack.Kind = "node-api"
			stack.Frameworks = append(stack.Frameworks, "node-api")
		case hasUISurface && hasAPISurface:
			stack.Kind = "multi-surface"
			stack.Frameworks = append(stack.Frameworks, "frontend-app", "node-api")
		case hasPackageScripts(root, manifests, "build") && stack.Kind == "generic":
			stack.Kind = "frontend-app"
			stack.Frameworks = append(stack.Frameworks, "frontend-app")
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

func hasAnyUISurface(files []string) bool {
	for _, file := range files {
		if categoryFor(file) == "ui" || pathLooksFrontendSurface(file) {
			return true
		}
	}
	return false
}

func hasAnyAPISurface(files []string) bool {
	for _, file := range files {
		if isRouteSurfacePath(file) || categoryFor(file) == "route" {
			return true
		}
		base := filepath.Base(filepath.ToSlash(file))
		if base == "server.ts" || base == "server.js" || base == "main.go" {
			return true
		}
	}
	return false
}

func hasPackageScripts(root string, manifests []string, scripts ...string) bool {
	for _, manifest := range manifests {
		pkg := readPackage(filepath.Join(root, manifest))
		for _, script := range scripts {
			if _, ok := pkg.Scripts[script]; ok {
				return true
			}
		}
	}
	return false
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
	return packageManagerForDir(files, ".")
}

func packageManagerForDir(files []string, dir string) string {
	dir = filepath.ToSlash(filepath.Clean(dir))
	if dir == "." || dir == "" {
		dir = ""
	} else {
		dir += "/"
	}
	switch {
	case containsExact(files, dir+"pnpm-lock.yaml"):
		return "pnpm"
	case containsExact(files, dir+"yarn.lock"):
		return "yarn"
	case containsExact(files, dir+"bun.lockb") || containsExact(files, dir+"bun.lock"):
		return "bun"
	case containsExact(files, dir+"package-lock.json"):
		return "npm"
	case dir != "":
		return packageManagerForDir(files, ".")
	default:
		return "npm"
	}
}
