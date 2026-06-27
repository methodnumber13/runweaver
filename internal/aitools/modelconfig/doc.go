// Package modelconfig inspects model provider configuration for AI coding
// runtimes.
//
// It checks project-local, user-global, managed, environment-provided, and auth
// file locations for a provider, model, base URL, and usable credential. The
// package reports both the effective detected state and every inspected file so
// CLI diagnostics can explain exactly why a runtime is ready or blocked.
//
// The package does not call model providers. It only validates local discovery
// and credential wiring.
package modelconfig
