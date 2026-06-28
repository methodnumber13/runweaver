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
		return map[string]any{"tools": runWeaverMCPTools(opts.AllowWorkflowWrites)}, 0, ""
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

func runWeaverMCPTools(allowWorkflowWrites bool) []mcpTool {
	tools := []mcpTool{
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
	if !allowWorkflowWrites {
		return tools
	}
	tools = append(tools,
		workflowWriteMCPTool(
			"runweaver_start_or_resume",
			"Start Or Resume RunWeaver Workflow",
			"Single task intake entrypoint: refresh context when needed, create or resume workflow state, select participants, and return the execution contract.",
			map[string]any{
				"repo":      map[string]any{"type": "string", "description": "Repository path. Defaults to the server --repo value."},
				"task":      map[string]any{"type": "string", "description": "User task text to route into a RunWeaver workflow."},
				"runtime":   map[string]any{"type": "string", "description": "Runtime profile to inspect: opencode, codex, or claude.", "default": "opencode"},
				"workflow":  map[string]any{"type": "string", "description": "Optional explicit workflow JSON path."},
				"profile":   map[string]any{"type": "string", "description": "Optional explicit RunWeaver profile JSON path."},
				"skipIndex": map[string]any{"type": "boolean", "description": "Skip automatic index refresh."},
				"forceNew":  map[string]any{"type": "boolean", "description": "Create a new workflow even when the latest run matches."},
			},
			[]string{"task"},
		),
		workflowWriteMCPTool(
			"runweaver_plan_workflow",
			"Plan RunWeaver Workflow",
			"Create a durable RunWeaver workflow plan/checkpoint under .runweaver/tmp/swarm-runs.",
			map[string]any{
				"repo":     map[string]any{"type": "string", "description": "Repository path. Defaults to the server --repo value."},
				"workflow": map[string]any{"type": "string", "description": "Workflow JSON path. Defaults to feature-delivery-swarm when omitted."},
				"task":     map[string]any{"type": "string", "description": "Task text to store in plan.json and checkpoint.json."},
			},
			[]string{"task"},
		),
		workflowWriteMCPTool(
			"runweaver_update_workflow",
			"Update RunWeaver Workflow",
			"Update checkpoint/todo/current workflow state under .runweaver/tmp/swarm-runs.",
			map[string]any{
				"repo":                 map[string]any{"type": "string", "description": "Repository path. Defaults to the server --repo value."},
				"resume":               map[string]any{"type": "string", "description": "Workflow run id, repo-relative run directory, or latest.", "default": "latest"},
				"phase":                map[string]any{"type": "string", "description": "Current workflow phase id."},
				"status":               map[string]any{"type": "string", "description": "Checkpoint status, for example in_progress or complete."},
				"participants":         stringArraySchema("Participant names to record."),
				"participantRationale": stringArraySchema("Reasons for participant selection."),
				"findings":             stringArraySchema("Findings to append."),
				"decisions":            stringArraySchema("Decisions to append."),
				"filesRead":            stringArraySchema("Repository files read."),
				"filesChanged":         stringArraySchema("Repository files changed."),
				"artifacts":            stringArraySchema("Workflow artifacts created or updated."),
				"lastResult":           map[string]any{"type": "string", "description": "Last command, agent, or phase result explaining why the workflow is moving or pausing."},
				"rejectedPaths":        stringArraySchema("Paths, commands, or approaches rejected or paused, with reasons."),
				"nextAction":           map[string]any{"type": "string", "description": "Next action to persist."},
				"nextVerification":     map[string]any{"type": "string", "description": "Next verification step before continuing or finishing."},
				"verification":         stringArraySchema("Verification commands or checks."),
				"verificationResults":  stringArraySchema("Verification results."),
				"blockers":             stringArraySchema("Concrete blockers."),
				"completePhase":        map[string]any{"type": "boolean", "description": "Mark current phase complete and advance nextPhase."},
			},
			nil,
		),
	)
	return tools
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

func workflowWriteMCPTool(name, title, description string, properties map[string]any, required []string) mcpTool {
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return mcpTool{
		Name:        name,
		Title:       title,
		Description: description,
		InputSchema: schema,
		Annotations: map[string]any{
			"readOnlyHint":    false,
			"destructiveHint": false,
			"idempotentHint":  false,
			"openWorldHint":   false,
		},
	}
}

func stringArraySchema(description string) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": description,
		"items": map[string]any{
			"type": "string",
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
	case "runweaver_start_or_resume":
		if !opts.AllowWorkflowWrites {
			return mcpToolResult{}, fmt.Errorf("workflow write tools are disabled; restart runweaver mcp serve with --allow-workflow-writes")
		}
		return mcpStructuredToolResult(StartWorkflow(repo, mcpStartOptions(call.Arguments)))
	case "runweaver_plan_workflow":
		if !opts.AllowWorkflowWrites {
			return mcpToolResult{}, fmt.Errorf("workflow write tools are disabled; restart runweaver mcp serve with --allow-workflow-writes")
		}
		workflow := argumentString(call.Arguments, "workflow", "")
		task := argumentString(call.Arguments, "task", "")
		return mcpStructuredToolResult(PlanWorkflow(repo, workflow, task))
	case "runweaver_update_workflow":
		if !opts.AllowWorkflowWrites {
			return mcpToolResult{}, fmt.Errorf("workflow write tools are disabled; restart runweaver mcp serve with --allow-workflow-writes")
		}
		return mcpStructuredToolResult(UpdateWorkflow(repo, mcpWorkflowUpdateOptions(call.Arguments)))
	default:
		return mcpToolResult{}, fmt.Errorf("unknown RunWeaver MCP tool %q", call.Name)
	}
}

func mcpStartOptions(arguments map[string]any) StartOptions {
	return StartOptions{
		Task:        argumentString(arguments, "task", ""),
		Runtime:     argumentString(arguments, "runtime", RuntimeOpenCode),
		Workflow:    argumentString(arguments, "workflow", ""),
		ProfilePath: argumentString(arguments, "profile", ""),
		SkipIndex:   argumentBool(arguments, "skipIndex"),
		ForceNew:    argumentBool(arguments, "forceNew"),
	}
}

func mcpWorkflowUpdateOptions(arguments map[string]any) WorkflowUpdateOptions {
	return WorkflowUpdateOptions{
		Resume:               argumentString(arguments, "resume", "latest"),
		Phase:                argumentString(arguments, "phase", ""),
		Status:               argumentString(arguments, "status", ""),
		Participants:         argumentStringSlice(arguments, "participants"),
		ParticipantRationale: argumentStringSlice(arguments, "participantRationale"),
		Findings:             argumentStringSlice(arguments, "findings"),
		Decisions:            argumentStringSlice(arguments, "decisions"),
		FilesRead:            argumentStringSlice(arguments, "filesRead"),
		FilesChanged:         argumentStringSlice(arguments, "filesChanged"),
		Artifacts:            argumentStringSlice(arguments, "artifacts"),
		LastResult:           argumentString(arguments, "lastResult", ""),
		RejectedPaths:        argumentStringSlice(arguments, "rejectedPaths"),
		NextAction:           argumentString(arguments, "nextAction", ""),
		NextVerification:     argumentString(arguments, "nextVerification", ""),
		Verification:         argumentStringSlice(arguments, "verification"),
		VerificationResults:  argumentStringSlice(arguments, "verificationResults"),
		Blockers:             argumentStringSlice(arguments, "blockers"),
		CompletePhase:        argumentBool(arguments, "completePhase"),
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

func argumentStringSlice(arguments map[string]any, key string) []string {
	if arguments == nil {
		return nil
	}
	value, ok := arguments[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				continue
			}
			text = strings.TrimSpace(text)
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	case []string:
		return compactStrings(typed)
	case string:
		return compactStrings(strings.Split(typed, ","))
	default:
		return nil
	}
}

func argumentBool(arguments map[string]any, key string) bool {
	if arguments == nil {
		return false
	}
	value, ok := arguments[key]
	if !ok || value == nil {
		return false
	}
	typed, ok := value.(bool)
	return ok && typed
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
