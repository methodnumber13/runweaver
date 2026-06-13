package jsonc

import (
	"encoding/json"
	"testing"
)

func TestStripCommentsPreservesStringsAndRemovesJSONCComments(t *testing.T) {
	raw := []byte(`{
  // comment
  "url": "https://example.com/path//still-string",
  "pattern": "/* still string */",
  /* block comment */
  "enabled": true
}`)

	var parsed map[string]any
	if err := json.Unmarshal(StripComments(raw), &parsed); err != nil {
		t.Fatalf("unmarshal stripped JSONC: %v", err)
	}
	if parsed["url"] != "https://example.com/path//still-string" {
		t.Fatalf("url = %#v", parsed["url"])
	}
	if parsed["pattern"] != "/* still string */" {
		t.Fatalf("pattern = %#v", parsed["pattern"])
	}
	if parsed["enabled"] != true {
		t.Fatalf("enabled = %#v", parsed["enabled"])
	}
}
