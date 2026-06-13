package aitools

import (
	"path/filepath"
	"strings"
)

func containsExact(files []string, name string) bool {
	for _, file := range files {
		if file == name {
			return true
		}
	}
	return false
}

func containsPath(files []string, path string) bool {
	path = filepath.ToSlash(path)
	for _, file := range files {
		if file == path || strings.HasPrefix(file, path+"/") {
			return true
		}
	}
	return false
}

func isSourceFile(path string) bool {
	return strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx") || strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".jsx") || strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".kt") || strings.HasSuffix(path, ".java") || strings.HasSuffix(path, ".py")
}

func isTestPath(path string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(lower)
	return strings.Contains(lower, "/test/") ||
		strings.Contains(lower, "/tests/") ||
		strings.Contains(lower, "/__tests__/") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".e2e-spec.ts") ||
		strings.HasSuffix(base, ".spec.tsx") ||
		strings.HasSuffix(base, ".test.tsx") ||
		strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, "test.kt") ||
		strings.HasSuffix(base, "test.java")
}

func isRouteSurfacePath(path string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(lower)
	if strings.Contains(lower, "/dto/") || strings.HasSuffix(base, ".dto.ts") || strings.HasSuffix(base, ".dto.js") {
		return false
	}
	return strings.Contains(lower, "/routes/") ||
		strings.Contains(lower, "/controllers/") ||
		strings.HasSuffix(base, ".controller.ts") ||
		strings.HasSuffix(base, ".controller.js") ||
		strings.HasSuffix(base, "controller.kt") ||
		strings.HasSuffix(base, "controller.java")
}
