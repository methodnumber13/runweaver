package aitools

import (
	"context"
	"time"
)

// OpenCodeDoctorOptions configures OpenCode readiness checks.
type OpenCodeDoctorOptions struct {
	OpencodeBin    string
	Agent          string
	ProviderID     string
	SkipModelCheck bool
	Timeout        time.Duration
}

// OpenCodeDoctorResult is the structured readiness report for OpenCode.
type OpenCodeDoctorResult struct {
	Status          string                    `json:"status"`
	Ready           bool                      `json:"ready"`
	RepoRoot        string                    `json:"repoRoot"`
	OpencodeBin     string                    `json:"opencodeBin"`
	Agent           string                    `json:"agent"`
	Checks          []OpenCodeDiagnosticCheck `json:"checks"`
	ModelPreflight  *ModelConfigCheck         `json:"modelPreflight,omitempty"`
	Recommendations []string                  `json:"recommendations,omitempty"`
}

// OpenCodeDiagnosticCheck is one OpenCode readiness check item.
type OpenCodeDiagnosticCheck struct {
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Summary     string   `json:"summary"`
	Evidence    []string `json:"evidence,omitempty"`
	NextActions []string `json:"nextActions,omitempty"`
}

type outputRunner func(ctx context.Context, dir string, name string, args []string, env []string) ([]byte, error)
