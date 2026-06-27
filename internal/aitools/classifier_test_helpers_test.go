package aitools

import (
	"strings"
	"testing"
)

func argValue(args []string, key string) string {
	for i, arg := range args {
		if arg == key && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func testClassifierJSON(domain string) string {
	return `{
  "summary": "AI-classified ` + domain + ` surface.",
  "domains": [{"name":"` + domain + `","description":"Owns ` + domain + ` source and tests","files":["src/checkout/checkout.controller.ts","src/checkout/checkout.service.ts"]}],
  "agents": [{"name":"` + domain + `-contract-agent","description":"Maintains ` + domain + ` contracts","focusFiles":["src/checkout/checkout.controller.ts"],"workflow":["Trace controller"],"verification":["npm test"]}],
  "skills": [{"name":"` + domain + `-contract-surface","description":"` + domain + ` contract workflow","focusFiles":["src/checkout/checkout.controller.ts"],"workflow":["Read DTOs"],"verification":["npm test"]}],
  "verification": ["npm test"]
}`
}

func writeClassifierFixture(t *testing.T, root string) {
	t.Helper()
	writeTestFile(t, root, "opencode.json", `{
  "model": "openai-compatible/coder-model",
  "provider": {
    "openai-compatible": {
      "name": "openai-compatible",
      "npm": "@ai-sdk/openai-compatible",
      "models": {
        "coder-model": {
          "name": "coder-model"
        }
      },
      "options": {
        "baseURL": "https://llm-provider.example.com/v1",
        "apiKey": "{env:RUNWEAVER_MODEL_API_KEY}"
      }
    }
  }
}`)
	writeTestFile(t, root, "package.json", `{
  "scripts": {
    "test": "jest --config ./test/jest-unit.json --runInBand"
  },
  "dependencies": {
    "@nestjs/common": "^10.0.0",
    "@nestjs/core": "^10.0.0",
    "class-validator": "0.14.0"
  },
  "devDependencies": {
    "jest": "29.0.0",
    "typescript": "^5.0.0"
  }
}`)
	writeTestFile(t, root, "src/main.ts", "NestFactory.create(AppModule)\n")
	writeTestFile(t, root, "src/app.module.ts", "@Module({ imports: [] })\nexport class AppModule {}\n")
	writeTestFile(t, root, "src/checkout/checkout.controller.ts", "@Controller('checkout')\nexport class CheckoutController {}\n")
	writeTestFile(t, root, "src/checkout/checkout.service.ts", "export class CheckoutService {}\n")
	writeTestFile(t, root, "test/jest-unit.json", "{}\n")
}

func profileHasAgent(repo RepoProfile, name string) bool {
	for _, agent := range repo.Agents {
		if agent.Name == name {
			return true
		}
	}
	return false
}

func profileHasSkill(repo RepoProfile, name string) bool {
	for _, skill := range repo.CustomSkills {
		if skill.Name == name {
			return true
		}
	}
	return false
}

func containsWarning(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}
