package aitools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	mcpProtocolVersion = "2025-11-25"
	mcpServerName      = "runweaver"
)

type mcpRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpInitializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type mcpTool struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

type mcpCallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type mcpToolResult struct {
	Content           []mcpToolTextContent `json:"content"`
	StructuredContent any                  `json:"structuredContent,omitempty"`
	IsError           bool                 `json:"isError,omitempty"`
}

type mcpToolTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ServeMCPStdio serves a small read-only MCP server over newline-delimited stdio.
func ServeMCPStdio(in io.Reader, out io.Writer, opts MCPServerOptions) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		response, ok := handleMCPMessage([]byte(line), opts)
		if !ok {
			continue
		}
		if err := encoder.Encode(response); err != nil {
			return fmt.Errorf("write MCP response: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read MCP request: %w", err)
	}
	return nil
}

func handleMCPMessage(data []byte, opts MCPServerOptions) (mcpResponse, bool) {
	var request mcpRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return mcpErrorResponse(json.RawMessage("null"), -32700, "parse error: "+err.Error()), true
	}
	if request.ID == nil {
		return mcpResponse{}, false
	}
	id := *request.ID
	if request.JSONRPC != "2.0" || request.Method == "" {
		return mcpErrorResponse(id, -32600, "invalid JSON-RPC request"), true
	}
	result, errCode, errMessage := handleMCPMethod(request.Method, request.Params, opts)
	if errMessage != "" {
		return mcpErrorResponse(id, errCode, errMessage), true
	}
	return mcpResponse{JSONRPC: "2.0", ID: id, Result: result}, true
}

func handleMCPMethod(method string, params json.RawMessage, opts MCPServerOptions) (any, int, string) {
	switch method {
	case "initialize":
		return mcpInitializeResult(params, opts), 0, ""
	case "ping":
		return map[string]any{}, 0, ""
	case "tools/list":
		return map[string]any{"tools": runWeaverMCPTools()}, 0, ""
	case "tools/call":
		result, err := callRunWeaverMCPTool(params, opts)
		if err != nil {
			return nil, -32603, err.Error()
		}
		return result, 0, ""
	default:
		return nil, -32601, "method not found: " + method
	}
}

func mcpInitializeResult(params json.RawMessage, opts MCPServerOptions) map[string]any {
	protocolVersion := mcpProtocolVersion
	var parsed mcpInitializeParams
	if len(params) > 0 && json.Unmarshal(params, &parsed) == nil && strings.TrimSpace(parsed.ProtocolVersion) != "" {
		protocolVersion = parsed.ProtocolVersion
	}
	version := strings.TrimSpace(opts.Version)
	if version == "" {
		version = "dev"
	}
	return map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]any{
			"name":    mcpServerName,
			"version": version,
		},
	}
}

func runWeaverMCPTools() []mcpTool {
	return []mcpTool{
		readOnlyMCPTool(
			"runweaver_status",
			"RunWeaver Status",
			"Return repository initialization, index, and latest workflow state.",
			map[string]any{
				"repo": map[string]any{"type": "string", "description": "Repository path. Defaults to the server --repo value."},
			},
		),
		readOnlyMCPTool(
			"runweaver_get_current",
			"Current RunWeaver Workflow",
			"Return the markdown current workflow mirror used for automatic resume after context loss.",
			map[string]any{
				"repo": map[string]any{"type": "string", "description": "Repository path. Defaults to the server --repo value."},
			},
		),
		readOnlyMCPTool(
			"runweaver_list_workflows",
			"List RunWeaver Workflows",
			"List repo-local workflow templates available to the runtime agent.",
			map[string]any{
				"repo": map[string]any{"type": "string", "description": "Repository path. Defaults to the server --repo value."},
			},
		),
		readOnlyMCPTool(
			"runweaver_verify_workflow",
			"Verify RunWeaver Workflow",
			"Verify latest or explicit workflow run artifacts before finishing a task.",
			map[string]any{
				"repo":   map[string]any{"type": "string", "description": "Repository path. Defaults to the server --repo value."},
				"resume": map[string]any{"type": "string", "description": "Workflow run id, repo-relative run directory, or latest.", "default": "latest"},
			},
		),
	}
}

func readOnlyMCPTool(name, title, description string, properties map[string]any) mcpTool {
	return mcpTool{
		Name:        name,
		Title:       title,
		Description: description,
		InputSchema: map[string]any{
			"type":                 "object",
			"properties":           properties,
			"additionalProperties": false,
		},
		Annotations: map[string]any{
			"readOnlyHint":    true,
			"destructiveHint": false,
			"idempotentHint":  true,
			"openWorldHint":   false,
		},
	}
}

func callRunWeaverMCPTool(params json.RawMessage, opts MCPServerOptions) (mcpToolResult, error) {
	var call mcpCallToolParams
	if len(params) == 0 {
		return mcpToolResult{}, fmt.Errorf("tools/call params are required")
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return mcpToolResult{}, fmt.Errorf("invalid tools/call params: %w", err)
	}
	repo := argumentString(call.Arguments, "repo", opts.RepoPath)
	if repo == "" {
		repo = "."
	}
	switch call.Name {
	case "runweaver_status":
		return mcpStructuredToolResult(RunWeaverStatus(repo))
	case "runweaver_get_current":
		return mcpStructuredToolResult(RunWeaverCurrent(repo))
	case "runweaver_list_workflows":
		return mcpStructuredToolResult(ListRunWeaverWorkflows(repo))
	case "runweaver_verify_workflow":
		resume := argumentString(call.Arguments, "resume", "latest")
		return mcpStructuredToolResult(VerifyWorkflowRun(repo, resume))
	default:
		return mcpToolResult{}, fmt.Errorf("unknown RunWeaver MCP tool %q", call.Name)
	}
}

func mcpStructuredToolResult(value any, err error) (mcpToolResult, error) {
	if err != nil {
		return mcpToolResult{}, err
	}
	text, err := safeJSON(value)
	if err != nil {
		return mcpToolResult{}, err
	}
	return mcpToolResult{
		Content: []mcpToolTextContent{{
			Type: "text",
			Text: text,
		}},
		StructuredContent: value,
	}, nil
}

func argumentString(arguments map[string]any, key, fallback string) string {
	if arguments == nil {
		return fallback
	}
	value, ok := arguments[key]
	if !ok || value == nil {
		return fallback
	}
	text, ok := value.(string)
	if !ok {
		return fallback
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return fallback
	}
	return text
}

func mcpErrorResponse(id json.RawMessage, code int, message string) mcpResponse {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	return mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &mcpError{
			Code:    code,
			Message: message,
		},
	}
}
