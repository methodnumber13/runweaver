package aitools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanDetectsReactRoutesPagesAndPackageManager(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{
  "scripts": {
    "lint": "eslint .",
    "test": "vitest run",
    "build": "vite build"
  },
  "dependencies": {
    "express": "latest",
    "react": "latest"
  },
  "devDependencies": {
    "@reduxjs/toolkit": "latest"
  }
}`)
	writeTestFile(t, root, "pnpm-lock.yaml", "lockfileVersion: 9\n")
	writeTestFile(t, root, "tsconfig.json", "{}\n")
	writeTestFile(t, root, "src/index.tsx", "export {}\n")
	writeTestFile(t, root, "src/routes/orders.ts", "export {}\n")
	writeTestFile(t, root, "src/components/Checkout.tsx", "export function Checkout() { return null }\n")
	writeTestFile(t, root, "src/pages/Home.tsx", "export function Home() { return null }\n")

	index, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}

	if index.Stack.Kind != "frontend-react" {
		t.Fatalf("kind = %q, want frontend-react", index.Stack.Kind)
	}
	if index.Stack.PackageManager != "pnpm" {
		t.Fatalf("package manager = %q, want pnpm", index.Stack.PackageManager)
	}
	for _, framework := range []string{"react", "node-api", "redux"} {
		if !containsString(index.Stack.Frameworks, framework) {
			t.Fatalf("frameworks = %#v, want %q", index.Stack.Frameworks, framework)
		}
	}
	if !containsString(index.Routes, "src/routes/orders.ts") {
		t.Fatalf("routes = %#v, want src/routes/orders.ts", index.Routes)
	}
	if !containsString(index.Pages, "src/components/Checkout.tsx") || !containsString(index.Pages, "src/pages/Home.tsx") {
		t.Fatalf("pages = %#v, want component and page surfaces", index.Pages)
	}
	for _, command := range []string{"pnpm run lint", "pnpm run test", "pnpm run build"} {
		if !containsString(index.BuildCommands, command) {
			t.Fatalf("build commands = %#v, want %q", index.BuildCommands, command)
		}
	}
}

func TestScanSkipsGeneratedRuntimeMetadataDirectories(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{"scripts":{"test":"vitest run"},"dependencies":{"express":"latest"}}`)
	writeTestFile(t, root, "src/routes/orders.ts", "export {}\n")
	writeTestFile(t, root, ".codex/agents/generated.toml", "name = \"generated\"\n")
	writeTestFile(t, root, ".claude/agents/generated.md", "`src/generated-claude.ts`\n")
	writeTestFile(t, root, ".agents/skills/generated/SKILL.md", "`src/generated-skill.ts`\n")
	writeTestFile(t, root, ".runweaver/tmp/go-build/cache.go", "package cache\n")
	writeTestFile(t, root, ".runweaver/tmp/swarm-runs/latest/checkpoint.json", `{"status":"complete"}`)
	writeTestFile(t, root, ".runweaver/workflows/bugfix-swarm.json", `{"id":"bugfix-swarm"}`)

	index, err := Index(root, true)
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range index.Files {
		for _, prefix := range []string{".codex/", ".claude/", ".agents/"} {
			if strings.HasPrefix(file.Path, prefix) {
				t.Fatalf("indexed generated runtime metadata file %q", file.Path)
			}
		}
		if strings.HasPrefix(file.Path, ".runweaver/tmp/") {
			t.Fatalf("indexed runtime tmp artifact %q", file.Path)
		}
	}
	if !indexFileExists(index.Files, ".runweaver/workflows/bugfix-swarm.json") {
		t.Fatalf("workflow metadata was skipped; indexed files = %#v", index.Files)
	}
}

