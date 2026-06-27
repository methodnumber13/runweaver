package aitools

// SurfaceIndex is the lightweight repository scan used before full indexing.
type SurfaceIndex struct {
	SchemaVersion int              `json:"schemaVersion"`
	GeneratedAt   string           `json:"generatedAt"`
	RepoRoot      string           `json:"repoRoot"`
	Stack         StackInfo        `json:"stack"`
	Tools         ToolchainInfo    `json:"tools,omitempty"`
	Packages      []PackageInsight `json:"packages,omitempty"`
	ConfigFiles   []string         `json:"configFiles"`
	SourceDirs    []string         `json:"sourceDirs"`
	TestDirs      []string         `json:"testDirs"`
	EntryPoints   []string         `json:"entryPoints"`
	Routes        []string         `json:"routes"`
	Pages         []string         `json:"pages"`
	BuildCommands []string         `json:"buildCommands"`
	Warnings      []string         `json:"warnings,omitempty"`
}

// StackInfo summarizes the dominant language, framework, and package manager.
type StackInfo struct {
	Kind           string   `json:"kind"`
	Languages      []string `json:"languages"`
	Frameworks     []string `json:"frameworks"`
	PackageManager string   `json:"packageManager,omitempty"`
}

// ToolchainInfo captures detected build, lint, format, and test tooling.
type ToolchainInfo struct {
	Languages           []string `json:"languages"`
	PackageManagers     []string `json:"packageManagers,omitempty"`
	Frameworks          []string `json:"frameworks,omitempty"`
	TestTools           []string `json:"testTools,omitempty"`
	Linters             []string `json:"linters,omitempty"`
	Formatters          []string `json:"formatters,omitempty"`
	BuildTools          []string `json:"buildTools,omitempty"`
	RecommendedCommands []string `json:"recommendedCommands,omitempty"`
}

// PackageInsight records a dependency and the role RunWeaver inferred for it.
type PackageInsight struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
	Version   string `json:"version,omitempty"`
	Scope     string `json:"scope,omitempty"`
	Role      string `json:"role"`
	Action    string `json:"action"`
}

// RepoIndex is the full repo-local index consumed by classifiers and runtimes.
type RepoIndex struct {
	SchemaVersion  int                 `json:"schemaVersion"`
	GeneratedAt    string              `json:"generatedAt"`
	RepoRoot       string              `json:"repoRoot"`
	Surface        SurfaceIndex        `json:"surface"`
	Tools          ToolchainInfo       `json:"tools"`
	Packages       []PackageInsight    `json:"packages"`
	Files          []FileInventoryItem `json:"files"`
	Symbols        []SymbolInfo        `json:"symbols"`
	Edges          []IndexEdge         `json:"edges"`
	Classification RepoClassification  `json:"classification,omitempty"`
	ClassifierRun  *ClassifyRunSummary `json:"classifierRun,omitempty"`
	Stats          IndexStats          `json:"stats"`
	Artifacts      IndexArtifacts      `json:"artifacts"`
	Warnings       []string            `json:"warnings,omitempty"`
}

// IndexOptions configures cache reuse, pruning, and classification behavior.
type IndexOptions struct {
	ChangedOnly    bool
	Prune          bool
	MaxCacheMB     int
	Classification ClassifyOptions
}

// FileInventoryItem describes one indexed file and its cache identity.
type FileInventoryItem struct {
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	ModTime   string `json:"modTime"`
	Hash      string `json:"hash"`
	Language  string `json:"language,omitempty"`
	Category  string `json:"category,omitempty"`
	Generated bool   `json:"generated,omitempty"`
}

