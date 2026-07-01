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
	taskScope := scoreParticipantCandidates(candidates, task, workflow, contextEvidence)
	candidates, availabilityWarnings := filterUnavailableRuntimeParticipants(root, runtimeID, candidates)
	warnings = append(warnings, availabilityWarnings...)
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			if participantKindRank(candidates[i].Kind) == participantKindRank(candidates[j].Kind) {
				return candidates[i].Name < candidates[j].Name
			}
			return participantKindRank(candidates[i].Kind) < participantKindRank(candidates[j].Kind)
		}
		return candidates[i].Score > candidates[j].Score
	})
	selected := compactSelectedParticipantRationale(selectParticipantCandidates(candidates, cap, taskScope))
	assignments := participantAssignments(selected)
	names := make([]string, 0, len(selected))
	rationale := make([]string, 0, len(selected))
	selectedSet := map[string]bool{}
	for _, candidate := range selected {
		names = append(names, candidate.Name)
		selectedSet[candidate.Name] = true
		if len(candidate.Rationale) > 0 {
			rationale = append(rationale, candidate.Name+": "+participantRationaleSummary(candidate.Rationale))
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
	return repoPathHasSuffix(filepath.ToSlash(filepath.Clean(root)), normalizedDir)
}

func repoPathHasSuffix(rootPath, repoDir string) bool {
	rootParts := splitCleanPath(rootPath)
	repoParts := splitCleanPath(repoDir)
	if len(repoParts) == 0 || len(repoParts) > len(rootParts) {
		return false
	}
	offset := len(rootParts) - len(repoParts)
	for index := range repoParts {
		if rootParts[offset+index] != repoParts[index] {
			return false
		}
	}
	return true
}

func splitCleanPath(value string) []string {
	value = strings.Trim(filepath.ToSlash(filepath.Clean(value)), "/")
	if value == "" || value == "." {
		return nil
	}
	return strings.Split(value, "/")
}

func filterUnavailableRuntimeParticipants(root, runtimeID string, candidates []ParticipantSelectionCandidate) ([]ParticipantSelectionCandidate, []string) {
	out := make([]ParticipantSelectionCandidate, 0, len(candidates))
	var warnings []string
	for _, candidate := range candidates {
		ok, relPath := runtimeParticipantMetadataAvailable(root, runtimeID, candidate)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("skipped %s: %s %s metadata file is missing at %s", candidate.Name, runtimeID, candidate.Kind, relPath))
			continue
		}
		out = append(out, candidate)
	}
	return out, warnings
}

func runtimeParticipantMetadataAvailable(root, runtimeID string, candidate ParticipantSelectionCandidate) (bool, string) {
	if candidate.Name == "" || candidate.Kind == "" {
		return true, ""
	}
	if candidate.Source != "repo" {
		return true, ""
	}
	baseDir, relPath := runtimeParticipantMetadataPath(runtimeID, candidate.Kind, candidate.Name)
	if baseDir == "" || relPath == "" {
		return true, ""
	}
	if !Exists(filepath.Join(root, baseDir)) {
		return true, relPath
	}
	return Exists(filepath.Join(root, relPath)), relPath
}

func runtimeParticipantMetadataPath(runtimeID, kind, name string) (string, string) {
	switch runtimeID {
	case RuntimeCodex:
		if kind == "skill" {
			return ".agents/skills", filepath.ToSlash(filepath.Join(".agents", "skills", name, "SKILL.md"))
		}
		if kind == "agent" {
			return ".codex/agents", filepath.ToSlash(filepath.Join(".codex", "agents", name+".toml"))
		}
	case RuntimeClaude:
		if kind == "skill" {
			return ".claude/skills", filepath.ToSlash(filepath.Join(".claude", "skills", name, "SKILL.md"))
		}
		if kind == "agent" {
			return ".claude/agents", filepath.ToSlash(filepath.Join(".claude", "agents", name+".md"))
		}
	default:
		if kind == "skill" {
			return ".opencode/skills", filepath.ToSlash(filepath.Join(".opencode", "skills", name, "SKILL.md"))
		}
		if kind == "agent" {
			return ".opencode/agents", filepath.ToSlash(filepath.Join(".opencode", "agents", name+".md"))
		}
	}
	return "", ""
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
	scope := inferSurfaceScope("", description, focusFiles)
	for _, command := range verificationForSurface(verification, scope, "") {
		evidence = append(evidence, "verification: "+command)
	}
	return evidence
}

