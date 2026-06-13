package aitools

import (
	"os"
	"path/filepath"

	"github.com/methodnumber13/runweaver/internal/aitools/statepath"
)

// RefreshResult contains regenerated index, drift, profile, and classifier data.
type RefreshResult struct {
	SurfaceIndexPath string             `json:"surfaceIndexPath"`
	DriftReportPath  string             `json:"driftReportPath"`
	ProfilePath      string             `json:"profilePath"`
	ProfilePaths     []string           `json:"profilePaths,omitempty"`
	SurfaceIndex     SurfaceIndex       `json:"surfaceIndex"`
	DriftReport      DriftReport        `json:"driftReport"`
	Classifier       ClassifyRunSummary `json:"classifier,omitempty"`
}

// RefreshOptions configures metadata refresh and optional runtime materialization.
type RefreshOptions struct {
	Apply          bool
	Runtime        string
	Classification ClassifyOptions
}

// Refresh rebuilds repo metadata and optionally applies generated runtime files.
func Refresh(repoPath string, apply bool) (RefreshResult, error) {
	return RefreshWithOptions(repoPath, RefreshOptions{
		Apply: apply,
		Classification: ClassifyOptions{
			Mode: ClassificationDeterministic,
		},
	})
}

// RefreshWithOptions rebuilds metadata with explicit runtime and classifier options.
func RefreshWithOptions(repoPath string, opts RefreshOptions) (RefreshResult, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return RefreshResult{}, err
	}
	if err := os.MkdirAll(statepath.TmpPath(root), 0o755); err != nil {
		return RefreshResult{}, err
	}
	repoIndex, err := IndexWithOptions(root, IndexOptions{ChangedOnly: true, Prune: true, MaxCacheMB: 256, Classification: opts.Classification})
	if err != nil {
		return RefreshResult{}, err
	}
	index := repoIndex.Surface
	report, err := Drift(root, index)
	if err != nil {
		return RefreshResult{}, err
	}
	profile := GenerateProfileFromIndex(repoIndex)

	indexPath := statepath.TmpPath(root, "surface-index.json")
	driftPath := statepath.TmpPath(root, "drift-report.json")
	profilePath := statepath.TmpPath(root, "profile.generated.json")
	var profilePaths []string
	if opts.Apply {
		providers, err := ResolveRuntimeSelection(opts.Runtime)
		if err != nil {
			return RefreshResult{}, err
		}
		runtimeIDs := runtimeProviderIDs(providers)
		profileJSON, err := safeJSON(profile)
		if err != nil {
			return RefreshResult{}, err
		}
		for _, path := range runtimeProfilePaths(runtimeIDs) {
			if err := writeIfAllowed(filepath.Join(root, path), profileJSON, true); err != nil {
				return RefreshResult{}, err
			}
			profilePaths = append(profilePaths, path)
		}
		if len(profilePaths) > 0 {
			profilePath = filepath.Join(root, profilePaths[0])
		}
	}
	if err := WriteJSON(indexPath, index); err != nil {
		return RefreshResult{}, err
	}
	if err := WriteJSON(driftPath, report); err != nil {
		return RefreshResult{}, err
	}
	if err := WriteJSON(profilePath, profile); err != nil {
		return RefreshResult{}, err
	}
	if opts.Apply {
		providers, err := ResolveRuntimeSelection(opts.Runtime)
		if err != nil {
			return RefreshResult{}, err
		}
		if err := MaterializeProfileForRuntimes(root, profile, true, runtimeProviderIDs(providers)); err != nil {
			return RefreshResult{}, err
		}
	}
	classifier := ClassifyRunSummary{}
	if repoIndex.ClassifierRun != nil {
		classifier = *repoIndex.ClassifierRun
	}
	return RefreshResult{
		SurfaceIndexPath: rel(root, indexPath),
		DriftReportPath:  rel(root, driftPath),
		ProfilePath:      rel(root, profilePath),
		ProfilePaths:     profilePaths,
		SurfaceIndex:     index,
		DriftReport:      report,
		Classifier:       classifier,
	}, nil
}
