package aitools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
)

// AdoptionDoctorOptions configures runtime adoption readiness checks.
type AdoptionDoctorOptions struct {
	Runtime string
}

// AdoptionDoctorResult summarizes whether runtimes can natively adopt RunWeaver.
type AdoptionDoctorResult struct {
	Status          string                  `json:"status"`
	Ready           bool                    `json:"ready"`
	RepoRoot        string                  `json:"repoRoot"`
	Runtime         string                  `json:"runtime"`
	Runtimes        []RuntimeAdoptionResult `json:"runtimes"`
	Recommendations []string                `json:"recommendations,omitempty"`
}

// RuntimeAdoptionResult reports one runtime's repo-local startup contract.
type RuntimeAdoptionResult struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Ready           bool            `json:"ready"`
	Status          string          `json:"status"`
	Checks          []AdoptionCheck `json:"checks"`
	Recommendations []string        `json:"recommendations,omitempty"`
}

// AdoptionCheck is one adoption readiness check.
type AdoptionCheck struct {
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Summary     string   `json:"summary"`
	Evidence    []string `json:"evidence,omitempty"`
	NextActions []string `json:"nextActions,omitempty"`
}

// DoctorAdoption checks whether generated runtime metadata points to runweaver start.
func DoctorAdoption(repoPath string, opts AdoptionDoctorOptions) (AdoptionDoctorResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return AdoptionDoctorResult{}, err
	}
	providers, err := ResolveRuntimeSelection(opts.Runtime)
	if err != nil {
		return AdoptionDoctorResult{}, err
	}
	result := AdoptionDoctorResult{
		Status:   "ok",
		Ready:    true,
		RepoRoot: root,
		Runtime:  opts.Runtime,
	}
	for _, provider := range providers {
		runtimeResult := doctorRuntimeAdoption(root, provider)
		result.Runtimes = append(result.Runtimes, runtimeResult)
		if !runtimeResult.Ready {
			result.Ready = false
		}
		result.Recommendations = append(result.Recommendations, runtimeResult.Recommendations...)
	}
	result.Recommendations = Unique(result.Recommendations)
	if !result.Ready {
		result.Status = "warning"
	}
	return result, nil
}

func doctorRuntimeAdoption(root string, provider RuntimeProvider) RuntimeAdoptionResult {
	result := RuntimeAdoptionResult{
		ID:     provider.ID,
		Name:   provider.Name,
		Status: "ok",
		Ready:  true,
	}
	add := func(check AdoptionCheck) {
		result.Checks = append(result.Checks, check)
		if check.Status == "error" {
			result.Ready = false
		}
		if len(check.NextActions) > 0 {
			result.Recommendations = append(result.Recommendations, check.NextActions...)
		}
	}
	add(adoptionRunWeaverPathCheck())
	add(adoptionWorkflowCheck(root))
	add(adoptionProfileCheck(root, provider.ID))
	add(adoptionStartupContractCheck(root, provider.ID))
	add(adoptionCommandShortcutCheck(root, provider.ID))
	add(adoptionMCPToolingCheck(root, provider.ID))
	add(adoptionRuntimePermissionCheck(root, provider.ID))
	if !result.Ready {
		result.Status = "warning"
	}
	result.Recommendations = Unique(result.Recommendations)
	return result
}

func adoptionCommandShortcutCheck(root, runtimeID string) AdoptionCheck {
	paths := commandShortcutPaths(runtimeID)
	for _, relPath := range paths {
		path := filepath.Join(root, relPath)
		data, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(data), "runweaver start") {
			return AdoptionCheck{
				Name:     "command-shortcut",
				Status:   "ok",
				Summary:  "Runtime shortcut points to runweaver start",
				Evidence: []string{relPath},
			}
		}
	}
	return AdoptionCheck{
		Name:     "command-shortcut",
		Status:   "warning",
		Summary:  "Runtime shortcut was not found or does not point to runweaver start",
		Evidence: paths,
		NextActions: []string{
			"Run runweaver init --repo . --runtime " + runtimeID + " --force to refresh generated shortcuts.",
		},
	}
}

