package aitools

import (
	"context"
	"time"
)

// WorkflowExecuteOptions configures execution through a selected AI runtime.
type WorkflowExecuteOptions struct {
	WorkflowPath     string
	Task             string
	Resume           string
	Runtime          string
	OpencodeBin      string
	CodexBin         string
	ClaudeBin        string
	Agent            string
	ProviderID       string
	Model            string
	Attach           string
	Format           string
	Sandbox          string
	ApprovalPolicy   string
	PermissionMode   string
	ClaudeTools      string
	SkipGitRepoCheck bool
	SkipRuntimeCheck bool
	DryRun           bool
	SkipModelCheck   bool
	Timeout          time.Duration
}

// WorkflowExecutionResult describes the planned or executed runtime command.
type WorkflowExecutionResult struct {
	Status           string                  `json:"status"`
	Summary          string                  `json:"summary"`
	Runtime          string                  `json:"runtime"`
	DryRun           bool                    `json:"dryRun,omitempty"`
	Executed         bool                    `json:"executed"`
	Plan             WorkflowRunSummary      `json:"plan"`
	PostCheck        *WorkflowPostCheck      `json:"postCheck,omitempty"`
	PromptPath       string                  `json:"promptPath"`
	OutputPath       string                  `json:"outputPath,omitempty"`
	StdoutPath       string                  `json:"stdoutPath,omitempty"`
	StderrPath       string                  `json:"stderrPath,omitempty"`
	Command          []string                `json:"command"`
	ExitCode         int                     `json:"exitCode,omitempty"`
	ModelPreflight   *ModelConfigCheck       `json:"modelPreflight,omitempty"`
	RuntimePreflight *RuntimeDiscoveryResult `json:"runtimePreflight,omitempty"`
}

// WorkflowPostCheck summarizes checkpoint progress after a runtime exits.
type WorkflowPostCheck struct {
	Status          string   `json:"status"`
	Summary         string   `json:"summary"`
	CompletedPhases int      `json:"completedPhases"`
	NextPhase       string   `json:"nextPhase,omitempty"`
	CheckpointState string   `json:"checkpointState,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
}

type commandRunner func(ctx context.Context, dir string, name string, args []string, stdoutPath string, stderrPath string, env []string) (int, error)

type workflowRuntimeExecutionSpec struct {
	Runtime     string
	DisplayName string
	Binary      string
	Args        []string
	Command     []string
	Env         []string
	PromptPath  string
	OutputPath  string
	StdoutPath  string
	StderrPath  string
}
