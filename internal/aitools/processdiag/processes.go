package processdiag

import (
	"os/exec"
	"sort"
)

// ProcessDoctorResult summarizes local AI runtime process diagnostics.
type ProcessDoctorResult struct {
	Status          string                  `json:"status"`
	Summary         string                  `json:"summary"`
	Supervisors     []ProcessSupervisor     `json:"supervisors"`
	Duplicates      []ProcessDuplicateGroup `json:"duplicates,omitempty"`
	VSCode          VSCodeDiagnostics       `json:"vscode"`
	Recommendations []string                `json:"recommendations,omitempty"`
}

// ProcessSupervisor groups an AI runtime parent process with direct children.
type ProcessSupervisor struct {
	PID      int           `json:"pid"`
	PPID     int           `json:"ppid"`
	Elapsed  string        `json:"elapsed"`
	Kind     string        `json:"kind"`
	Command  string        `json:"command"`
	Children []ProcessInfo `json:"children"`
}

// ProcessInfo is one normalized operating-system process row.
type ProcessInfo struct {
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
	Elapsed string `json:"elapsed"`
	Kind    string `json:"kind"`
	Command string `json:"command"`
}

// ProcessDuplicateGroup reports multiple matching runtime-like processes.
type ProcessDuplicateGroup struct {
	Kind    string        `json:"kind"`
	Count   int           `json:"count"`
	Command string        `json:"command"`
	Items   []ProcessInfo `json:"items"`
}

// VSCodeDiagnostics reports VS Code auto-attach and debug-process signals.
type VSCodeDiagnostics struct {
	Detected                 bool     `json:"detected"`
	HelperProcessCount       int      `json:"helperProcessCount"`
	DebuggerProcessCount     int      `json:"debuggerProcessCount"`
	NodeLikeProcessCount     int      `json:"nodeLikeProcessCount"`
	SettingsPath             string   `json:"settingsPath,omitempty"`
	HasAutoAttachSetting     bool     `json:"hasAutoAttachSetting"`
	AutoAttachFilter         string   `json:"autoAttachFilter,omitempty"`
	LegacyNodeAutoAttach     string   `json:"legacyNodeAutoAttach,omitempty"`
	SmartPatternCount        int      `json:"smartPatternCount,omitempty"`
	AutoAttachRecommendation string   `json:"autoAttachRecommendation,omitempty"`
	Notes                    []string `json:"notes,omitempty"`
}

// DoctorProcesses scans the local process table for common runtime issues.
func DoctorProcesses() (ProcessDoctorResult, error) {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,etime=,command=").Output()
	if err != nil {
		return ProcessDoctorResult{}, err
	}
	return DoctorProcessesFromPSOutput(string(out)), nil
}

// DoctorProcessesFromPSOutput analyzes captured ps output for deterministic tests.
func DoctorProcessesFromPSOutput(value string) ProcessDoctorResult {
	processes := parseProcessLines(value)
	byParent := map[int][]ProcessInfo{}
	for _, proc := range processes {
		byParent[proc.PPID] = append(byParent[proc.PPID], proc)
	}
	var supervisors []ProcessSupervisor
	for _, proc := range processes {
		kind := supervisorKind(proc.Command)
		if kind == "" {
			continue
		}
		children := directRuntimeChildren(byParent[proc.PID])
		supervisors = append(supervisors, ProcessSupervisor{
			PID:      proc.PID,
			PPID:     proc.PPID,
			Elapsed:  proc.Elapsed,
			Kind:     kind,
			Command:  proc.Command,
			Children: children,
		})
	}
	sort.Slice(supervisors, func(i, j int) bool {
		if supervisors[i].Kind == supervisors[j].Kind {
			return supervisors[i].PID < supervisors[j].PID
		}
		return supervisors[i].Kind < supervisors[j].Kind
	})
	duplicates := duplicateRuntimeGroups(processes)
	vscode := detectVSCodeDiagnostics(processes, readVSCodeDebugSettings())
	recommendations := processRecommendations(supervisors, duplicates, vscode)
	status := "ok"
	if len(duplicates) > 0 || vscode.AutoAttachRecommendation != "" {
		status = "warning"
	}
	return ProcessDoctorResult{
		Status:          status,
		Summary:         processSummary(supervisors, duplicates, vscode),
		Supervisors:     supervisors,
		Duplicates:      duplicates,
		VSCode:          vscode,
		Recommendations: recommendations,
	}
}
