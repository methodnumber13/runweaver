// Package jsonc contains the minimal JSON-with-comments support RunWeaver needs
// for developer-tool configuration files.
//
// The package is intentionally not a general JSONC parser. It strips line and
// block comments while preserving string contents, then lets encoding/json
// perform normal JSON validation. This keeps config parsing predictable and
// avoids accepting unrelated syntax.
package jsonc
