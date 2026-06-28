package aitools

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestServeMCPStdioInitializesListsToolsAndReturnsStatus(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/tool\n")
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"runweaver_status","arguments":{"repo":"` + root + `"}}}`,
	}, "\n") + "\n"
	var out bytes.Buffer

	if err := ServeMCPStdio(strings.NewReader(input), &out, MCPServerOptions{RepoPath: root, Version: "test"}); err != nil {
		t.Fatal(err)
	}

	responses := decodeMCPResponses(t, out.String())
	if len(responses) != 3 {
		t.Fatalf("responses = %d, want 3; output:\n%s", len(responses), out.String())
	}
	if responses[0]["id"].(float64) != 1 || responses[0]["error"] != nil {
		t.Fatalf("initialize response = %#v, want id 1 without error", responses[0])
	}
	initResult := responses[0]["result"].(map[string]any)
	if initResult["protocolVersion"] != "2025-03-26" {
		t.Fatalf("protocolVersion = %v, want client version", initResult["protocolVersion"])
	}
	serverInfo := initResult["serverInfo"].(map[string]any)
	if serverInfo["version"] != "test" {
		t.Fatalf("serverInfo.version = %v, want test", serverInfo["version"])
	}
	if responses[1]["id"].(float64) != 2 || !strings.Contains(mustJSON(t, responses[1]), "runweaver_get_current") {
		t.Fatalf("tools/list response = %#v, want RunWeaver tools", responses[1])
	}
	if responses[2]["id"].(float64) != 3 || !strings.Contains(mustJSON(t, responses[2]), `"ready":false`) {
		t.Fatalf("status tool response = %#v, want ready false status", responses[2])
	}
}

func TestServeMCPStdioWorkflowToolsExposeCurrentAndVerification(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {"id": "plan", "name": "Plan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "plan"}
  ]
}`)
	if _, err := PlanWorkflow(root, ".runweaver/workflows/test-swarm.json", "ship status tool"); err != nil {
		t.Fatal(err)
	}
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"workflows","method":"tools/call","params":{"name":"runweaver_list_workflows","arguments":{"repo":"` + root + `"}}}`,
		`{"jsonrpc":"2.0","id":"current","method":"tools/call","params":{"name":"runweaver_get_current","arguments":{"repo":"` + root + `"}}}`,
		`{"jsonrpc":"2.0","id":"verify","method":"tools/call","params":{"name":"runweaver_verify_workflow","arguments":{"repo":"` + root + `","resume":"latest"}}}`,
	}, "\n") + "\n"
	var out bytes.Buffer

	if err := ServeMCPStdio(strings.NewReader(input), &out, MCPServerOptions{RepoPath: root}); err != nil {
		t.Fatal(err)
	}

	output := out.String()
	for _, want := range []string{"test-swarm", "Current RunWeaver Workflow", `"run-dir"`, `"id":"workflows"`, `"id":"current"`, `"id":"verify"`} {
		if !strings.Contains(output, want) {
			t.Fatalf("MCP output missing %q:\n%s", want, output)
		}
	}
}

func TestServeMCPStdioHidesWorkflowWriteToolsByDefault(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/tool\n")
	var out bytes.Buffer
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"list","method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":"write","method":"tools/call","params":{"name":"runweaver_plan_workflow","arguments":{"repo":"` + root + `","task":"ship feature"}}}`,
	}, "\n") + "\n"

	if err := ServeMCPStdio(strings.NewReader(input), &out, MCPServerOptions{RepoPath: root}); err != nil {
		t.Fatal(err)
	}

	output := out.String()
	if strings.Contains(output, "Plan RunWeaver Workflow") {
		t.Fatalf("default tools/list exposed workflow write tool:\n%s", output)
	}
	if !strings.Contains(output, "workflow write tools are disabled") {
		t.Fatalf("disabled write call output = %s, want explicit disabled error", output)
	}
}

func TestServeMCPStdioAllowsGatedWorkflowWrites(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".runweaver/workflows/test-swarm.json", `{
  "id": "test-swarm",
  "name": "Test Swarm",
  "phases": [
    {"id": "plan", "name": "Plan", "scope": "repo", "mode": "parallel", "writeMode": "read", "agents": ["repo-surface-indexer"], "prompt": "plan"}
  ]
}`)
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"list","method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":"plan","method":"tools/call","params":{"name":"runweaver_plan_workflow","arguments":{"repo":"` + root + `","workflow":".runweaver/workflows/test-swarm.json","task":"ship feature"}}}`,
		`{"jsonrpc":"2.0","id":"update","method":"tools/call","params":{"name":"runweaver_update_workflow","arguments":{"repo":"` + root + `","resume":"latest","phase":"plan","status":"in_progress","participants":["repo-surface-indexer"],"findings":["mapped repo"],"lastResult":"plan created and index is missing","rejectedPaths":["skip implementation until plan participants are recorded"],"nextAction":"verify","nextVerification":"go test ./...","verification":["go test ./..."]}}}`,
	}, "\n") + "\n"
	var out bytes.Buffer

	if err := ServeMCPStdio(strings.NewReader(input), &out, MCPServerOptions{RepoPath: root, AllowWorkflowWrites: true}); err != nil {
		t.Fatal(err)
	}

	output := out.String()
	for _, want := range []string{"Plan RunWeaver Workflow", "runweaver_update_workflow", `"workflow":"test-swarm"`, `"currentPhase":"plan"`, "repo-surface-indexer", "mapped repo", "lastResult", "rejectedPaths", "nextVerification"} {
		if !strings.Contains(output, want) {
			t.Fatalf("MCP gated write output missing %q:\n%s", want, output)
		}
	}
	status, err := WorkflowStatus(root, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if status["currentPhase"] != "plan" || status["nextAction"] != "verify" || status["lastResult"] != "plan created and index is missing" {
		t.Fatalf("workflow status = %#v, want update persisted through MCP", status)
	}
}

func TestServeMCPStdioReturnsProtocolErrors(t *testing.T) {
	var out bytes.Buffer
	input := strings.Join([]string{
		`{bad-json`,
		`{"jsonrpc":"2.0","id":7,"method":"unknown/method"}`,
		`{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"missing_tool"}}`,
	}, "\n") + "\n"

	if err := ServeMCPStdio(strings.NewReader(input), &out, MCPServerOptions{}); err != nil {
		t.Fatal(err)
	}

	output := out.String()
	for _, want := range []string{`"code":-32700`, `"code":-32601`, `"code":-32603`, "missing_tool"} {
		if !strings.Contains(output, want) {
			t.Fatalf("MCP error output missing %q:\n%s", want, output)
		}
	}
}

func decodeMCPResponses(t *testing.T, output string) []map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	responses := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var response map[string]any
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("decode MCP response %q: %v", line, err)
		}
		responses = append(responses, response)
	}
	return responses
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
