package aitools

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

const defaultParticipantCap = 3

// SelectParticipants chooses repo-specific agents and skills for a task/workflow.
func SelectParticipants(repoPath string, opts ParticipantSelectOptions) (ParticipantSelectResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return ParticipantSelectResult{}, err
	}
	task := strings.TrimSpace(opts.Task)
	if task == "" {
		return ParticipantSelectResult{}, fmt.Errorf("participant task is required")
	}
	workflow, workflowPath, err := loadWorkflowForParticipantSelection(root, task, opts.Workflow)
	if err != nil {
		return ParticipantSelectResult{}, err
	}
	runtimeID, _, err := ResolveSingleRuntime(root, opts.Runtime)
	if err != nil {
		return ParticipantSelectResult{}, err
	}
	profile, profilePath, profileLoaded, err := loadParticipantProfile(root, runtimeID, opts.ProfilePath)
	if err != nil {
		return ParticipantSelectResult{}, err
	}
	cap := workflowParticipantCap(workflow)
	if cap <= 0 {
		cap = defaultParticipantCap
	}
	taskTier := opts.TaskTier
	if strings.TrimSpace(taskTier) == "" {
		taskTier = ClassifyTaskTier(task).Tier
	}
	cap = applyTaskTierCap(cap, taskTier)
	candidates := participantCandidates(profile, profileLoaded, workflow)
	scoreParticipantCandidates(candidates, task, workflow)
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			if participantKindRank(candidates[i].Kind) == participantKindRank(candidates[j].Kind) {
				return candidates[i].Name < candidates[j].Name
			}
			return participantKindRank(candidates[i].Kind) < participantKindRank(candidates[j].Kind)
		}
		return candidates[i].Score > candidates[j].Score
	})
	selected := selectParticipantCandidates(candidates, cap)
	names := make([]string, 0, len(selected))
	rationale := make([]string, 0, len(selected))
	selectedSet := map[string]bool{}
	for _, candidate := range selected {
		names = append(names, candidate.Name)
		selectedSet[candidate.Name] = true
		if len(candidate.Rationale) > 0 {
			rationale = append(rationale, candidate.Name+": "+strings.Join(candidate.Rationale, "; "))
		}
	}
	for index := range candidates {
		candidates[index].Selected = selectedSet[candidates[index].Name]
	}
	return ParticipantSelectResult{
		Status:       "success",
		RepoRoot:     root,
		Task:         task,
		Runtime:      runtimeID,
		TaskTier:     taskTier,
		Workflow:     workflow.ID,
		WorkflowPath: rel(root, workflowPath),
		ProfilePath:  relOrEmpty(root, profilePath),
		Cap:          cap,
		Participants: names,
		Rationale:    rationale,
		Candidates:   candidates,
	}, nil
}

func loadWorkflowForParticipantSelection(root, task, workflowPath string) (WorkflowSpec, string, error) {
	if strings.TrimSpace(workflowPath) == "" {
		selected, err := SelectWorkflow(root, WorkflowSelectOptions{Task: task})
		if err != nil {
			return WorkflowSpec{}, "", err
		}
		workflowPath = selected.WorkflowPath
	}
	path := workflowPath
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	var workflow WorkflowSpec
	if err := ReadJSON(path, &workflow); err != nil {
		return WorkflowSpec{}, "", fmt.Errorf("load workflow %s: %w", workflowPath, err)
	}
	if workflow.ID == "" {
		return WorkflowSpec{}, "", fmt.Errorf("workflow id is required in %s", workflowPath)
	}
	if err := validateWorkflowSpec(workflow); err != nil {
		return WorkflowSpec{}, "", fmt.Errorf("validate workflow %s: %w", workflowPath, err)
	}
	return workflow, path, nil
}

func loadParticipantProfile(root, runtimeID, profilePath string) (Profile, string, bool, error) {
	if strings.TrimSpace(profilePath) != "" {
		path := profilePath
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		var profile Profile
		if err := ReadJSON(path, &profile); err != nil {
			return Profile{}, "", false, fmt.Errorf("load profile %s: %w", profilePath, err)
		}
		return profile, path, true, nil
	}
	paths := participantProfilePaths(runtimeID)
	for _, relPath := range paths {
		path := filepath.Join(root, relPath)
		if !Exists(path) {
			continue
		}
		var profile Profile
		if err := ReadJSON(path, &profile); err != nil {
			return Profile{}, "", false, fmt.Errorf("load profile %s: %w", relPath, err)
		}
		return profile, path, true, nil
	}
	return Profile{}, "", false, nil
}

