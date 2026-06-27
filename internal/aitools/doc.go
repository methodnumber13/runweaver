// Package aitools is RunWeaver's core orchestration engine.
//
// The package turns a source repository into runtime-specific AI assistant
// metadata. Its responsibilities are:
//
//   - scan and index repository files, symbols, routes, dependencies, and
//     toolchain commands;
//   - classify repository surfaces into domains, risks, agents, skills, and
//     verification steps;
//   - render generated metadata for OpenCode, Codex, and Claude Code without
//     mixing runtime-specific details into the scanner;
//   - create durable workflow plans, checkpoints, todo lists, and event logs so
//     a swarm can resume after a model context reset;
//   - run readiness diagnostics for model configuration, runtime metadata, and
//     noisy local processes.
//
// Generated state is repo-local. Index artifacts, workflow runs, and temporary
// caches are written under .runweaver/tmp. Runtime-facing metadata is written to
// the selected provider's conventional locations, for example .opencode,
// .codex, or .claude.
//
// Typical callers start with InitSmartWithOptions for first-time bootstrap,
// IndexWithOptions for incremental indexing, ClassifyRepository for semantic
// classification, and ExecuteWorkflow for runtime-backed swarm execution.
package aitools