func adoptionMCPToolingCheck(root, runtimeID string) AdoptionCheck {
	paths := mcpConfigPaths(runtimeID)
	for _, relPath := range paths {
		path := filepath.Join(root, relPath)
		data, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(data), "runweaver") && strings.Contains(strings.ToLower(string(data)), "mcp") {
			return AdoptionCheck{
				Name:     "mcp-tooling",
				Status:   "ok",
				Summary:  "Runtime config references RunWeaver MCP tooling",
				Evidence: []string{relPath},
			}
		}
	}
	return AdoptionCheck{
		Name:     "mcp-tooling",
		Status:   "warning",
		Summary:  "RunWeaver MCP is not configured for this runtime; CLI fallback remains available",
		Evidence: paths,
		NextActions: []string{
			"Optional: connect runweaver mcp serve --repo . --allow-workflow-writes in the selected runtime config.",
		},
	}
}

func adoptionRuntimePermissionCheck(root, runtimeID string) AdoptionCheck {
	paths := permissionHintPaths(runtimeID)
	for _, relPath := range paths {
		path := filepath.Join(root, relPath)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := strings.ToLower(string(data))
		if runtimePermissionLooksWritable(runtimeID, text) {
			return AdoptionCheck{
				Name:     "runtime-permissions",
				Status:   "ok",
				Summary:  "Runtime metadata includes a write-capable RunWeaver permission hint",
				Evidence: []string{relPath},
			}
		}
	}
	return AdoptionCheck{
		Name:     "runtime-permissions",
		Status:   "warning",
		Summary:  "Runtime write permissions/sandbox hints were not found",
		Evidence: paths,
		NextActions: []string{
			"Ensure the selected runtime can execute runweaver start and update .runweaver/tmp workflow state.",
		},
	}
}

func adoptionRunWeaverPathCheck() AdoptionCheck {
	path, err := exec.LookPath("runweaver")
	if err != nil {
		return AdoptionCheck{
			Name:    "runweaver-path",
			Status:  "warning",
			Summary: "runweaver was not found on PATH for this shell",
			NextActions: []string{
				"Install runweaver on a PATH visible to the selected CLI/Desktop runtime.",
			},
		}
	}
	return AdoptionCheck{
		Name:     "runweaver-path",
		Status:   "ok",
		Summary:  "runweaver is available on PATH",
		Evidence: []string{path},
	}
}

func adoptionWorkflowCheck(root string) AdoptionCheck {
	paths, err := workflowTemplatePaths(root)
	if err != nil {
		return AdoptionCheck{Name: "workflow-templates", Status: "error", Summary: "Workflow templates could not be listed", Evidence: []string{err.Error()}}
	}
	if len(paths) == 0 {
		return AdoptionCheck{
			Name:    "workflow-templates",
			Status:  "error",
			Summary: "No workflow templates found",
			NextActions: []string{
				"Run runweaver init --repo . --force to generate .runweaver/workflows.",
			},
		}
	}
	evidence := make([]string, 0, len(paths))
	for _, path := range paths {
		evidence = append(evidence, rel(root, path))
	}
	return AdoptionCheck{Name: "workflow-templates", Status: "ok", Summary: "Workflow templates are available", Evidence: evidence}
}

