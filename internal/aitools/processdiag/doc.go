// Package processdiag inspects local developer-tool process state.
//
// RunWeaver uses these diagnostics to spot duplicate AI runtime processes and
// IDE settings that can accidentally attach debuggers to every Node.js process.
// The package accepts live ps output or captured process text, which keeps the
// parser deterministic in tests and usable in CLI doctor commands.
package processdiag
