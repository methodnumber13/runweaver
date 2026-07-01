package aitools

import (
	"path/filepath"
	"strings"
)

func buildCommands(root string, files []string, stack StackInfo) []string {
	var commands []string
	for _, manifest := range packageJSONManifestPaths(files) {
		pkg := readPackage(filepath.Join(root, manifest))
		dir := filepath.ToSlash(filepath.Dir(manifest))
		packageManager := packageManagerForDir(files, dir)
		if manifest == "package.json" && stack.PackageManager != "" {
			packageManager = stack.PackageManager
		}
		for _, name := range []string{"ci:lint", "lint", "typecheck", "test", "test:e2e", "test:all", "build", "ci:build"} {
			if _, ok := pkg.Scripts[name]; ok {
				commands = append(commands, packageCommand(dir, packageManager, name))
			}
		}
	}
	if Exists(filepath.Join(root, "go.mod")) {
		commands = append(commands, "go test ./...")
	}
	if Exists(filepath.Join(root, "gradlew")) {
		commands = append(commands, "./gradlew test")
	} else if Exists(filepath.Join(root, "build.gradle")) || Exists(filepath.Join(root, "build.gradle.kts")) {
		commands = append(commands, "gradle test")
	}
	if Exists(filepath.Join(root, "pom.xml")) {
		commands = append(commands, "mvn test")
	}
	return Unique(commands)
}

func packageCommand(dir, packageManager, script string) string {
	if packageManager == "" {
		packageManager = "npm"
	}
	command := packageManager + " run " + script
	if dir == "." || dir == "" {
		return command
	}
	return "cd " + filepath.ToSlash(dir) + " && " + command
}

func warnings(index SurfaceIndex) []string {
	var out []string
	if len(index.BuildCommands) == 0 {
		out = append(out, "No build/test command detected; verification guidance must be manual.")
	}
	if len(index.EntryPoints) == 0 {
		out = append(out, "No obvious entrypoint detected.")
	}
	return out
}

func existingDirs(files []string, candidates []string) []string {
	var out []string
	for _, candidate := range candidates {
		if containsPath(files, candidate) {
			out = append(out, candidate)
		}
	}
	return Unique(out)
}

func hasTestBelow(files []string, dir string) bool {
	for _, file := range files {
		if strings.HasPrefix(file, dir+"/") && (strings.Contains(strings.ToLower(file), "test") || strings.Contains(strings.ToLower(file), "spec")) {
			return true
		}
	}
	return false
}
