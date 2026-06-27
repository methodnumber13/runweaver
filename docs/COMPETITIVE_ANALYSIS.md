# Competitive Analysis

## Positioning

RunWeaver is not another coding agent runtime. It is a repository bootstrapper and orchestration layer for coding-agent runtimes:

- scans a repository and builds compact repo-local context;
- generates runtime metadata for OpenCode, Codex, and Claude Code plus portable `.runweaver/workflows`;
- creates domain-first swarm profiles for the current codebase;
- stores durable workflow state in `.runweaver/tmp/swarm-runs`;
- verifies plan/checkpoint/todo/event consistency before a swarm run is considered done.

The closest market category is "agent workflow/context tooling for coding agents", not "AI editor" and not "general multi-agent framework".

## Comparator Matrix

| Tool | What it is | Stronger than RunWeaver | Weaker / less direct fit |
| --- | --- | --- | --- |
| OpenCode | Open source AI coding agent with primary agents, subagents, project `.opencode/agents`, permissions, and model/provider config. | It is the actual runtime. It has the UI, agent loop, tools, providers, and Desktop/CLI surfaces. | It does not currently generate repository-specific swarms, durable workflow plans, or drift-aware skills by itself. RunWeaver should integrate with it, not compete with it. |
| Claude Code Workflows | Dynamic workflow runtime where Claude writes JavaScript orchestration scripts for many subagents. | More native and powerful inside Claude Code; background workflow execution is built into the product. | Vendor-specific. RunWeaver stores portable repo-local workflow artifacts that can be used across supported runtimes. |
| Aider | Terminal pair-programming agent with strong git integration and a repo map. | Mature editing loop, strong repo-map idea, broad model support, simple UX. | It is primarily one coding loop, not a generator of project-local agents/skills/workflows for another runtime. |
| Cline | Open source coding agent across IDE, CLI, SDK, and Kanban-style parallel agents. | More polished product surface, VS Code/JetBrains integration, human approval UX, checkpoints, and broad adoption. | It is its own runtime/ecosystem. It does not focus on generating OpenCode-compatible repo-local swarm metadata. |
| Roo Code | VS Code coding agent with modes and multi-step workflows. | Strong editor UX and mode-based workflow ergonomics. | Editor-bound compared with RunWeaver's repo-local runtime bootstrap model. |
| Continue | Coding agent/check platform with context providers, codebase retrieval, and source-controlled checks. | Better PR/check positioning and codebase retrieval UX. | Current public docs emphasize checks/context providers more than repo-local swarm orchestration and durable multi-agent run state. |
| OpenHands | Full AI-driven development platform and agent SDK/runtime. | Heavier autonomous software engineering environment, sandbox/runtime focus, cloud and SDK direction. | Much larger product surface. RunWeaver is intentionally smaller: local metadata, workflow state, and runtime adapters. |
| CrewAI | General multi-agent framework with agents, crews, flows, state, persistence, and enterprise deployment. | Better if you are building production app automations or custom agent systems in code. | Not specialized for existing software repos opened in OpenCode; requires users to build the agent app rather than bootstrap repo-local coding assistants. |
| Agent Skills for Context Engineering | Skill library for context engineering, multi-agent patterns, memory, evaluation, and harness design. | Excellent reusable knowledge pack; strong context-engineering vocabulary and progressive disclosure approach. | It is primarily instructions/skills, not a CLI that scans a repository, materializes runtime metadata, and manages durable execution state. |

## What RunWeaver Should Borrow

- From Aider: keep improving compact repo maps and relevance ranking, especially symbol/dependency ranking under token budget.
- From Claude Code Workflows: keep the "workflow as rerunnable artifact" mental model, but make it OpenCode-compatible and repo-local.
- From Cline: improve visible checkpoints, approval boundaries, and cleanup of spawned runtime processes.
- From Continue: consider source-controlled policy/check definitions for review gates.
- From Agent Skills for Context Engineering: keep skills short, task-triggered, and progressively disclosed; avoid dumping large generic guidance into every agent prompt.
- From CrewAI/OpenHands: add observability and run lifecycle controls only where they help local coding workflows.

## Reuse Decision

Do not vendor code or copy repository internals from these projects for the first public release. Their useful parts are product and architecture patterns:

- Aider's repo-map/relevance-ranking idea;
- Claude Code's workflow-as-rerunnable-artifact model;
- Codex and Claude skill progressive-disclosure conventions;
- Cline's explicit Plan/Act and checkpoint UX;
- Continue's source-controlled checks/context-provider model;
- CrewAI/OpenHands observability and run lifecycle concepts.

The implementation should stay original and small: core repository intelligence plus thin runtime adapters.

## Public Differentiator

The public claim should be narrow:

> RunWeaver turns any repository into a runtime-ready swarm workspace by generating repo-specific agents, skills, workflow templates, compact context indexes, and durable run checkpoints.

Avoid broader claims like "better than Cline" or "an autonomous developer". Those products are full agent runtimes. RunWeaver's value is the missing preparation/orchestration layer around a repo.

## Post-Alpha Release Gaps

- Decide whether to keep the generic OpenAI-compatible provider as the default example or make the default provider explicitly configurable in examples.
- Add package distribution: Homebrew, npm wrapper, or GitHub Releases with prebuilt binaries.
- Add a public demo repository and a short recorded workflow.
- Add integration tests that run against a disposable sample repo and assert generated OpenCode, Codex, and Claude Code structures.
- Add a compatibility matrix for OpenCode versions once a minimum version is known.

## Sources

- OpenCode agents documentation: https://opencode.ai/docs/agents/
- OpenCode config documentation: https://opencode.ai/docs/config/
- Claude Code dynamic workflows documentation: https://code.claude.com/docs/en/workflows
- Aider repository map documentation: https://aider.chat/docs/repomap.html
- Aider GitHub repository: https://github.com/Aider-AI/aider
- Cline GitHub repository: https://github.com/cline/cline
- Roo Code GitHub repository: https://github.com/RooCodeInc/Roo-Code
- Continue context providers documentation: https://docs.continue.dev/customize/custom-providers
- OpenHands GitHub repository: https://github.com/OpenHands/OpenHands
- CrewAI documentation: https://docs.crewai.com/
- Agent Skills for Context Engineering repository: https://github.com/muratcankoylan/Agent-Skills-for-Context-Engineering
