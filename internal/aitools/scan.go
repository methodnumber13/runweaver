package aitools

import (
	"os"
	"path/filepath"
	"sort"
)

type packageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// Scan creates a lightweight repository surface map without writing artifacts.
func Scan(repoPath string) (SurfaceIndex, error) {
	root, err := RepoRoot(repoPath)
	if err != nil {
		return SurfaceIndex{}, err
	}

	var files []string
	var walkWarnings []string
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			walkWarnings = append(walkWarnings, "walk skipped "+rel(root, path)+": "+walkErr.Error())
			return nil
		}
		if entry.IsDir() {
			if path != root && shouldSkipWalkDir(root, path, entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		files = append(files, rel(root, path))
		return nil
	})
	if err != nil {
		return SurfaceIndex{}, err
	}
	sort.Strings(files)

	stack := detectStack(root, files)
	index := SurfaceIndex{
		SchemaVersion: 1,
		GeneratedAt:   Now(),
		RepoRoot:      root,
		Stack:         stack,
		ConfigFiles:   configFiles(files),
		SourceDirs:    sourceDirs(files),
		TestDirs:      testDirs(files),
		EntryPoints:   entryPoints(files, stack),
		Routes:        routeFiles(files),
		Pages:         pageFiles(files),
	}
	index.BuildCommands = buildCommands(root, index.Stack)
	packages, packageWarnings := DetectPackagesWithWarnings(root, files)
	index.Packages = packages
	index.Tools = BuildToolchain(index.Stack, index.Packages, index.BuildCommands)
	index.Warnings = append(warnings(index), Limit(walkWarnings, 40)...)
	index.Warnings = append(index.Warnings, packageWarnings...)
	return index, nil
}
