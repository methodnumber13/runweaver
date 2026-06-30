package aitools

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func repoFiles(root string) ([]string, []string, error) {
	var files []string
	var warnings []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			warnings = appendIndexWarning(warnings, fmt.Sprintf("walk skipped %s: %v", rel(root, path), walkErr))
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
	sort.Strings(files)
	return files, warnings, err
}

func computeFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	buf := make([]byte, hashChunkSize)
	for {
		n, readErr := file.Read(buf)
		if n > 0 {
			if _, err := hash.Write(buf[:n]); err != nil {
				return "", err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func analyzeFile(absPath, relPath, hash, language, category string) FileAnalysis {
	analysis := FileAnalysis{
		SchemaVersion: fileAnalysisSchemaVersion,
		Hash:          hash,
		SourcePaths:   []string{relPath},
		Language:      language,
		Category:      category,
		Summary:       category + " " + language + " file",
	}
	if language == "" || isGeneratedFile(relPath) {
		return analysis
	}
	file, err := os.Open(absPath)
	if err != nil {
		return analysis
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	controllerPath := ""
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if language == "typescript" || language == "javascript" {
			if found, ok := nestControllerPathFromLine(line); ok {
				controllerPath = found
			}
		}
		analysis.Imports = append(analysis.Imports, importsFromLine(line, language)...)
		analysis.Exports = append(analysis.Exports, exportsFromLine(line, language)...)
		analysis.Symbols = append(analysis.Symbols, symbolsFromLine(line, language, relPath, lineNo)...)
		analysis.Routes = append(analysis.Routes, routesFromLine(line, language, relPath, lineNo)...)
		analysis.Routes = append(analysis.Routes, nestRoutesFromLine(line, controllerPath, language, relPath, lineNo)...)
		if lineNo > 2000 {
			analysis.Signals = append(analysis.Signals, "analysis-truncated-after-2000-lines")
			break
		}
	}
	analysis.Imports = Unique(analysis.Imports)
	analysis.Exports = Unique(analysis.Exports)
	analysis.Symbols = LimitSymbols(analysis.Symbols, 80)
	analysis.Routes = limitRoutes(analysis.Routes, 80)
	if len(analysis.Routes) > 0 {
		analysis.Signals = append(analysis.Signals, "declares-routes")
	}
	if len(analysis.Exports) > 0 {
		analysis.Signals = append(analysis.Signals, "exports-public-symbols")
	}
	if err := scanner.Err(); err != nil {
		analysis.Signals = append(analysis.Signals, "analysis-scan-error: "+err.Error())
	}
	return analysis
}

func appendIndexWarning(warnings []string, warning string) []string {
	if strings.TrimSpace(warning) == "" || len(warnings) >= 80 {
		return warnings
	}
	return append(warnings, warning)
}
