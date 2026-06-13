package aitools

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func checkRunWeaverAvailability(result *OpenCodeDoctorResult) {
	path, err := exec.LookPath("runweaver")
	if err != nil {
		addDoctorCheck(result, "runweaver-path", "warning", "runweaver is not available on PATH", []string{err.Error()}, []string{"Install runweaver into a PATH visible to OpenCode Desktop and CLI, for example with ./scripts/install.sh."})
		return
	}
	addDoctorCheck(result, "runweaver-path", "ok", "runweaver is available on PATH", []string{path}, nil)
}

func checkLocalMetadata(root, agent string, result *OpenCodeDoctorResult) {
	agentPath := filepath.Join(root, ".opencode", "agents", agent+".md")
	if Exists(agentPath) {
		addDoctorCheck(result, "local-agent-file", "ok", "Project swarm agent file exists", []string{rel(root, agentPath)}, nil)
	} else {
		addDoctorCheck(result, "local-agent-file", "error", "Project swarm agent file is missing", []string{rel(root, agentPath)}, []string{"Run runweaver init --repo . --force."})
	}
	skillCount := countSkillFiles(filepath.Join(root, ".opencode", "skills"))
	if skillCount == 0 {
		addDoctorCheck(result, "local-skills", "warning", "No project-local OpenCode skills were found", []string{".opencode/skills"}, []string{"Run runweaver init --repo . --force or runweaver refresh --repo . --apply."})
	} else {
		addDoctorCheck(result, "local-skills", "ok", fmt.Sprintf("Found %d project-local OpenCode skill(s)", skillCount), []string{".opencode/skills"}, nil)
	}
}

func checkResolvedConfig(config map[string]any, agent string, result *OpenCodeDoctorResult) {
	defaultAgent := firstString(config, "default_agent", "defaultAgent")
	if defaultAgent == "" {
		addDoctorCheck(result, "default-agent", "warning", "Resolved OpenCode config has no default_agent", nil, []string{"Set default_agent to " + agent + " in project opencode.json."})
	} else if defaultAgent != agent {
		addDoctorCheck(result, "default-agent", "warning", "Resolved OpenCode default_agent is not the swarm agent", []string{"default_agent=" + defaultAgent}, []string{"Set default_agent to " + agent + " in project opencode.json or active OpenCode config."})
	} else {
		addDoctorCheck(result, "default-agent", "ok", "Resolved OpenCode default_agent points to swarm", []string{"default_agent=" + defaultAgent}, nil)
	}
	if agentExistsInConfig(config, agent) {
		addDoctorCheck(result, "resolved-agent-config", "ok", "Resolved OpenCode config includes the swarm agent", []string{agent}, nil)
	} else {
		addDoctorCheck(result, "resolved-agent-config", "warning", "Resolved OpenCode config does not expose the swarm agent in the agent map", []string{agent}, []string{"This may be acceptable if opencode debug agent " + agent + " succeeds; otherwise run runweaver init --repo . --force."})
	}
	if permissionsAllowTools(config, "task", "todowrite") {
		addDoctorCheck(result, "resolved-permissions", "ok", "Resolved OpenCode permissions allow task and todowrite", []string{"task", "todowrite"}, nil)
	} else {
		addDoctorCheck(result, "resolved-permissions", "info", "Top-level OpenCode permissions do not explicitly allow task and todowrite; agent-level tool permissions will be checked separately", []string{"task", "todowrite"}, nil)
	}
}

func checkResolvedAgent(agentConfig map[string]any, agent string, result *OpenCodeDoctorResult) {
	name := firstString(agentConfig, "name", "id")
	if name != "" && name != agent {
		addDoctorCheck(result, "resolved-agent-name", "warning", "OpenCode resolved a different agent name", []string{name}, []string{"Check .opencode/agents/" + agent + ".md."})
	} else {
		addDoctorCheck(result, "resolved-agent-name", "ok", "OpenCode resolves the swarm agent", []string{agent}, nil)
	}
	mode := firstString(agentConfig, "mode")
	if mode != "" && mode != "primary" {
		addDoctorCheck(result, "resolved-agent-mode", "warning", "Swarm agent is not primary mode", []string{"mode=" + mode}, []string{"Set mode: primary in .opencode/agents/" + agent + ".md."})
	} else {
		addDoctorCheck(result, "resolved-agent-mode", "ok", "Swarm agent is primary", []string{"mode=" + firstNonEmpty(mode, "primary")}, nil)
	}
	if agentToolEnabled(agentConfig, "task") && agentToolEnabled(agentConfig, "todowrite") {
		addDoctorCheck(result, "resolved-agent-tools", "ok", "Swarm agent can use task and todowrite", []string{"task", "todowrite"}, nil)
	} else {
		addDoctorCheck(result, "resolved-agent-tools", "error", "Swarm agent is missing task or todowrite", []string{"task", "todowrite"}, []string{"Regenerate .opencode/agents/swarm.md and opencode.json with runweaver init --repo . --force."})
	}
}

func addDoctorCheck(result *OpenCodeDoctorResult, name, status, summary string, evidence []string, nextActions []string) {
	check := OpenCodeDiagnosticCheck{
		Name:        name,
		Status:      status,
		Summary:     summary,
		Evidence:    compactStrings(evidence),
		NextActions: compactStrings(nextActions),
	}
	result.Checks = append(result.Checks, check)
}

func finalizeDoctorResult(result *OpenCodeDoctorResult) {
	status := "ok"
	ready := true
	var recommendations []string
	for _, check := range result.Checks {
		switch check.Status {
		case "error":
			status = "error"
			ready = false
		case "warning":
			if status != "error" {
				status = "warning"
			}
			ready = false
		}
		recommendations = append(recommendations, check.NextActions...)
	}
	result.Status = status
	result.Ready = ready
	result.Recommendations = Unique(recommendations)
}
