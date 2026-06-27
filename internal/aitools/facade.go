package aitools

import (
	"github.com/methodnumber13/runweaver/internal/aitools/foundation"
	"github.com/methodnumber13/runweaver/internal/aitools/modelconfig"
	"github.com/methodnumber13/runweaver/internal/aitools/processdiag"
)

// ModelConfigCheckOptions configures model provider readiness checks.
type ModelConfigCheckOptions = modelconfig.ModelConfigCheckOptions

// ModelConfigCheck reports model provider, model, endpoint, and credential state.
type ModelConfigCheck = modelconfig.ModelConfigCheck

// ConfigFileCheck reports one inspected runtime configuration file.
type ConfigFileCheck = modelconfig.ConfigFileCheck

// AuthFileCheck reports one inspected runtime auth file.
type AuthFileCheck = modelconfig.AuthFileCheck

// DetectedModelConfigState summarizes the effective model configuration source.
type DetectedModelConfigState = modelconfig.DetectedModelConfigState

// ProcessDoctorResult summarizes local AI runtime process diagnostics.
type ProcessDoctorResult = processdiag.ProcessDoctorResult

// ProcessSupervisor groups an AI runtime parent process with its direct children.
type ProcessSupervisor = processdiag.ProcessSupervisor

// ProcessInfo is one process row normalized from the operating system.
type ProcessInfo = processdiag.ProcessInfo

// ProcessDuplicateGroup reports duplicate runtime-like processes.
type ProcessDuplicateGroup = processdiag.ProcessDuplicateGroup

// VSCodeDiagnostics reports VS Code auto-attach and debug-process signals.
type VSCodeDiagnostics = processdiag.VSCodeDiagnostics

// DoctorProcesses scans local processes for common AI runtime process issues.
func DoctorProcesses() (ProcessDoctorResult, error) {
	return processdiag.DoctorProcesses()
}

// DoctorProcessesFromPSOutput analyzes captured ps output for deterministic tests.
func DoctorProcessesFromPSOutput(value string) ProcessDoctorResult {
	return processdiag.DoctorProcessesFromPSOutput(value)
}

// CheckModelConfig inspects model provider configuration for the selected repo.
func CheckModelConfig(repoPath string, opts ModelConfigCheckOptions) (ModelConfigCheck, error) {
	return modelconfig.CheckModelConfig(repoPath, opts)
}

// WriteJSON writes indented JSON and creates the parent directory.
func WriteJSON(path string, value any) error {
	return foundation.WriteJSON(path, value)
}

// ReadJSON reads JSON into value and wraps path-aware parse errors.
func ReadJSON(path string, value any) error {
	return foundation.ReadJSON(path, value)
}

func safeJSON(value any) (string, error) {
	return foundation.SafeJSON(value)
}

// Now returns the current UTC timestamp in RFC3339 format.
func Now() string {
	return foundation.Now()
}

// Unique trims, slash-normalizes, sorts, and de-duplicates strings.
func Unique(items []string) []string {
	return foundation.Unique(items)
}

// Limit returns at most n strings while preserving order.
func Limit(items []string, n int) []string {
	return foundation.Limit(items, n)
}

// RepoRoot resolves and validates a repository directory path.
func RepoRoot(path string) (string, error) {
	return foundation.RepoRoot(path)
}

// Exists reports whether path exists.
func Exists(path string) bool {
	return foundation.Exists(path)
}

func rel(root, path string) string {
	return foundation.Rel(root, path)
}

func shouldSkipDir(name string) bool {
	return foundation.ShouldSkipDir(name)
}
