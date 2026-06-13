// Package runtimecatalog defines static metadata for supported AI coding
// runtimes.
//
// Runtime metadata is intentionally separated from the heavier runtime adapters.
// Catalog adapters describe stable IDs, binary names, generated profile paths,
// managed paths, capability flags, and prompt guidance. The core runtime layer
// wraps this metadata with filesystem discovery, rendering, and command
// execution behavior.
package runtimecatalog
