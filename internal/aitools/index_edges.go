package aitools

import (
	"strings"
)

func testEdges(files []FileInventoryItem) []IndexEdge {
	exists := map[string]bool{}
	for _, file := range files {
		exists[file.Path] = true
	}
	var edges []IndexEdge
	for _, file := range files {
		if file.Category != "test" {
			continue
		}
		candidates := possibleTestTargets(file.Path)
		for _, candidate := range candidates {
			if candidate != file.Path && exists[candidate] {
				edges = append(edges, IndexEdge{From: file.Path, To: candidate, Kind: "tests", Reason: "matched nearby source filename"})
				break
			}
		}
	}
	return edges
}

func possibleTestTargets(path string) []string {
	replacements := []string{".test.", ".spec.", "_test.", "Test."}
	var out []string
	for _, marker := range replacements {
		if strings.Contains(path, marker) {
			out = append(out, strings.Replace(path, marker, ".", 1))
		}
	}
	out = append(out, strings.Replace(path, "src/test/", "src/main/", 1))
	out = append(out, strings.Replace(path, "/__tests__/", "/", 1))
	return Unique(out)
}
