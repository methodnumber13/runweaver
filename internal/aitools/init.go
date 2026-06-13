package aitools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// InitResult summarizes files and workflow state created by initialization.
type InitResult struct {
	Status           string                   `json:"status"`
	Runtime          string                   `json:"runtime"`
	Runtimes         []RuntimeDiscoveryResult `json:"runtimes,omitempty"`
	IndexPath        string                   `json:"indexPath"`
	ProfileGenerated string                   `json:"profileGenerated"`
	SurfaceIndexPath string                   `json:"surfaceIndexPath"`
	DriftReportPath  string                   `json:"driftReportPath"`
	ModelPreflight   ModelConfigCheck         `json:"modelPreflight"`
	Classifier       ClassifyRunSummary       `json:"classifier,omitempty"`
	IntelligenceRun  WorkflowRunSummary       `json:"intelligenceRun"`
}

// InitOptions configures first-time repository bootstrap.
type InitOptions struct {
	Force          bool
	RequireModel   bool
	Runtime        string
	ModelCheck     ModelConfigCheckOptions
	Classification ClassifyOptions
	Progress       InitProgressReporter
}

// InitProgressEvent reports one user-visible initialization step.
type InitProgressEvent struct {
	Current int
	Total   int
	Step    string
	Message string
	Elapsed time.Duration
	Pulse   int
}

// InitProgressReporter receives initialization progress events.
type InitProgressReporter func(InitProgressEvent)

const initProgressTotal = 8

func reportInitProgress(opts InitOptions, current int, step, message string) {
	if opts.Progress == nil {
		return
	}
	opts.Progress(InitProgressEvent{
		Current: current,
		Total:   initProgressTotal,
		Step:    step,
		Message: message,
	})
}

// Init bootstraps default RunWeaver metadata using deterministic classification.
func Init(repoPath string, force bool) error {
	_, err := InitSmart(repoPath, force)
	return err
}

// InitSmart bootstraps a repo with indexing, classification, and workflow planning.
func InitSmart(repoPath string, force bool) (InitResult, error) {
	return InitSmartWithOptions(repoPath, InitOptions{Force: force})
}

