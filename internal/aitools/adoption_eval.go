package aitools

import "strings"

// AdoptionEvalOptions configures local adoption smoke checks.
type AdoptionEvalOptions struct {
	Runtime   string
	Task      string
	SkipIndex bool
}

// AdoptionEvalResult combines adoption doctor output with a start smoke run.
type AdoptionEvalResult struct {
	Status string               `json:"status"`
	Ready  bool                 `json:"ready"`
	Doctor AdoptionDoctorResult `json:"doctor"`
	Start  StartResult          `json:"start,omitempty"`
}

// EvaluateAdoption verifies that generated metadata can route a task through start.
func EvaluateAdoption(repoPath string, opts AdoptionEvalOptions) (AdoptionEvalResult, error) {
	runtimeID := normalizeRuntimeID(opts.Runtime)
	if runtimeID == "" {
		runtimeID = RuntimeOpenCode
	}
	doctor, err := DoctorAdoption(repoPath, AdoptionDoctorOptions{Runtime: runtimeID})
	if err != nil {
		return AdoptionEvalResult{}, err
	}
	result := AdoptionEvalResult{
		Status: "ok",
		Ready:  doctor.Ready,
		Doctor: doctor,
	}
	if !doctor.Ready {
		result.Status = "warning"
		return result, nil
	}
	startRuntime := runtimeID
	if startRuntime == RuntimeAll {
		startRuntime = RuntimeOpenCode
	}
	task := strings.TrimSpace(opts.Task)
	if task == "" {
		task = "RunWeaver adoption smoke test"
	}
	start, err := StartWorkflow(repoPath, StartOptions{
		Task:      task,
		Runtime:   startRuntime,
		SkipIndex: opts.SkipIndex,
		ForceNew:  true,
	})
	if err != nil {
		return AdoptionEvalResult{}, err
	}
	result.Start = start
	result.Ready = doctor.Ready && start.Ready
	if !result.Ready {
		result.Status = "warning"
	}
	return result, nil
}
