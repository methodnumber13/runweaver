// Package runtimeenv prepares environment variables for external AI coding
// runtimes.
//
// Its main responsibility is preserving the caller's environment while adding
// provider hosts to NO_PROXY/no_proxy when a runtime needs to reach a local or
// private OpenAI-compatible endpoint. The helpers are pure functions where
// possible so command builders can test environment mutations without launching
// a runtime.
package runtimeenv