func adoptionProfileCheck(root, runtimeID string) AdoptionCheck {
	adapter, ok := RuntimeAdapterByID(runtimeID)
	if !ok {
		return AdoptionCheck{Name: "runtime-profile", Status: "error", Summary: "Unsupported runtime " + runtimeID}
	}
	path := filepath.Join(root, adapter.ProfilePath())
	var profile Profile
	if err := ReadJSON(path, &profile); err != nil {
		return AdoptionCheck{
			Name:     "runtime-profile",
			Status:   "error",
			Summary:  "Runtime profile is missing or invalid",
			Evidence: []string{adapter.ProfilePath(), err.Error()},
			NextActions: []string{
				"Run runweaver init --repo . --runtime " + runtimeID + " --force.",
			},
		}
	}
	if len(profile.Repos) == 0 {
		return AdoptionCheck{
			Name:     "runtime-profile",
			Status:   "warning",
			Summary:  "Runtime profile is parseable but has no repository entries",
			Evidence: []string{adapter.ProfilePath()},
		}
	}
	return AdoptionCheck{Name: "runtime-profile", Status: "ok", Summary: "Runtime profile is readable", Evidence: []string{adapter.ProfilePath()}}
}

func adoptionStartupContractCheck(root, runtimeID string) AdoptionCheck {
	paths := startupContractInstructionPaths(runtimeID)
	var checked []string
	var missing []string
	for _, relPath := range paths {
		path := filepath.Join(root, relPath)
		data, err := os.ReadFile(path)
		if err != nil {
			missing = append(missing, relPath)
			continue
		}
		checked = append(checked, relPath)
		if strings.Contains(string(data), "runweaver start") {
			return AdoptionCheck{
				Name:     "startup-contract",
				Status:   "ok",
				Summary:  "Runtime instructions point agents to runweaver start",
				Evidence: []string{relPath},
			}
		}
	}
	evidence := append([]string{}, checked...)
	evidence = append(evidence, missing...)
	return AdoptionCheck{
		Name:     "startup-contract",
		Status:   "error",
		Summary:  "Runtime instructions do not contain the runweaver start contract",
		Evidence: evidence,
		NextActions: []string{
			"Run runweaver init --repo . --runtime " + runtimeID + " --force to refresh generated instructions.",
		},
	}
}

func startupContractInstructionPaths(runtimeID string) []string {
	switch runtimeID {
	case RuntimeOpenCode:
		return []string{".opencode/agents/swarm.md"}
	case RuntimeCodex:
		return []string{"AGENTS.md", ".codex/agents/swarm.toml"}
	case RuntimeClaude:
		return []string{"CLAUDE.md", ".claude/agents/swarm.md"}
	default:
		return []string{statepath.RootDir}
	}
}

func commandShortcutPaths(runtimeID string) []string {
	switch runtimeID {
	case RuntimeOpenCode:
		return []string{".opencode/commands/runweaver-start.md"}
	case RuntimeCodex:
		return []string{".agents/skills/runweaver-start/SKILL.md"}
	case RuntimeClaude:
		return []string{".claude/skills/runweaver-start/SKILL.md", ".claude/commands/runweaver-start.md"}
	default:
		return nil
	}
}

func mcpConfigPaths(runtimeID string) []string {
	switch runtimeID {
	case RuntimeOpenCode:
		return []string{"opencode.json", "opencode.jsonc"}
	case RuntimeCodex:
		return []string{".codex/config.toml"}
	case RuntimeClaude:
		return []string{".mcp.json", ".claude/settings.json"}
	default:
		return nil
	}
}

func permissionHintPaths(runtimeID string) []string {
	switch runtimeID {
	case RuntimeOpenCode:
		return []string{".opencode/agents/swarm.md", "opencode.json", "opencode.jsonc"}
	case RuntimeCodex:
		return []string{".codex/config.toml", "AGENTS.md"}
	case RuntimeClaude:
		return []string{".claude/settings.json", "CLAUDE.md", ".claude/agents/swarm.md"}
	default:
		return nil
	}
}

func runtimePermissionLooksWritable(runtimeID, text string) bool {
	switch runtimeID {
	case RuntimeOpenCode:
		return strings.Contains(text, "runweaver start") && strings.Contains(text, "allow")
	case RuntimeCodex:
		return strings.Contains(text, "workspace-write") || strings.Contains(text, "danger-full-access")
	case RuntimeClaude:
		return strings.Contains(text, "runweaver start") || strings.Contains(text, "bash(runweaver")
	default:
		return false
	}
}