// InitSmartWithOptions bootstraps a repo with explicit runtime/classifier options.
func InitSmartWithOptions(repoPath string, opts InitOptions) (InitResult, error) {
	reportInitProgress(opts, 1, "resolve-repo", "Resolving repository root")
	root, err := RepoRoot(repoPath)
	if err != nil {
		return InitResult{}, err
	}
	providers, err := ResolveRuntimeSelection(opts.Runtime)
	if err != nil {
		return InitResult{}, err
	}
	runtimeIDs := runtimeProviderIDs(providers)
	runtimeSelection := runtimeSelectionString(runtimeIDs)
	opts.Classification = defaultInitClassificationRuntime(opts.Classification, runtimeIDs)
	cleanup := trackOpenCodeDependencyArtifacts(root)
	defer cleanup()
	reportInitProgress(opts, 2, "model-preflight", "Checking runtime provider discovery and OpenCode model config when needed")
	runtimes, err := DiscoverRuntimes(root, RuntimeDiscoveryOptions{Runtime: runtimeSelection})
	if err != nil {
		return InitResult{}, err
	}
	modelPreflight := skippedModelPreflight(opts.ModelCheck.ProviderID)
	if shouldRunOpenCodeModelPreflight(opts, runtimeIDs) {
		modelPreflight, err = CheckModelConfig(root, opts.ModelCheck)
		if err != nil {
			return InitResult{}, err
		}
	}
	if opts.RequireModel && !modelPreflight.Ready {
		return InitResult{}, fmt.Errorf("OpenCode model preflight failed: %s", strings.Join(modelPreflight.Issues, "; "))
	}
	dirs := runtimeBaselineDirs(runtimeIDs)
	reportInitProgress(opts, 3, "create-directories", "Creating RunWeaver core and runtime provider directories")
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			return InitResult{}, err
		}
	}
	files := runtimeBaselineFiles(runtimeIDs)
	reportInitProgress(opts, 4, "write-baseline", "Writing RunWeaver core workflows and selected runtime provider metadata")
	for name, content := range files {
		if err := writeIfAllowed(filepath.Join(root, name), content, opts.Force); err != nil {
			return InitResult{}, err
		}
	}
	reportInitProgress(opts, 5, "index-classify", "Indexing repository and running "+initClassifierSummary(opts.Classification))
	repoIndex, err := IndexWithOptions(root, IndexOptions{ChangedOnly: true, Prune: true, MaxCacheMB: 256, Classification: opts.Classification})
	if err != nil {
		return InitResult{}, err
	}
	reportInitProgress(opts, 6, "plan-workflow", "Creating durable repo intelligence workflow checkpoint")
	intelligenceRun, err := PlanWorkflow(root, ".runweaver/workflows/repo-intelligence-swarm.json", "smart init repository intelligence scan")
	if err != nil {
		return InitResult{}, err
	}
	reportInitProgress(opts, 7, "materialize-profile", "Generating swarm profile and materializing repo-specific agents and skills")
	profile := GenerateProfileFromIndex(repoIndex)
	profileJSON, err := safeJSON(profile)
	if err != nil {
		return InitResult{}, err
	}
	for _, path := range runtimeProfilePaths(runtimeIDs) {
		if err := writeIfAllowed(filepath.Join(root, path), profileJSON, opts.Force); err != nil {
			return InitResult{}, err
		}
	}
	if err := MaterializeProfileForRuntimes(root, profile, opts.Force, runtimeIDs); err != nil {
		return InitResult{}, err
	}
	reportInitProgress(opts, 8, "refresh-drift", "Refreshing surface index and drift report")
	refresh, err := RefreshWithOptions(root, RefreshOptions{
		Apply: false,
		Classification: ClassifyOptions{
			Mode: ClassificationDeterministic,
		},
	})
	if err != nil {
		return InitResult{}, err
	}
	if err := restoreInitClassificationArtifacts(root, repoIndex); err != nil {
		return InitResult{}, err
	}
	classifier := ClassifyRunSummary{}
	if repoIndex.ClassifierRun != nil {
		classifier = *repoIndex.ClassifierRun
	}
	return InitResult{
		Status:           "initialized",
		Runtime:          runtimeSelection,
		Runtimes:         runtimes,
		IndexPath:        repoIndex.Artifacts.Index,
		ProfileGenerated: refresh.ProfilePath,
		SurfaceIndexPath: refresh.SurfaceIndexPath,
		DriftReportPath:  refresh.DriftReportPath,
		ModelPreflight:   modelPreflight,
		Classifier:       classifier,
		IntelligenceRun:  intelligenceRun,
	}, nil
}

func restoreInitClassificationArtifacts(root string, repoIndex RepoIndex) error {
	if repoIndex.Artifacts.Classification != "" {
		if err := WriteJSON(filepath.Join(root, repoIndex.Artifacts.Classification), repoIndex.Classification); err != nil {
			return err
		}
	}
	if repoIndex.Artifacts.Index != "" {
		if err := WriteJSON(filepath.Join(root, repoIndex.Artifacts.Index), repoIndex); err != nil {
			return err
		}
	}
	return nil
}

func initClassifierSummary(opts ClassifyOptions) string {
	normalized, err := normalizeClassifyOptions(opts, ClassificationDeterministic)
	if err != nil {
		return "classifier setup"
	}
	if normalized.Mode == ClassificationDeterministic {
		return "deterministic classifier"
	}
	model := normalized.Model
	if model == "" {
		model = "configured default model"
	}
	return fmt.Sprintf("%s AI classifier through %s with model %s and timeout %s", normalized.Mode, normalized.Runtime, model, normalized.Timeout)
}

func defaultInitClassificationRuntime(opts ClassifyOptions, runtimeIDs []string) ClassifyOptions {
	if strings.TrimSpace(opts.Runtime) == "" && len(runtimeIDs) == 1 {
		opts.Runtime = runtimeIDs[0]
	}
	if strings.TrimSpace(opts.ApplyRuntime) == "" {
		opts.ApplyRuntime = runtimeSelectionString(runtimeIDs)
	}
	return opts
}

func writeIfAllowed(path, content string, force bool) error {
	if Exists(path) && !force {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
