package aitools

import (
	"testing"
)

func TestIndexClassifiesNestJSBFFSurfacesPrecisely(t *testing.T) {
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
	writeTestFile(t, root, "src/main.ts", "NestFactory.create(AppModule)\napp.listen(3000)\n")
	writeTestFile(t, root, "src/app.module.ts", "@Module({ imports: [], controllers: [] })\nexport class AppModule {}\n")
	writeTestFile(t, root, "src/auth/auth.controller.ts", "@Controller('auth')\nexport class AuthController { @Get('groups') groups() {} }\n")
	writeTestFile(t, root, "src/auth/dto/identity-introspect-response.dto.ts", "export class IdentityProviderIntrospectResponseDto {}\n")
	writeTestFile(t, root, "src/user/dto/user-routes-response.dto.ts", "export class UserRoutesResponseDto {}\n")
	writeTestFile(t, root, "src/cache/dto/cache-test-response.dto.ts", "export class CacheTestResponseDto {}\n")
	writeTestFile(t, root, "src/templates/templates.repository.ts", "export class TemplatesRepository {}\n")
	writeTestFile(t, root, "src/app.controller.spec.ts", "describe('AppController', () => {})\n")
	writeTestFile(t, root, "test/e2e/auth-module.e2e-spec.ts", "describe('AuthModule', () => {})\n")
	writeTestFile(t, root, "prisma/schema.prisma", "model TemplateTrigger { id Int @id }\n")
	writeTestFile(t, root, "prisma/migrations/20250101000000_init/migration.sql", "select 1;\n")
	writeTestFile(t, root, "test/jest-unit.json", "{}\n")
	writeTestFile(t, root, "test/jest-e2e.json", "{}\n")

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
		"src/templates/templates.repository.ts":               "persistence",
		"src/app.controller.spec.ts":                          "test",
		"test/e2e/auth-module.e2e-spec.ts":                    "test",
		"prisma/schema.prisma":                                "persistence",
		"prisma/migrations/20250101000000_init/migration.sql": "persistence",
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
