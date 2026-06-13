// Package foundation contains shared low-level helpers used by RunWeaver's
// internal packages.
//
// The helpers deliberately stay small and dependency-free: JSON read/write with
// contextual errors, repository path validation, stable timestamps, path
// normalization, list de-duplication, and generated/vendor directory filters.
// Higher-level packages should use these helpers instead of reimplementing
// path and JSON handling.
package foundation