func participantProfilePaths(runtimeID string) []string {
	if runtimeID == "" || runtimeID == RuntimeAll {
		runtimeID = RuntimeOpenCode
	}
	if adapter, ok := RuntimeAdapterByID(runtimeID); ok {
		return []string{adapter.ProfilePath()}
	}
	return runtimeProfilePaths([]string{RuntimeOpenCode, RuntimeCodex, RuntimeClaude})
}

func participantCandidates(profile Profile, profileLoaded bool, workflow WorkflowSpec) []ParticipantSelectionCandidate {
	seen := map[string]bool{}
	var out []ParticipantSelectionCandidate
	add := func(name, kind, source string, score int, rationale ...string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, ParticipantSelectionCandidate{
			Name:      name,
			Kind:      kind,
			Source:    source,
			Score:     score,
			Rationale: Unique(rationale),
		})
	}
	if profileLoaded {
		for _, repo := range profile.Repos {
			for _, agent := range repo.Agents {
				add(agent.Name, "agent", "repo", 0, participantEvidence(agent.Description, agent.FocusFiles, agent.Workflow, agent.Verification)...)
			}
			for _, skill := range repo.CustomSkills {
				add(skill.Name, "skill", "repo", 0, participantEvidence(skill.Description, skill.FocusFiles, skill.Workflow, skill.Verification)...)
			}
		}
		for _, agent := range profile.GlobalAgents {
			add(agent.Name, "agent", "global", 0, participantEvidence(agent.Description, agent.FocusFiles, agent.Workflow, agent.Verification)...)
		}
	}
	for _, phase := range workflow.Phases {
		for _, name := range phase.Agents {
			add(name, "agent", "workflow", 2, "workflow phase fallback: "+phase.ID)
		}
	}
	return out
}

func participantEvidence(description string, focusFiles, workflow, verification []string) []string {
	var evidence []string
	if strings.TrimSpace(description) != "" {
		evidence = append(evidence, "description: "+description)
	}
	for _, file := range focusFiles {
		evidence = append(evidence, "focus file: "+file)
	}
	for _, step := range workflow {
		evidence = append(evidence, "workflow: "+step)
	}
	for _, command := range verification {
		evidence = append(evidence, "verification: "+command)
	}
	return evidence
}

func scoreParticipantCandidates(candidates []ParticipantSelectionCandidate, task string, workflow WorkflowSpec) {
	taskTokens := tokenizeSelectionText(task)
	workflowAgentNames := map[string]bool{}
	for _, phase := range workflow.Phases {
		for _, name := range phase.Agents {
			workflowAgentNames[name] = true
		}
	}
	for index := range candidates {
		candidate := &candidates[index]
		tokens := tokenizeSelectionText(candidate.Name + " " + strings.Join(candidate.Rationale, " "))
		matches := 0
		for token := range taskTokens {
			if tokens[token] {
				matches++
			}
		}
		if matches > 0 {
			candidate.Score += matches * 8
			candidate.Rationale = append(candidate.Rationale, fmt.Sprintf("matched %d task token(s)", matches))
		}
		if workflowAgentNames[candidate.Name] {
			candidate.Score += 5
			candidate.Rationale = append(candidate.Rationale, "listed by workflow phase")
		}
		switch candidate.Kind {
		case "agent":
			candidate.Score += 3
		case "skill":
			candidate.Score += 2
		}
		candidate.Rationale = Unique(candidate.Rationale)
	}
}

func selectParticipantCandidates(candidates []ParticipantSelectionCandidate, cap int) []ParticipantSelectionCandidate {
	if cap <= 0 {
		cap = defaultParticipantCap
	}
	selected := make([]ParticipantSelectionCandidate, 0, cap)
	add := func(candidate ParticipantSelectionCandidate) {
		if len(selected) >= cap {
			return
		}
		for _, existing := range selected {
			if existing.Name == candidate.Name {
				return
			}
		}
		selected = append(selected, candidate)
	}
	for _, candidate := range candidates {
		if candidate.Kind == "agent" && candidate.Source == "repo" && candidate.Score >= 8 {
			add(candidate)
			break
		}
	}
	for _, candidate := range candidates {
		if candidate.Kind == "skill" && candidate.Score >= 8 {
			add(candidate)
		}
	}
	for _, candidate := range candidates {
		if candidate.Score > 0 {
			add(candidate)
		}
	}
	if len(selected) == 0 && len(candidates) > 0 {
		add(candidates[0])
	}
	return selected
}

func participantKindRank(kind string) int {
	switch kind {
	case "agent":
		return 0
	case "skill":
		return 1
	default:
		return 2
	}
}
