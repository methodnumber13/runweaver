package aitools

// DriftReport lists stale generated anchors and metadata refresh suggestions.
type DriftReport struct {
	SchemaVersion   int      `json:"schemaVersion"`
	GeneratedAt     string   `json:"generatedAt"`
	RepoRoot        string   `json:"repoRoot"`
	StaleAnchors    []Anchor `json:"staleAnchors"`
	MissingSurfaces []string `json:"missingSurfaces"`
	Recommendations []string `json:"recommendations"`
}

// Anchor identifies generated metadata that no longer matches current code.
type Anchor struct {
	File      string `json:"file"`
	Anchor    string `json:"anchor"`
	Reason    string `json:"reason"`
	Suggested string `json:"suggested,omitempty"`
}
