package aitools

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestPackageAndLanguageHelpersCoverSupportedStacks(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", `module example.com/service

go 1.23

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/stretchr/testify v1.9.0
)
`)
	writeTestFile(t, root, "requirements.txt", "pytest==8.0.0\nruff>=0.5.0\n")

	packages := DetectPackages(root, []string{"go.mod", "requirements.txt"})
	if !packageNamed(packages, "github.com/gin-gonic/gin") || !packageNamed(packages, "pytest") || !packageNamed(packages, "ruff") {
		t.Fatalf("packages = %#v, want Go and Python package detection", packages)
	}
	for _, path := range []string{"main.go", "app.kt", "App.java", "tool.py", "config.yaml", "README.md"} {
		if languageFor(path) == "" {
			t.Fatalf("languageFor(%q) returned empty", path)
		}
	}
}

func TestModelConfigFileCredentialAndAuthFileDetection(t *testing.T) {
	root := t.TempDir()
	isolateModelEnv(t)
	writeTestFile(t, root, "secrets/openai-compatible.key", "test-key\n")
	writeTestFile(t, root, "opencode.json", `{
  "model": "openai-compatible/coder-model",
  "provider": {
    "openai-compatible": {
      "options": {
        "baseURL": "https://llm-provider.example.com/v1",
        "apiKey": "{file:secrets/openai-compatible.key}"
      }
    }
  }
}`)

	result, err := CheckModelConfig(root, ModelConfigCheckOptions{ProviderID: "openai-compatible"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || !result.ProjectConfig.HasAPIKey || result.ProjectConfig.APIKeySource != "file" {
		t.Fatalf("result = %#v, want file-backed ready credential", result)
	}

	authRoot := t.TempDir()
	isolateModelEnv(t)
	t.Setenv("XDG_DATA_HOME", authRoot)
	writeTestFile(t, root, "opencode.json", `{
  "model": "openai-compatible/coder-model",
  "provider": {
    "openai-compatible": {
      "options": {
        "baseURL": "https://llm-provider.example.com/v1"
      }
    }
  }
}`)
	writeTestFile(t, authRoot, "opencode/auth.json", `{"openai-compatible":{"key":"stored-key"}}`)
	result, err = CheckModelConfig(root, ModelConfigCheckOptions{ProviderID: "openai-compatible"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || len(result.AuthFiles) == 0 || !result.AuthFiles[0].HasProvider || !result.AuthFiles[0].HasKey || !result.AuthFiles[0].Parseable {
		t.Fatalf("auth files = %#v, want provider key", result.AuthFiles)
	}
}

func TestRunCommandOutputCapturesLargeStdoutWithoutPipeTruncation(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "large-output.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\npython3 - <<'PY'\nprint('x' * 90000)\nPY\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	output, err := runCommandOutput(context.Background(), root, script, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(output) < 90000 || !strings.Contains(string(output[:10]), "x") {
		t.Fatalf("output length = %d, want large stdout captured", len(output))
	}
}

func TestRunCommandToFilesWritesLogsAndReportsExitCode(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "fail.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho out\necho err >&2\nexit 7\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	stdoutPath := filepath.Join(root, "stdout.log")
	stderrPath := filepath.Join(root, "stderr.log")

	exitCode, err := runCommandToFiles(context.Background(), root, script, nil, stdoutPath, stderrPath, nil)

	if err == nil || exitCode != 7 {
		t.Fatalf("exitCode=%d err=%v, want exit 7 error", exitCode, err)
	}
	stdout, _ := os.ReadFile(stdoutPath)
	stderr, _ := os.ReadFile(stderrPath)
	if strings.TrimSpace(string(stdout)) != "out" || strings.TrimSpace(string(stderr)) != "err" {
		t.Fatalf("stdout=%q stderr=%q, want captured logs", stdout, stderr)
	}
}

func TestAppendWorkflowEventCreatesDirectoryAndAppendsJSONLines(t *testing.T) {
	root := t.TempDir()
	runDir := ".runweaver/tmp/swarm-runs/manual"

	if err := appendWorkflowEvent(root, runDir, WorkflowEvent{Type: "one", At: time.Now().UTC().Format(time.RFC3339)}); err != nil {
		t.Fatal(err)
	}
	if err := appendWorkflowEvent(root, runDir, WorkflowEvent{Type: "two", At: time.Now().UTC().Format(time.RFC3339)}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, runDir, "events.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "\n") != 2 || !strings.Contains(string(data), `"type":"one"`) || !strings.Contains(string(data), `"type":"two"`) {
		t.Fatalf("events log = %q, want two JSON lines", string(data))
	}
}

func TestGoDocCommentsUseIdiomaticPrefixes(t *testing.T) {
	root := repoRootForQualityTest(t)
	fset := token.NewFileSet()
	packages := map[string]struct {
		name       string
		hasDoc     bool
		docExample string
	}{}
	var failures []string

	for _, scanRoot := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, scanRoot), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if strings.HasPrefix(entry.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
			relPath := relForQualityTest(root, path)
			dir := filepath.Dir(path)
			pkg := packages[dir]
			pkg.name = file.Name.Name
			if file.Doc != nil {
				text := strings.TrimSpace(file.Doc.Text())
				pkg.docExample = text
				if packageDocHasExpectedPrefix(file.Name.Name, text) {
					pkg.hasDoc = true
				}
			}
			packages[dir] = pkg

			if file.Name.Name == "main" {
				return nil
			}
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					if d.Name.IsExported() && receiverIsExportedForQualityTest(d.Recv) {
						if !docStartsWithName(d.Doc, d.Name.Name) {
							failures = append(failures, relPath+": func "+d.Name.Name+" must have GoDoc starting with "+d.Name.Name)
						}
					}
				case *ast.GenDecl:
					for _, spec := range d.Specs {
						switch s := spec.(type) {
						case *ast.TypeSpec:
							if s.Name.IsExported() && !docStartsWithName(firstDocForQualityTest(s.Doc, d.Doc), s.Name.Name) {
								failures = append(failures, relPath+": type "+s.Name.Name+" must have GoDoc starting with "+s.Name.Name)
							}
						case *ast.ValueSpec:
							for _, name := range s.Names {
								if name.IsExported() && !docStartsWithName(firstDocForQualityTest(s.Doc, d.Doc), name.Name) {
									failures = append(failures, relPath+": value "+name.Name+" must have GoDoc starting with "+name.Name)
								}
							}
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for dir, pkg := range packages {
		if !pkg.hasDoc {
			failures = append(failures, relForQualityTest(root, dir)+": package "+pkg.name+" must have an idiomatic package GoDoc comment")
		}
	}
	if len(failures) > 0 {
		t.Fatalf("GoDoc style failures:\n%s", strings.Join(failures, "\n"))
	}
}

func packageNamed(packages []PackageInsight, name string) bool {
	for _, pkg := range packages {
		if pkg.Name == name {
			return true
		}
	}
	return false
}

func isolateModelEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("OPENCODE_CONFIG_CONTENT", "")
	t.Setenv("RUNWEAVER_MODEL_API_KEY", "")
}

func repoRootForQualityTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func relForQualityTest(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func receiverIsExportedForQualityTest(recv *ast.FieldList) bool {
	if recv == nil || len(recv.List) == 0 {
		return true
	}
	return exprNameIsExportedForQualityTest(recv.List[0].Type)
}

func exprNameIsExportedForQualityTest(expr ast.Expr) bool {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.IsExported()
	case *ast.StarExpr:
		return exprNameIsExportedForQualityTest(value.X)
	default:
		return false
	}
}

func firstDocForQualityTest(primary, fallback *ast.CommentGroup) *ast.CommentGroup {
	if primary != nil {
		return primary
	}
	return fallback
}

func docStartsWithName(doc *ast.CommentGroup, name string) bool {
	if doc == nil {
		return false
	}
	text := strings.TrimSpace(doc.Text())
	return text == name || strings.HasPrefix(text, name+" ") || strings.HasPrefix(text, name+"\n")
}

func packageDocHasExpectedPrefix(packageName, text string) bool {
	if strings.HasPrefix(text, "Package "+packageName+" ") || strings.HasPrefix(text, "Package "+packageName+"\n") {
		return true
	}
	if packageName == "main" {
		return strings.HasPrefix(text, "Command runweaver ") || strings.HasPrefix(text, "Command runweaver\n")
	}
	return false
}
