package aitools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func parseRepoClassification(data []byte) (RepoClassification, error) {
	data = bytes.TrimSpace(stripANSI(data))
	candidate, err := extractFirstJSONObject(data)
	if err != nil {
		return RepoClassification{}, err
	}
	var direct RepoClassification
	if err := json.Unmarshal(candidate, &direct); err == nil && classificationHasContent(direct) {
		return direct, nil
	}
	var wrapped struct {
		Classification RepoClassification `json:"classification"`
	}
	if err := json.Unmarshal(candidate, &wrapped); err != nil {
		return RepoClassification{}, err
	}
	if !classificationHasContent(wrapped.Classification) {
		return RepoClassification{}, fmt.Errorf("classification JSON is empty")
	}
	return wrapped.Classification, nil
}

func extractFirstJSONObject(data []byte) ([]byte, error) {
	start := bytes.IndexByte(data, '{')
	if start < 0 {
		return nil, fmt.Errorf("no JSON object found")
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(data); i++ {
		ch := data[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return data[start : i+1], nil
			}
			if depth < 0 {
				return nil, fmt.Errorf("malformed JSON object boundary")
			}
		}
	}
	return nil, fmt.Errorf("unterminated JSON object")
}

func stripANSI(data []byte) []byte {
	text := string(data)
	var out strings.Builder
	inEscape := false
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if inEscape {
			if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
				inEscape = false
			}
			continue
		}
		if ch == 0x1b {
			inEscape = true
			continue
		}
		out.WriteByte(ch)
	}
	return []byte(out.String())
}
