# Changelog

All notable changes to the Go line (Reasonix 1.0+) are recorded here. The legacy
`0.x` TypeScript history lives on the [`v1`](https://github.com/esengine/DeepSeek-Reasonix/tree/v1)
branch.

## Unreleased

### Added

- **Planner MCP tool opt-in**: MCP tools are now opt-in for the planner. When
  MCP servers are connected, only tools listed in the new `planner_allowed_tools`
  config are visible to the planner; all other MCP tools are blocked. Built-in
  read-only tools are unaffected. (`internal/agent/task.go`,
  `internal/config/config.go`, `internal/config/render.go`,
  `internal/boot/boot.go`)

- **Planner controls Executor invocation**: In two-model mode, when plan mode is
  active the Coordinator stops after the planner produces its plan instead of
  unconditionally running the executor. This avoids wasted tokens from the
  executor re-doing read-only research. (`internal/agent/coordinator.go`)

- **Executor summary feedback to Planner**: After each executor run, the
  Coordinator caches the executor's final assistant message and injects it as
  `[Previous execution summary]` into the planner's input on the next turn. This
  keeps the planner aware of what was actually done. The cache is cleared on
  `ResetPlannerSession` to avoid cross-session leakage.
  (`internal/agent/coordinator.go`)

### Changed

- Agent runtime defaults now leave both executor and dedicated planner tool-call
  rounds unlimited (`max_steps = 0`, `planner_max_steps = 0`). Step limits now
  come from the user/global config only; project `reasonix.toml` does not
  override them.

## [1.0.0] — 2026-06-03

First stable release — a **ground-up rewrite in Go**. Not an upgrade of the `0.x`
TypeScript line; a new codebase that becomes the default (`main-v2`).

### Highlights

- **Go kernel**: a single static binary (CGO-free), cross-compiled for
  darwin/linux/windows on amd64 + arm64. Distributed via npm (the package wraps
  the native binary), Homebrew (`esengine/reasonix` tap), and release archives;
  no Node runtime needed to run it.
- **Agent core**: the loop, built-in tools (read/write/edit/multi_edit/glob/grep/
  ls/bash/web_fetch/todo_write), permission gate, sandboxed bash, and the
  DeepSeek prefix-cache–oriented design.
- **Subagents**: `task` plus explore/research/review/security_review skill agents.
- **Skills & hooks**: Claude-Code-style skills (`internal/skill`) and hooks
  (`internal/hook`), symlink-aware and slash-integrated.
- **MCP client**: connect external servers over stdio / Streamable HTTP; reads
  `[[plugins]]` and a Claude-Code `.mcp.json`.
- **Code intelligence via CodeGraph**: a tree-sitter symbol/call graph
  (`codegraph_*` tools) replaces embedding semantic search — no embedding service
  or API cost. Fetched into a local cache on first use (or `reasonix codegraph
  install`) and indexed in the background, so installs and startup stay fast.
- **Plan mode** with evidence-backed step sign-off (`complete_step`).
- **Memory**: `REASONIX.md` hierarchy + auto-memory, folded into the cache-stable
  prefix.
- **ACP** (`reasonix acp`) and an HTTP/SSE server frontend; desktop app (Wails).

### Fixed

- **File encoding support restored** — GBK/GB18030 (and other non-UTF-8) files
  can now be read, edited, and grepped correctly. The v2 rewrite had dropped
  v1's encoding detection; files in CJK Windows charsets were silently misread
  or rejected as binary. The read/edit/write round-trip now preserves the
  original file encoding. (#2637)

### Notes

- Versions: the legacy TypeScript line stays in `0.x`; the Go line starts at
  `1.0.0`. See [docs/MIGRATING.md](docs/MIGRATING.md).
- Release archives ship a bare binary; CodeGraph is fetched on first use. Windows
  support for the fetched runtime is unverified — install `codegraph` on PATH if
  the auto-fetch doesn't resolve there.

[1.0.0]: https://github.com/esengine/DeepSeek-Reasonix/releases/tag/v1.0.0