// FileAnalysis is the cached per-file analysis stored by content hash.
type FileAnalysis struct {
	SchemaVersion int          `json:"schemaVersion"`
	Hash          string       `json:"hash"`
	SourcePaths   []string     `json:"sourcePaths"`
	Language      string       `json:"language,omitempty"`
	Category      string       `json:"category,omitempty"`
	Imports       []string     `json:"imports,omitempty"`
	Exports       []string     `json:"exports,omitempty"`
	Symbols       []SymbolInfo `json:"symbols,omitempty"`
	Routes        []RouteInfo  `json:"routes,omitempty"`
	Signals       []string     `json:"signals,omitempty"`
	Summary       string       `json:"summary"`
}

// SymbolInfo is a compact symbol reference extracted from a source file.
type SymbolInfo struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Path string `json:"path"`
	Line int    `json:"line,omitempty"`
}

// RouteInfo describes an API or page route discovered in source code.
type RouteInfo struct {
	Method string `json:"method,omitempty"`
	Path   string `json:"path,omitempty"`
	File   string `json:"file"`
	Line   int    `json:"line,omitempty"`
}

// IndexEdge links files, routes, tests, or other surfaces inside the index.
type IndexEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Kind   string `json:"kind"`
	Reason string `json:"reason,omitempty"`
}

// IndexStats reports indexing volume and cache effectiveness.
type IndexStats struct {
	Files        int   `json:"files"`
	Skipped      int   `json:"skipped,omitempty"`
	CacheHits    int   `json:"cacheHits"`
	CacheMisses  int   `json:"cacheMisses"`
	CachePruned  int   `json:"cachePruned,omitempty"`
	CacheEntries int   `json:"cacheEntries,omitempty"`
	CacheBytes   int64 `json:"cacheBytes,omitempty"`
	Packages     int   `json:"packages"`
	Symbols      int   `json:"symbols"`
	Edges        int   `json:"edges"`
}

// IndexArtifacts lists repo-relative paths written by the indexer.
type IndexArtifacts struct {
	Root             string `json:"root"`
	Files            string `json:"files"`
	Packages         string `json:"packages"`
	Symbols          string `json:"symbols"`
	Edges            string `json:"edges"`
	Index            string `json:"index"`
	Compact          string `json:"compact,omitempty"`
	Context          string `json:"context,omitempty"`
	Manifest         string `json:"manifest,omitempty"`
	Classification   string `json:"classification,omitempty"`
	ClassifierPrompt string `json:"classifierPrompt,omitempty"`
	ClassifierOutput string `json:"classifierOutput,omitempty"`
}

// IndexManifest records the latest index run and live cache accounting.
type IndexManifest struct {
	SchemaVersion    int            `json:"schemaVersion"`
	GeneratedAt      string         `json:"generatedAt"`
	RepoRoot         string         `json:"repoRoot"`
	Artifacts        IndexArtifacts `json:"artifacts"`
	Stats            IndexStats     `json:"stats"`
	LiveCacheEntries int            `json:"liveCacheEntries"`
	Warnings         []string       `json:"warnings,omitempty"`
}

// CompactRepoIndex is the reduced form intended for model prompts and review.
type CompactRepoIndex struct {
	SchemaVersion  int                 `json:"schemaVersion"`
	GeneratedAt    string              `json:"generatedAt"`
	RepoRoot       string              `json:"repoRoot"`
	Stack          StackInfo           `json:"stack"`
	Tools          ToolchainInfo       `json:"tools"`
	Packages       []PackageInsight    `json:"packages"`
	Files          []FileInventoryItem `json:"files"`
	Symbols        []SymbolInfo        `json:"symbols,omitempty"`
	Routes         []IndexEdge         `json:"routes,omitempty"`
	Tests          []IndexEdge         `json:"tests,omitempty"`
	Classification RepoClassification  `json:"classification,omitempty"`
	ClassifierRun  *ClassifyRunSummary `json:"classifierRun,omitempty"`
	Stats          IndexStats          `json:"stats"`
	Artifacts      IndexArtifacts      `json:"artifacts"`
	Warnings       []string            `json:"warnings,omitempty"`
}