func sanitizeParticipantRationale(rationale []string, scope participantTaskScope) []string {
	out := make([]string, 0, len(rationale))
	for _, item := range rationale {
		if command, ok := strings.CutPrefix(item, "verification: "); ok && !participantVerificationRelevant(command, scope) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func participantVerificationRelevant(command string, scope participantTaskScope) bool {
	if !(scope.FrontendOnly || scope.BackendForbidden) {
		return true
	}
	return !verificationTargetsNonFrontendSurface(command)
}

func verificationTargetsNonFrontendSurface(command string) bool {
	lower := strings.ToLower(command)
	for _, marker := range []string{
		"src/api/",
		"backend",
		"database",
		"runtime",
		"runtime worker",
		"worker",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
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

func compactSelectedParticipantRationale(selected []ParticipantSelectionCandidate) []ParticipantSelectionCandidate {
	out := make([]ParticipantSelectionCandidate, len(selected))
	for index, candidate := range selected {
		candidate.Rationale = compactParticipantRationale(candidate.Rationale)
		out[index] = candidate
	}
	return out
}

func compactParticipantRationale(rationale []string) []string {
	var descriptions, matches, focus, workflow, verification, other []string
	for _, item := range rationale {
		switch {
		case strings.HasPrefix(item, "description: "):
			descriptions = append(descriptions, compactRationaleItem(item, 120))
		case strings.HasPrefix(item, "verification: "):
			verification = append(verification, compactRationaleItem(item, 140))
		case strings.HasPrefix(item, "focus file: "):
			focus = append(focus, compactRationaleItem(item, 120))
		case strings.HasPrefix(item, "workflow: "):
			workflow = append(workflow, compactRationaleItem(item, 120))
		case strings.Contains(item, "matched") ||
			strings.Contains(item, "listed by workflow") ||
			strings.Contains(item, "frontend task") ||
			strings.Contains(item, "backend task") ||
			strings.Contains(item, "post-implementation") ||
			strings.Contains(item, "penalized"):
			matches = append(matches, compactRationaleItem(item, 120))
		default:
			other = append(other, compactRationaleItem(item, 120))
		}
	}
	var compact []string
	compact = append(compact, Limit(descriptions, 1)...)
	compact = append(compact, Limit(verification, 1)...)
	compact = append(compact, Limit(matches, 2)...)
	if len(compact) < 3 {
		compact = append(compact, Limit(focus, 1)...)
	}
	if len(compact) < 3 {
		compact = append(compact, Limit(workflow, 1)...)
	}
	if len(compact) < 3 {
		compact = append(compact, Limit(other, 1)...)
	}
	return Unique(compact)
}

func participantRationaleSummary(rationale []string) string {
	joined := strings.Join(rationale, "; ")
	if len(joined) <= 220 {
		return joined
	}
	return compactRationaleItem(joined, 220)
}

func compactRationaleItem(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
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

func scoreParticipantCandidates(candidates []ParticipantSelectionCandidate, task string, workflow WorkflowSpec, context participantContextEvidence) participantTaskScope {
	taskTokens := tokenizeSelectionText(task)
	taskScope := participantTaskScopeFromTask(task, taskTokens, workflow)
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
		scoreParticipantDomainName(candidate, taskTokens, taskScope)
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
		scoreParticipantTaskScope(candidate, taskScope)
		candidate.Rationale = sanitizeParticipantRationale(Unique(candidate.Rationale), taskScope)
	}
	return taskScope
}

func scoreParticipantDomainName(candidate *ParticipantSelectionCandidate, taskTokens map[string]bool, scope participantTaskScope) {
	text := strings.ToLower(candidate.Name + " " + strings.Join(candidate.Rationale, " "))
	if (scope.FrontendOnly || scope.BackendForbidden) && participantLooksBackend(text) {
		return
	}
	nameTokens := tokenizeSelectionText(candidate.Name)
	matches := 0
	for token := range nameTokens {
		if !taskTokens[token] || genericParticipantDomainToken(token) {
			continue
		}
		matches++
	}
	if matches < 2 {
		return
	}
	boost := 14
	if candidate.Kind == "skill" {
		boost = 20
	}
	candidate.Score += boost
	candidate.Rationale = append(candidate.Rationale, "domain-specific participant name matched task domain")
}

func genericParticipantDomainToken(token string) bool {
	switch token {
	case "agent", "code", "context", "delivery", "engineer", "feature", "implementation", "master", "page", "repo", "repository", "review", "reviewer", "skill", "surface", "system", "test", "tooling", "ui", "workflow":
		return true
	default:
		return false
	}
}

type participantTaskScope struct {
	FrontendOnly            bool
	FrontendTask            bool
	BackendTask             bool
	BackendForbidden        bool
	ClientBoundaryTask      bool
	ClientBoundaryForbidden bool
	FeatureWorkflow         bool
	Implementation          bool
	MetadataTask            bool
	TestTask                bool
}

func participantTaskScopeFromTask(task string, taskTokens map[string]bool, workflow WorkflowSpec) participantTaskScope {
	frontendTask := hasAnySelectionToken(taskTokens, []string{
		"browser", "card", "client", "component", "dialog", "frontend", "modal", "page", "screen", "ui", "view",
	})
	backendTask := hasAnySelectionToken(taskTokens, []string{
		"api", "backend", "controller", "database", "endpoint", "handler", "migration", "persistence", "repository", "route", "routes", "server", "storage",
	})
	backendForbidden := taskForbidsBackendChanges(task)
	clientBoundaryTask := taskRequestsClientBoundaryWork(task, taskTokens)
	clientBoundaryForbidden := taskForbidsClientBoundaryChanges(task)
	testTask := hasAnySelectionToken(taskTokens, []string{
		"coverage", "e2e", "fixture", "fixtures", "mock", "mocks", "spec", "test", "tests",
	})
	featureWorkflow := workflow.ID == "feature-delivery-swarm" || workflow.ID == "feature-swarm"
	implementation := featureWorkflow || hasAnySelectionToken(taskTokens, []string{
		"add", "build", "create", "feature", "implement", "implementation", "new", "ship",
	})
	metadataTask := isMetadataRefreshTask(taskTokens)
	return participantTaskScope{
		FrontendOnly:            frontendTask && (backendForbidden || !backendTask),
		FrontendTask:            frontendTask,
		BackendTask:             backendTask,
		BackendForbidden:        backendForbidden,
		ClientBoundaryTask:      clientBoundaryTask,
		ClientBoundaryForbidden: clientBoundaryForbidden,
		FeatureWorkflow:         featureWorkflow,
		Implementation:          implementation,
		MetadataTask:            metadataTask,
		TestTask:                testTask,
	}
}

func scoreParticipantTaskScope(candidate *ParticipantSelectionCandidate, scope participantTaskScope) {
	text := strings.ToLower(candidate.Name + " " + strings.Join(candidate.Rationale, " "))
	if scope.FrontendTask && participantLooksFrontend(text) {
		candidate.Score += 10
		candidate.Rationale = append(candidate.Rationale, "frontend task matched frontend participant scope")
	}
	if scope.BackendTask && !scope.BackendForbidden && participantLooksBackend(text) {
		candidate.Score += 6
		candidate.Rationale = append(candidate.Rationale, "backend task matched backend participant scope")
	}
	if scope.ClientBoundaryTask && participantLooksClientBoundary(text) {
		candidate.Score += 12
		candidate.Rationale = append(candidate.Rationale, "client-boundary task matched client contract participant")
	}
	if scope.ClientBoundaryForbidden && participantLooksClientBoundary(text) {
		candidate.Score -= 60
		candidate.Rationale = append(candidate.Rationale, "penalized client contract participant for forbidden boundary task")
	}
	if scope.FrontendOnly && !participantLooksClientBoundary(text) && participantLooksBackend(text) {
		penalty := 26
		if scope.BackendForbidden {
			penalty = 60
		}
		candidate.Score -= penalty
		candidate.Rationale = append(candidate.Rationale, "penalized backend/API participant for frontend-only task")
	}
	if scope.FrontendOnly && scope.BackendForbidden && participantLooksBackendTestHarness(text) {
		candidate.Score -= 50
		candidate.Rationale = append(candidate.Rationale, "penalized backend/API test harness for backend-forbidden frontend task")
	}
	if scope.Implementation && !scope.BackendForbidden && participantLooksBackend(text) && !participantLooksFrontend(text) {
		candidate.Score -= 12
		candidate.Rationale = append(candidate.Rationale, "penalized backend/API participant for implementation task without backend evidence")
	}
	if scope.Implementation && !scopeRequestsTests(scope) && participantLooksTestOnly(text) {
		candidate.Score -= 30
		candidate.Rationale = append(candidate.Rationale, "penalized test/harness participant for implementation task without explicit test request")
	}
	if scope.Implementation && !scope.MetadataTask && participantLooksMetadataMaintenance(text) {
		candidate.Score -= 40
		candidate.Rationale = append(candidate.Rationale, "penalized metadata-maintenance participant for non-metadata implementation task")
	}
	if candidate.Name == "post-implementation-reviewer" && scope.Implementation {
		candidate.Score += 16
		candidate.Rationale = append(candidate.Rationale, "post-implementation review required for implementation workflow")
	}
}

func scopeRequestsTests(scope participantTaskScope) bool {
	return scope.TestTask
}

func participantLooksFrontend(text string) bool {
	return strings.Contains(text, "frontend") ||
		strings.Contains(text, "ui") ||
		strings.Contains(text, "component") ||
		strings.Contains(text, "dialog") ||
		strings.Contains(text, "card") ||
		strings.Contains(text, "page") ||
		strings.Contains(text, "browser")
}

func participantLooksBackend(text string) bool {
	if strings.Contains(text, "post-implementation-reviewer") {
		return false
	}
	if participantLooksClientBoundary(text) {
		return false
	}
	return strings.Contains(text, "api-route") ||
		strings.Contains(text, "api-security") ||
		strings.Contains(text, "api code") ||
		strings.Contains(text, "api boundary") ||
		strings.Contains(text, "backend") ||
		strings.Contains(text, "controller") ||
		strings.Contains(text, "database") ||
		strings.Contains(text, "endpoint") ||
		strings.Contains(text, "handler") ||
		strings.Contains(text, "middleware") ||
		strings.Contains(text, "migration") ||
		strings.Contains(text, "persistence") ||
		strings.Contains(text, "repository") ||
		strings.Contains(text, "runtime") ||
		strings.Contains(text, "server") ||
		strings.Contains(text, "storage") ||
		strings.Contains(text, "worker") ||
		strings.Contains(text, "/api/") ||
		strings.Contains(text, " route") ||
		strings.Contains(text, " routes")
}

func participantLooksClientBoundary(text string) bool {
	return textLooksClientBoundarySurface(text)
}

func participantLooksTestOnly(text string) bool {
	if strings.Contains(text, "post-implementation-reviewer") {
		return false
	}
	if strings.Contains(text, "test-quality") || strings.Contains(text, "quality-gates") || strings.Contains(text, "style-tooling") {
		return false
	}
	return strings.Contains(text, "test-harness") ||
		strings.Contains(text, "harness") ||
		strings.Contains(text, " route tests") ||
		strings.Contains(text, "integration harness") ||
		strings.Contains(text, "fixtures") ||
		strings.Contains(text, "mocks")
}

func participantLooksBackendTestHarness(text string) bool {
	return strings.Contains(text, "api-test-harness") ||
		strings.Contains(text, "route tests") ||
		strings.Contains(text, "integration harness") ||
		strings.Contains(text, "backend test") ||
		strings.Contains(text, "api test") ||
		strings.Contains(text, "runtime test") ||
		strings.Contains(text, "worker test")
}

func participantLooksMetadataMaintenance(text string) bool {
	return strings.Contains(text, "agent-skill-drift") ||
		strings.Contains(text, "metadata-refresh") ||
		strings.Contains(text, "profile-regenerator") ||
		strings.Contains(text, "stale-anchor") ||
		strings.Contains(text, "runtime metadata") ||
		strings.Contains(text, "generated runtime metadata")
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

func selectParticipantCandidates(candidates []ParticipantSelectionCandidate, cap int, scope participantTaskScope) []ParticipantSelectionCandidate {
	if cap <= 0 {
		cap = defaultParticipantCap
	}
	selected := make([]ParticipantSelectionCandidate, 0, cap)
	add := func(candidate ParticipantSelectionCandidate) bool {
		if len(selected) >= cap {
			return false
		}
		if participantExcludedByTaskScope(candidate, scope) {
			return false
		}
		for _, existing := range selected {
			if existing.Name == candidate.Name {
				return false
			}
		}
		selected = append(selected, candidate)
		return true
	}
	for _, candidate := range candidates {
		if participantCanOwn(candidate, scope) && candidate.Source == "repo" && candidate.Score >= 8 {
			if add(candidate) {
				break
			}
		}
	}
	if !selectedParticipantKind(selected, "agent") {
		for _, candidate := range candidates {
			if participantCanOwn(candidate, scope) && candidate.Score > 0 {
				if add(candidate) {
					break
				}
			}
		}
	}
	for _, candidate := range candidates {
		if candidate.Kind == "skill" && candidate.Score >= 8 && participantIsTaskSpecialist(candidate) {
			add(candidate)
		}
	}
	for _, candidate := range candidates {
		if candidate.Name == "post-implementation-reviewer" && candidate.Score > 0 {
			if add(candidate) {
				break
			}
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

func participantExcludedByTaskScope(candidate ParticipantSelectionCandidate, scope participantTaskScope) bool {
	text := strings.ToLower(candidate.Name + " " + strings.Join(candidate.Rationale, " "))
	if scope.FrontendOnly && scope.BackendForbidden && participantLooksClientBoundary(text) {
		return scope.ClientBoundaryForbidden || !scope.ClientBoundaryTask
	}
	if scope.FrontendOnly && scope.BackendForbidden && participantLooksBackend(text) {
		return true
	}
	if scope.FrontendOnly && scope.BackendForbidden && participantLooksBackendTestHarness(text) {
		return true
	}
	return false
}

func participantCanOwn(candidate ParticipantSelectionCandidate, scope participantTaskScope) bool {
	if candidate.Kind != "agent" {
		return false
	}
	if !scope.Implementation {
		return true
	}
	text := strings.ToLower(candidate.Name + " " + strings.Join(candidate.Rationale, " "))
	if candidate.Name == "post-implementation-reviewer" {
		return false
	}
	if strings.Contains(text, "reviewer") && !strings.Contains(text, "engineer") {
		return false
	}
	return !participantLooksTestOnly(text)
}

func participantIsTaskSpecialist(candidate ParticipantSelectionCandidate) bool {
	text := strings.ToLower(candidate.Name + " " + strings.Join(candidate.Rationale, " "))
	if strings.Contains(candidate.Name, "quality-gates") || strings.Contains(candidate.Name, "style-tooling") || strings.Contains(candidate.Name, "test-tooling") {
		return false
	}
	return !participantLooksTestOnly(text)
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
