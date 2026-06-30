package aitools

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
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
		addDoctorCheck(result, "local-agent-file", "ok", "Project RunWeaver OpenCode agent file exists", []string{rel(root, agentPath)}, nil)
	} else {
		addDoctorCheck(result, "local-agent-file", "error", "Project RunWeaver OpenCode agent file is missing", []string{rel(root, agentPath)}, []string{"Run runweaver init --repo . --force."})
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
		addDoctorCheck(result, "default-agent", "warning", "Resolved OpenCode default_agent is not the RunWeaver OpenCode agent", []string{"default_agent=" + defaultAgent}, []string{"Set default_agent to " + agent + " in project opencode.json or use the generated OpenCode runweaver-start command."})
	} else {
		addDoctorCheck(result, "default-agent", "ok", "Resolved OpenCode default_agent points to the RunWeaver OpenCode agent", []string{"default_agent=" + defaultAgent}, nil)
	}
	if agentExistsInConfig(config, agent) {
		addDoctorCheck(result, "resolved-agent-config", "ok", "Resolved OpenCode config includes the RunWeaver OpenCode agent", []string{agent}, nil)
	} else {
		addDoctorCheck(result, "resolved-agent-config", "warning", "Resolved OpenCode config does not expose the RunWeaver OpenCode agent in the agent map", []string{agent}, []string{"This may be acceptable if opencode debug agent " + agent + " succeeds; otherwise run runweaver init --repo . --force."})
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
		addDoctorCheck(result, "resolved-agent-name", "error", "OpenCode resolved a different agent name", []string{name}, []string{"agent name collision, plugin/config shadows " + agent + "; check .opencode/agents/" + agent + ".md or use reserved fallback name " + openCodeFallbackPrimaryAgentName + "."})
	} else {
		addDoctorCheck(result, "resolved-agent-name", "ok", "OpenCode resolves the RunWeaver OpenCode agent", []string{agent}, nil)
	}
	checkResolvedAgentMarker(agentConfig, agent, result)
	mode := firstString(agentConfig, "mode")
	if mode != "" && mode != "primary" {
		addDoctorCheck(result, "resolved-agent-mode", "warning", "RunWeaver OpenCode agent is not primary mode", []string{"mode=" + mode}, []string{"Set mode: primary in .opencode/agents/" + agent + ".md."})
	} else {
		addDoctorCheck(result, "resolved-agent-mode", "ok", "RunWeaver OpenCode agent is primary", []string{"mode=" + firstNonEmpty(mode, "primary")}, nil)
	}
	if agentToolEnabled(agentConfig, "task") && agentToolEnabled(agentConfig, "todowrite") {
		addDoctorCheck(result, "resolved-agent-tools", "ok", "RunWeaver OpenCode agent can use task and todowrite", []string{"task", "todowrite"}, nil)
	} else {
		addDoctorCheck(result, "resolved-agent-tools", "error", "RunWeaver OpenCode agent is missing task or todowrite", []string{"task", "todowrite"}, []string{"Regenerate .opencode/agents/" + agent + ".md and opencode.json with runweaver init --repo . --force."})
	}
}

func checkResolvedAgentMarker(agentConfig map[string]any, agent string, result *OpenCodeDoctorResult) {
	prompt := resolvedAgentPrompt(agentConfig)
	if hasRunWeaverAgentMarker(prompt) {
		addDoctorCheck(result, "resolved-agent-marker", "ok", "Resolved OpenCode agent prompt contains the RunWeaver startup marker", []string{agent}, nil)
		return
	}
	addDoctorCheck(result, "resolved-agent-marker", "error", "agent name collision, plugin/config shadows "+agent, []string{agent, "missing RunWeaver Startup Protocol marker"}, []string{
		"agent name collision, plugin/config shadows " + agent + "; run opencode debug agent " + agent + " and inspect the prompt.",
		"Regenerate project metadata with runweaver init --repo . --force.",
		"If another plugin owns " + agent + ", switch RunWeaver to reserved fallback name " + openCodeFallbackPrimaryAgentName + " in a follow-up migration.",
	})
}

func resolvedAgentPrompt(agentConfig map[string]any) string {
	return strings.Join(compactStrings([]string{
		firstString(agentConfig, "prompt"),
		firstString(agentConfig, "instructions"),
		firstString(agentConfig, "system"),
		firstString(agentConfig, "description"),
	}), "\n")
}

func hasRunWeaverAgentMarker(prompt string) bool {
	prompt = strings.ToLower(prompt)
	return strings.Contains(prompt, "runweaver startup protocol") &&
		strings.Contains(prompt, "runweaver start") &&
		strings.Contains(prompt, "workflow-aware primary runweaver")
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