func TestScanDetectsJvmGradleService(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "build.gradle.kts", "plugins { kotlin(\"jvm\") version \"2.0.0\" }\n")
	writeTestFile(t, root, "src/main/kotlin/com/acme/Application.kt", "class Application\n")
	writeTestFile(t, root, "src/main/kotlin/com/acme/OrderController.kt", "class OrderController\n")
	writeTestFile(t, root, "src/test/kotlin/com/acme/OrderControllerTest.kt", "class OrderControllerTest\n")

	index, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}

	if index.Stack.Kind != "jvm-service" {
		t.Fatalf("kind = %q, want jvm-service", index.Stack.Kind)
	}
	for _, framework := range []string{"gradle", "kotlin"} {
		if !containsString(index.Stack.Frameworks, framework) {
			t.Fatalf("frameworks = %#v, want %q", index.Stack.Frameworks, framework)
		}
	}
	if !containsString(index.Routes, "src/main/kotlin/com/acme/OrderController.kt") {
		t.Fatalf("routes = %#v, want controller", index.Routes)
	}
	if !containsString(index.TestDirs, "src/test/kotlin") {
		t.Fatalf("test dirs = %#v, want src/test/kotlin", index.TestDirs)
	}
	if !containsString(index.BuildCommands, "gradle test") {
		t.Fatalf("build commands = %#v, want gradle test", index.BuildCommands)
	}
}

func TestScanDetectsNestJSBFFEntrypointsAndAvoidsDTOFalsePositives(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{
  "scripts": {
    "build": "nest build",
    "lint": "eslint \"{src,test}/**/*.ts\"",
    "test": "jest --config ./test/jest-unit.json --runInBand",
    "test:e2e": "jest --config ./test/jest-e2e.json --runInBand"
  },
  "dependencies": {
    "@nestjs/common": "^10.0.0",
    "@nestjs/core": "^10.0.0",
    "@prisma/client": "6.0.0",
    "class-validator": "0.14.0"
  },
  "devDependencies": {
    "eslint": "^8.0.0",
    "jest": "29.0.0",
    "prettier": "^3.0.0",
    "typescript": "^5.0.0"
  }
}`)
	writeTestFile(t, root, "nest-cli.json", "{}\n")
	writeTestFile(t, root, "tsconfig.build.json", "{}\n")
	writeTestFile(t, root, "test/jest-unit.json", "{}\n")
	writeTestFile(t, root, "test/jest-e2e.json", "{}\n")
	writeTestFile(t, root, "helm/values.yaml", "{}\n")
	writeTestFile(t, root, "docker-compose.yml", "services: {}\n")
	writeTestFile(t, root, "prisma/schema.prisma", "model TemplateTrigger { id Int @id }\n")
	writeTestFile(t, root, "src/main.ts", "async function bootstrap() {}\n")
	writeTestFile(t, root, "src/app.module.ts", "@Module({})\nexport class AppModule {}\n")
	writeTestFile(t, root, "src/auth/auth.controller.ts", "@Controller('auth')\nexport class AuthController {}\n")
	writeTestFile(t, root, "src/user/dto/user-routes-response.dto.ts", "export class UserRoutesResponseDto {}\n")
	writeTestFile(t, root, "src/cache/dto/cache-test-response.dto.ts", "export class CacheTestResponseDto {}\n")
	writeTestFile(t, root, "src/auth/dto/identity-introspect-response.dto.ts", "export class IdentityProviderIntrospectResponseDto {}\n")
	writeTestFile(t, root, "src/app.controller.spec.ts", "describe('AppController', () => {})\n")

	index, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, entrypoint := range []string{"src/main.ts", "src/app.module.ts"} {
		if !containsString(index.EntryPoints, entrypoint) {
			t.Fatalf("entryPoints = %#v, want %s", index.EntryPoints, entrypoint)
		}
	}
	if containsString(index.Routes, "src/user/dto/user-routes-response.dto.ts") || containsString(index.Routes, "src/app.controller.spec.ts") {
		t.Fatalf("routes = %#v, want controllers only, not DTOs/specs", index.Routes)
	}
	if !containsString(index.Routes, "src/auth/auth.controller.ts") {
		t.Fatalf("routes = %#v, want auth controller", index.Routes)
	}
	for _, config := range []string{"nest-cli.json", "tsconfig.build.json", "test/jest-unit.json", "test/jest-e2e.json", "helm/values.yaml", "docker-compose.yml", "prisma/schema.prisma"} {
		if !containsString(index.ConfigFiles, config) {
			t.Fatalf("configFiles = %#v, want %s", index.ConfigFiles, config)
		}
	}
	for _, command := range []string{"npm run lint", "npm run test", "npm run test:e2e", "npm run build"} {
		if !containsString(index.BuildCommands, command) {
			t.Fatalf("buildCommands = %#v, want %s", index.BuildCommands, command)
		}
	}
}

func indexFileExists(files []FileInventoryItem, path string) bool {
	for _, file := range files {
		if file.Path == path {
			return true
		}
	}
	return false
}

func writeTestFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
