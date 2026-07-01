package aitools

import (
	"testing"
)

func TestIndexClassifiesDecoratedAPISurfacesPrecisely(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "package.json", `{
  "scripts": {
    "build": "repo-build",
    "lint": "repo-lint \"{src,test}/**/*.ts\"",
    "test": "repo-test --unit-config ./test/unit.json",
    "test:e2e": "repo-test --e2e-config ./test/e2e.json"
  },
  "dependencies": {
    "route-decorator-lib": "^1.0.0",
    "schema-validator-lib": "1.0.0"
  },
  "devDependencies": {
    "lint-helper": "^1.0.0",
    "test-runner-lib": "1.0.0",
    "format-helper": "^1.0.0",
    "build-compiler-lib": "^1.0.0"
  }
}`)
	writeTestFile(t, root, "src/main.ts", "NestFactory.create(AppModule)\napp.listen(3000)\n")
	writeTestFile(t, root, "src/app.module.ts", "@Module({ imports: [], controllers: [] })\nexport class AppModule {}\n")
	writeTestFile(t, root, "src/auth/auth.controller.ts", "@Controller('auth')\nexport class AuthController { @Get('groups') groups() {} }\n")
	writeTestFile(t, root, "src/auth/dto/identity-introspect-response.dto.ts", "export class IdentityProviderIntrospectResponseDto {}\n")
	writeTestFile(t, root, "src/user/dto/user-routes-response.dto.ts", "export class UserRoutesResponseDto {}\n")
	writeTestFile(t, root, "src/cache/dto/cache-test-response.dto.ts", "export class CacheTestResponseDto {}\n")
	writeTestFile(t, root, "src/analytics/analytics.repository.ts", "export class AnalyticsRepository {}\n")
	writeTestFile(t, root, "src/app.controller.spec.ts", "describe('AppController', () => {})\n")
	writeTestFile(t, root, "test/e2e/auth-module.e2e-spec.ts", "describe('AuthModule', () => {})\n")
	writeTestFile(t, root, "db/migrations/20250101000000_init/migration.sql", "select 1;\n")
	writeTestFile(t, root, "test/unit.json", "{}\n")
	writeTestFile(t, root, "test/e2e.json", "{}\n")

	index, err := Index(root, true)
	if err != nil {
		t.Fatal(err)
	}

	wantCategories := map[string]string{
		"src/main.ts":                 "entrypoint",
		"src/app.module.ts":           "module",
		"src/auth/auth.controller.ts": "route",
		"src/auth/dto/identity-introspect-response.dto.ts":    "contract",
		"src/user/dto/user-routes-response.dto.ts":            "contract",
		"src/cache/dto/cache-test-response.dto.ts":            "contract",
		"src/analytics/analytics.repository.ts":               "persistence",
		"src/app.controller.spec.ts":                          "test",
		"test/e2e/auth-module.e2e-spec.ts":                    "test",
		"db/migrations/20250101000000_init/migration.sql": "persistence",
	}
	for path, want := range wantCategories {
		if got := categoryOf(index, path); got != want {
			t.Fatalf("category(%s) = %q, want %q", path, got, want)
		}
	}
	if containsString(pathsByCategory(index, "route"), "src/user/dto/user-routes-response.dto.ts") {
		t.Fatalf("DTO with routes in its name was classified as route")
	}
	if index.Artifacts.Classification != ".runweaver/tmp/index/repo-classification.json" {
		t.Fatalf("classification artifact path = %q, want repo-classification.json", index.Artifacts.Classification)
	}
	if index.Classification.ValidationStatus != "valid" {
		t.Fatalf("classification = %#v, want valid", index.Classification)
	}
	if !classificationHasDomain(index.Classification, "auth") {
		t.Fatalf("classification domains = %#v, want auth", index.Classification.Domains)
	}
	if !edgeExists(index.Edges, "src/auth/auth.controller.ts", "GET auth/groups", "declares-route") {
		t.Fatalf("route edges = %#v, want GET auth/groups from auth controller", edgesByKind(index.Edges, "declares-route", 20))
	}
}

func categoryOf(index RepoIndex, path string) string {
	for _, file := range index.Files {
		if file.Path == path {
			return file.Category
		}
	}
	return ""
}

func pathsByCategory(index RepoIndex, category string) []string {
	var out []string
	for _, file := range index.Files {
		if file.Category == category {
			out = append(out, file.Path)
		}
	}
	return out
}

func classificationHasDomain(classification RepoClassification, name string) bool {
	for _, domain := range classification.Domains {
		if domain.Name == name {
			return true
		}
	}
	return false
}

func edgeExists(edges []IndexEdge, from, to, kind string) bool {
	for _, edge := range edges {
		if edge.From == from && edge.To == to && edge.Kind == kind {
			return true
		}
	}
	return false
}
