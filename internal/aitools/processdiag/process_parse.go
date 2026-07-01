package processdiag

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func parseProcessLines(value string) []ProcessInfo {
	var out []ProcessInfo
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}
		pid, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		out = append(out, ProcessInfo{
			PID:     pid,
			PPID:    ppid,
			Elapsed: parts[2],
			Kind:    processKind(strings.Join(parts[3:], " ")),
			Command: strings.Join(parts[3:], " "),
		})
	}
	return out
}

func supervisorKind(command string) string {
	base := filepathBase(firstCommandToken(command))
	switch {
	case base == "codex" || strings.Contains(command, " codex "):
		return "codex"
	case base == "opencode" || strings.Contains(command, " opencode "):
		return "opencode"
	default:
		return ""
	}
}

func processKind(command string) string {
	switch {
	case isVSCodeDebuggerCommand(command):
		return "vscode-debugger"
	case isVSCodeHelperCommand(command):
		return "vscode"
	case strings.Contains(command, "@modelcontextprotocol/") || strings.Contains(command, "mcp-server-"):
		return "mcp"
	case strings.Contains(command, "@upstash/context7-mcp") || strings.Contains(command, "context7-mcp"):
		return "mcp"
	case strings.Contains(command, "@playwright/mcp") || strings.Contains(command, "playwright-mcp"):
		return "mcp"
	case strings.Contains(command, "opencode"):
		return "opencode"
	case strings.Contains(command, "codex"):
		return "codex"
	case strings.Contains(command, "node"):
		return "node"
	case strings.Contains(command, "npm"):
		return "npm"
	default:
		return "other"
	}
}

func directRuntimeChildren(children []ProcessInfo) []ProcessInfo {
	var out []ProcessInfo
	for _, child := range children {
		if child.Kind == "mcp" || child.Kind == "node" || child.Kind == "npm" {
			out = append(out, child)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PID < out[j].PID })
	return out
}

func duplicateRuntimeGroups(processes []ProcessInfo) []ProcessDuplicateGroup {
	groups := map[string][]ProcessInfo{}
	for _, proc := range processes {
		if proc.Kind != "mcp" {
			continue
		}
		key := normalizeRuntimeCommand(proc.Command)
		groups[key] = append(groups[key], proc)
	}
	var out []ProcessDuplicateGroup
	for command, items := range groups {
		if len(items) < 2 {
			continue
		}
		sort.Slice(items, func(i, j int) bool { return items[i].PID < items[j].PID })
		out = append(out, ProcessDuplicateGroup{
			Kind:    "mcp",
			Count:   len(items),
			Command: command,
			Items:   items,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Command < out[j].Command
		}
		return out[i].Count > out[j].Count
	})
	return out
}

func normalizeRuntimeCommand(command string) string {
	for _, marker := range []struct {
		match     string
		canonical string
	}{
		{"@modelcontextprotocol/server-github", "@modelcontextprotocol/server-github"},
		{"mcp-server-github", "@modelcontextprotocol/server-github"},
		{"@modelcontextprotocol/server-memory", "@modelcontextprotocol/server-memory"},
		{"mcp-server-memory", "@modelcontextprotocol/server-memory"},
		{"@modelcontextprotocol/server-sequential-thinking", "@modelcontextprotocol/server-sequential-thinking"},
		{"mcp-server-sequential-thinking", "@modelcontextprotocol/server-sequential-thinking"},
		{"@upstash/context7-mcp", "@upstash/context7-mcp"},
		{"context7-mcp", "@upstash/context7-mcp"},
		{"@playwright/mcp", "@playwright/mcp"},
		{"playwright-mcp", "@playwright/mcp"},
	} {
		if strings.Contains(command, marker.match) {
			return marker.canonical
		}
	}
	return command
}

func longRunningUnattendedSupervisors(supervisors []ProcessSupervisor) []ProcessSupervisor {
	var out []ProcessSupervisor
	for _, supervisor := range supervisors {
		if !isUnattendedRuntimeCommand(supervisor.Command) {
			continue
		}
		if elapsedDuration(supervisor.Elapsed) < 5*time.Minute {
			continue
		}
		if !hasMCPChild(supervisor.Children) {
			continue
		}
		out = append(out, supervisor)
	}
	return out
}

func isUnattendedRuntimeCommand(command string) bool {
	lower := strings.ToLower(command)
	return strings.Contains(lower, "codex exec") ||
		strings.Contains(lower, "opencode run") ||
		strings.Contains(lower, "claude --print")
}

func hasMCPChild(children []ProcessInfo) bool {
	for _, child := range children {
		if child.Kind == "mcp" {
			return true
		}
	}
	return false
}

func elapsedDuration(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	days := 0
	if before, after, ok := strings.Cut(value, "-"); ok {
		parsedDays, err := strconv.Atoi(before)
		if err != nil {
			return 0
		}
		days = parsedDays
		value = after
	}
	parts := strings.Split(value, ":")
	total := time.Duration(days) * 24 * time.Hour
	switch len(parts) {
	case 2:
		minutes, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0
		}
		seconds, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0
		}
		return total + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second
	case 3:
		hours, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0
		}
		minutes, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0
		}
		seconds, err := strconv.Atoi(parts[2])
		if err != nil {
			return 0
		}
		return total + time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second
	default:
		return 0
	}
}

func processSummary(supervisors []ProcessSupervisor, duplicates []ProcessDuplicateGroup, longRunningUnattended []ProcessSupervisor, vscode VSCodeDiagnostics) string {
	childCount := 0
	for _, supervisor := range supervisors {
		childCount += len(supervisor.Children)
	}
	vscodeSuffix := ""
	if vscode.Detected {
		vscodeSuffix = ", VS Code JavaScript debugger/helper process(es) are present"
	}
	longRunningSuffix := ""
	if len(longRunningUnattended) > 0 {
		longRunningSuffix = ", and long-running unattended runtime process(es) with MCP children"
	}
	if len(duplicates) > 0 {
		return "found " + strconv.Itoa(len(supervisors)) + " Codex/OpenCode supervisor process(es), " + strconv.Itoa(childCount) + " direct runtime child process(es), duplicate MCP runtime groups" + longRunningSuffix + vscodeSuffix
	}
	return "found " + strconv.Itoa(len(supervisors)) + " Codex/OpenCode supervisor process(es) and " + strconv.Itoa(childCount) + " direct runtime child process(es)" + longRunningSuffix + vscodeSuffix
}

func firstCommandToken(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func filepathBase(path string) string {
	if path == "" {
		return ""
	}
	parts := strings.Split(path, string(os.PathSeparator))
	return parts[len(parts)-1]
}

func processRecommendations(supervisors []ProcessSupervisor, duplicates []ProcessDuplicateGroup, longRunningUnattended []ProcessSupervisor, vscode VSCodeDiagnostics) []string {
	var out []string
	if len(duplicates) > 0 {
		out = append(out, "Close stale Codex/OpenCode sessions instead of terminating their child MCP Node.js processes one by one.")
	}
	if len(longRunningUnattended) > 0 {
		out = append(out, "Interrupt or close the long-running unattended Codex/OpenCode supervisor process so it can terminate its MCP child processes; avoid killing MCP child processes one by one unless the supervisor is already gone.")
	}
	if len(supervisors) > 1 {
		out = append(out, "Multiple Codex/OpenCode supervisor processes are active; close unused terminals or desktop sessions to release their MCP children.")
	}
	if vscode.AutoAttachRecommendation != "" {
		out = append(out, vscode.AutoAttachRecommendation)
	}
	return out
}
