package aitools

import (
	"fmt"
	pathpkg "path"
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
	candidates := participantCandidates(root, profile, profileLoaded, workflow)
	contextEvidence, warnings := loadParticipantContextEvidence(root, task)
	scoreParticipantCandidates(candidates, task, workflow, contextEvidence)
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
	assignments := participantAssignments(selected)
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
		Assignments:  assignments,
		Rationale:    rationale,
		Candidates:   candidates,
		Warnings:     warnings,
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

func participantCandidates(root string, profile Profile, profileLoaded bool, workflow WorkflowSpec) []ParticipantSelectionCandidate {
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
		for _, repo := range participantReposForRoot(root, profile.Repos) {
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

func participantReposForRoot(root string, repos []RepoProfile) []RepoProfile {
	if len(repos) <= 1 {
		return repos
	}
	out := make([]RepoProfile, 0, len(repos))
	for _, repo := range repos {
		if repoProfileDirMatchesRoot(root, repo.Dir) {
			out = append(out, repo)
		}
	}
	return out
}

func repoProfileDirMatchesRoot(root, dir string) bool {
	dir = strings.TrimSpace(dir)
	if dir == "" || dir == "." || dir == "./" {
		return true
	}
	cleanDir := filepath.Clean(dir)
	if filepath.IsAbs(cleanDir) {
		return filepath.Clean(root) == cleanDir
	}
	normalizedDir := strings.TrimPrefix(filepath.ToSlash(cleanDir), "./")
	rootBase := filepath.Base(filepath.Clean(root))
	if normalizedDir == rootBase {
		return true
	}
	return pathpkg.Base(normalizedDir) == rootBase
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

func participantAssignments(selected []ParticipantSelectionCandidate) []ParticipantAssignment {
	assignments := make([]ParticipantAssignment, 0, len(selected))
	ownerAssigned := false
	for _, candidate := range selected {
		role := participantRole(candidate, ownerAssigned)
		if role == "owner" {
			ownerAssigned = true
		}
		assignments = append(assignments, ParticipantAssignment{
			Name:      candidate.Name,
			Kind:      candidate.Kind,
			Role:      role,
			Source:    candidate.Source,
			Score:     candidate.Score,
			Rationale: candidate.Rationale,
		})
	}
	return assignments
}

func participantRole(candidate ParticipantSelectionCandidate, ownerAssigned bool) string {
	text := strings.ToLower(candidate.Name + " " + strings.Join(candidate.Rationale, " "))
	if candidate.Kind == "skill" {
		name := strings.ToLower(candidate.Name)
		if strings.Contains(name, "test") || strings.Contains(name, "verify") || strings.Contains(name, "quality") {
			return "verifier"
		}
		return "specialist"
	}
	if !ownerAssigned && candidate.Source == "repo" {
		return "owner"
	}
	if strings.Contains(text, "test") || strings.Contains(text, "verify") || strings.Contains(text, "quality") {
		return "verifier"
	}
	if strings.Contains(text, "review") || strings.Contains(text, "security") {
		return "reviewer"
	}
	if !ownerAssigned {
		return "owner"
	}
	return "reviewer"
}

func scoreParticipantCandidates(candidates []ParticipantSelectionCandidate, task string, workflow WorkflowSpec, context participantContextEvidence) {
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
		scoreParticipantContext(candidate, context)
		switch candidate.Kind {
		case "agent":
			candidate.Score += 3
		case "skill":
			candidate.Score += 2
		}
		candidate.Rationale = Unique(candidate.Rationale)
	}
}

type participantContextEvidence struct {
	strongPaths map[string]string
	weakPaths   map[string]string
}

func loadParticipantContextEvidence(root, task string) (participantContextEvidence, []string) {
	context, err := QueryContext(root, ContextQueryOptions{
		Task:  task,
		Limit: 12,
	})
	if err != nil {
		return participantContextEvidence{}, []string{"task context unavailable: " + err.Error()}
	}
	if context.Status != "success" {
		warnings := context.Warnings
		if len(warnings) == 0 {
			warnings = []string{"task context unavailable: context query did not succeed"}
		}
		return participantContextEvidence{}, warnings
	}
	return newParticipantContextEvidence(context), nil
}

func newParticipantContextEvidence(context ContextQueryResult) participantContextEvidence {
	evidence := participantContextEvidence{
		strongPaths: map[string]string{},
		weakPaths:   map[string]string{},
	}
	for _, file := range context.Files {
		if len(file.Rationale) > 0 {
			evidence.addWeak(file.Path, "file: "+file.Path)
		}
	}
	for _, symbol := range context.Symbols {
		evidence.addStrong(symbol.Path, "symbol: "+symbol.Name)
	}
	for _, edge := range context.Routes {
		evidence.addStrong(edge.From, "route: "+edge.To)
	}
	for _, edge := range context.Tests {
		evidence.addStrong(edge.From, "test: "+edge.From)
		evidence.addStrong(edge.To, "tested source: "+edge.To)
	}
	return evidence
}

func (e participantContextEvidence) addStrong(value, reason string) {
	path := normalizeParticipantContextPath(value)
	if !looksLikeRepoPath(path) {
		return
	}
	e.strongPaths[path] = reason
}

func (e participantContextEvidence) addWeak(value, reason string) {
	path := normalizeParticipantContextPath(value)
	if !looksLikeRepoPath(path) {
		return
	}
	e.weakPaths[path] = reason
}

func scoreParticipantContext(candidate *ParticipantSelectionCandidate, context participantContextEvidence) {
	if len(context.strongPaths) == 0 && len(context.weakPaths) == 0 {
		return
	}
	focusFiles := participantFocusFiles(*candidate)
	for _, focusFile := range focusFiles {
		focusPath := normalizeParticipantContextPath(focusFile)
		if _, ok := context.strongPaths[focusPath]; ok {
			candidate.Score += 14
			candidate.Rationale = append(candidate.Rationale, "task context matched focus file: "+focusFile)
			return
		}
	}
	for _, focusFile := range focusFiles {
		focusPath := normalizeParticipantContextPath(focusFile)
		if _, ok := context.weakPaths[focusPath]; ok {
			candidate.Score += 6
			candidate.Rationale = append(candidate.Rationale, "task context matched focus file: "+focusFile)
			return
		}
	}
	for _, focusFile := range focusFiles {
		focusPath := normalizeParticipantContextPath(focusFile)
		if matchedPath, ok := context.matchStrongDirectory(focusPath); ok {
			candidate.Score += 7
			candidate.Rationale = append(candidate.Rationale, "task context matched focus directory: "+pathpkg.Dir(matchedPath))
			return
		}
	}
}

func participantFocusFiles(candidate ParticipantSelectionCandidate) []string {
	var files []string
	for _, item := range candidate.Rationale {
		if file, ok := strings.CutPrefix(item, "focus file: "); ok {
			files = append(files, strings.TrimSpace(file))
		}
	}
	return files
}

func (e participantContextEvidence) matchStrongDirectory(focusPath string) (string, bool) {
	focusDir := pathpkg.Dir(focusPath)
	if focusDir == "." || focusDir == "" {
		return "", false
	}
	for contextPath := range e.strongPaths {
		if pathpkg.Dir(contextPath) == focusDir {
			return contextPath, true
		}
	}
	return "", false
}

func normalizeParticipantContextPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = filepath.ToSlash(filepath.Clean(value))
	return strings.TrimPrefix(value, "./")
}

func looksLikeRepoPath(value string) bool {
	if value == "" || strings.ContainsAny(value, " \t\n\r") {
		return false
	}
	base := pathpkg.Base(value)
	return strings.Contains(value, "/") || strings.Contains(base, ".")
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
	if !selectedParticipantKind(selected, "agent") {
		for _, candidate := range candidates {
			if candidate.Kind == "agent" && candidate.Score > 0 {
				add(candidate)
				break
			}
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

func selectedParticipantKind(selected []ParticipantSelectionCandidate, kind string) bool {
	for _, candidate := range selected {
		if candidate.Kind == kind {
			return true
		}
	}
	return false
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
