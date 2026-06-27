package aitools

import (
	"sort"
	"strings"
)

func surfacesByCategory(index RepoIndex, category, kind string) []ClassifiedSurface {
	files := filesByCategory(index, category)
	if len(files) == 0 {
		return nil
	}
	return []ClassifiedSurface{{
		Name:       category,
		Kind:       kind,
		Files:      files,
		Evidence:   []string{"repo-index.files.category=" + category},
		Confidence: "high",
	}}
}

func classifyDomains(index RepoIndex) []DomainClassification {
	grouped := map[string][]string{}
	for _, file := range index.Files {
		if file.Generated || !strings.HasPrefix(file.Path, "src/") {
			continue
		}
		parts := strings.Split(file.Path, "/")
		if len(parts) < 3 {
			continue
		}
		domain := parts[1]
		if domain == "common" || domain == "constants" || domain == "types" {
			continue
		}
		grouped[domain] = append(grouped[domain], file.Path)
	}
	var domains []DomainClassification
	for name, files := range grouped {
		files = Limit(Unique(files), 40)
		domains = append(domains, DomainClassification{
			Name:        name,
			Description: domainDescription(name),
			Kind:        domainKind(name),
			Files:       files,
			Evidence:    []string{"src/" + name + "/*"},
			Confidence:  "high",
		})
	}
	sort.Slice(domains, func(i, j int) bool {
		return domains[i].Name < domains[j].Name
	})
	return domains
}

func classifyExternalSystems(index RepoIndex, domains []DomainClassification) []ClassifiedSurface {
	var out []ClassifiedSurface
	for _, domain := range domains {
		switch domain.Name {
		case "auth", "scm", "kubernetes", "devops", "catalog-service", "deployment-service", "messaging", "mail", "observability", "templates", "object-storage", "cache":
			out = append(out, ClassifiedSurface{
				Name:       domain.Name,
				Kind:       domainKind(domain.Name),
				Files:      domain.Files,
				Evidence:   domain.Evidence,
				Confidence: "high",
			})
		}
	}
	return out
}
