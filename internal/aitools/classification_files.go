package aitools

import (
	"strings"
)

func filesByCategory(index RepoIndex, category string) []string {
	var out []string
	for _, file := range index.Files {
		if file.Generated {
			continue
		}
		if file.Category == category {
			out = append(out, file.Path)
		}
	}
	return Limit(Unique(out), 80)
}

func filterFiles(index RepoIndex, contains string) []string {
	var out []string
	for _, file := range index.Files {
		if !file.Generated && strings.Contains(file.Path, contains) {
			out = append(out, file.Path)
		}
	}
	return Limit(Unique(out), 80)
}

func configBoundaryFiles(index RepoIndex) []string {
	files := mergeFileLists(filesByCategory(index, "entrypoint"), filesByCategory(index, "module"), filesByCategory(index, "config"))
	for _, wanted := range []string{"src/common/configs/swagger/set-swagger.ts", "src/constants/bootstrap.ts", ".env.example"} {
		if fileSet(index)[wanted] {
			files = append(files, wanted)
		}
	}
	return Limit(Unique(files), 80)
}

func externalSystemFiles(classification RepoClassification) []string {
	var files []string
	for _, surface := range classification.ExternalSystems {
		files = append(files, surface.Files...)
	}
	return Limit(Unique(files), 120)
}

func domainFiles(classification RepoClassification, domainName string) []string {
	for _, domain := range classification.Domains {
		if domain.Name == domainName {
			return domain.Files
		}
	}
	return nil
}

func filesForDomains(classification RepoClassification, names ...string) []string {
	var files []string
	for _, name := range names {
		files = append(files, domainFiles(classification, name)...)
	}
	return Limit(Unique(files), 100)
}

func mergeFileLists(groups ...[]string) []string {
	var out []string
	for _, group := range groups {
		out = append(out, group...)
	}
	return Limit(Unique(out), 120)
}

func fileSet(index RepoIndex) map[string]bool {
	out := map[string]bool{}
	for _, file := range index.Files {
		out[file.Path] = true
	}
	return out
}
