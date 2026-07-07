<p align="center">
  <img src="docs/logo.svg" alt="Reasonix" width="640"/>
</p>

<p align="center">
  <strong>English</strong>
  &nbsp;·&nbsp;
  <a href="./README.zh-CN.md">简体中文</a>
  &nbsp;·&nbsp;
  <a href="./docs/GUIDE.md">Guide</a>
  &nbsp;·&nbsp;
  <a href="./docs/SPEC.md">Spec</a>
  &nbsp;·&nbsp;
  <a href="https://esengine.github.io/DeepSeek-Reasonix/">Website</a>
  &nbsp;·&nbsp;
  <strong><a href="https://discord.gg/XF78rEME2D">Discord</a></strong>
</p>

> [!IMPORTANT]
> **Reasonix 1.0 is a ground-up rewrite in Go** — this branch (`main-v2`) is the new default and where development happens now.
> The earlier `0.x` TypeScript releases are **legacy**, living on the [`v1`](https://github.com/esengine/DeepSeek-Reasonix/tree/v1) branch (maintenance only).
> See the **[migration guide](./docs/MIGRATING.md)**. `npm i -g reasonix` stays the install command — `1.0.0`+ delivers the Go binary, `0.x` is the legacy TS build.

<p align="center">
  <a href="https://www.npmjs.com/package/reasonix"><img src="https://img.shields.io/npm/v/reasonix.svg?style=flat-square&color=cb3837&labelColor=161b22&logo=npm&logoColor=white" alt="npm version"/></a>
  <a href="https://github.com/esengine/DeepSeek-Reasonix/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/esengine/DeepSeek-Reasonix/ci.yml?style=flat-square&label=ci&labelColor=161b22&logo=githubactions&logoColor=white" alt="CI"/></a>
  <a href="./LICENSE"><img src="https://img.shields.io/npm/l/reasonix.svg?style=flat-square&color=8b949e&labelColor=161b22" alt="license"/></a>
  <a href="https://www.npmjs.com/package/reasonix"><img src="https://img.shields.io/npm/dm/reasonix.svg?style=flat-square&color=3fb950&labelColor=161b22&label=downloads" alt="downloads"/></a>
  <a href="https://github.com/esengine/DeepSeek-Reasonix/stargazers"><img src="https://img.shields.io/github/stars/esengine/DeepSeek-Reasonix.svg?style=flat-square&color=dbab09&labelColor=161b22&logo=github&logoColor=white" alt="GitHub stars"/></a>
  <a href="https://atomgit.com/esengine/DeepSeek-Reasonix"><img src="https://atomgit.com/esengine/DeepSeek-Reasonix/star/badge.svg" alt="AtomGit stars"/></a>
  <a href="https://github.com/esengine/DeepSeek-Reasonix/graphs/contributors"><img src="https://img.shields.io/github/contributors/esengine/DeepSeek-Reasonix.svg?style=flat-square&color=bc8cff&labelColor=161b22&logo=github&logoColor=white" alt="contributors"/></a>
  <a href="https://github.com/esengine/DeepSeek-Reasonix/discussions"><img src="https://img.shields.io/github/discussions/esengine/DeepSeek-Reasonix.svg?style=flat-square&color=58a6ff&labelColor=161b22&logo=github&logoColor=white" alt="Discussions"/></a>
  <a href="https://discord.gg/XF78rEME2D"><img src="https://img.shields.io/badge/discord-join-5865F2.svg?style=flat-square&labelColor=161b22&logo=discord&logoColor=white" alt="Discord"/></a>
</p>

<br/>

<h3 align="center">A DeepSeek-native AI coding agent for your terminal, powered by a two-model Planner-Executor architecture.</h3>
<p align="center">Built around planner-executor collaboration, prefix-cache efficiency, and config-driven extensibility — delivered as a single static Go binary.</p>

<br/>

> [!IMPORTANT]
> **Community · 加入社区** — bilingual Discord for setup help (`#help` / `#求助`), workflow showcases, and feature ideas. → **<https://discord.gg/XF78rEME2D>**

<br/>

## Features

### Planner-Executor architecture

- **Two-model collaboration.** The planner — a lightweight, read-only model —
  decomposes every request before acting: (1) identify the Planner part
  (research), (2) identify the Executor part (code / commands), (3) execute
  only the Planner part with read-only tools, (4) produce a structured,
  executor-ready plan. The executor then carries it out with full tool access.
  Each model runs in its own cache-stable session, so neither's prefix cache
  is disturbed by the other's turns.
- **Executor summary feedback to Planner**: After each executor run, the
  Coordinator caches the executor's final assistant message and injects it as
  `[Previous execution summary]` into the planner's input on the next turn. This
  keeps the planner aware of what was actually done. The cache is cleared on
  `ResetPlannerSession` to avoid cross-session leakage.
  (`internal/agent/coordinator.go`)
- **Plan mode.** When enabled, the Coordinator stops after the planner produces
  its plan instead of unconditionally invoking the executor. This avoids wasted
  tokens from the executor re-doing read-only research the planner already
  performed.
- **Planner MCP tool opt-in.** MCP servers are connected, but only tools listed
  in the `planner_allowed_tools` config are visible to the planner. All other
  MCP tools are blocked. Built-in read-only tools (grep, read_file, glob, etc.)
  are always available.

### Foundation

- **Config-driven.** Providers, the agent, enabled tools, and plugins are all
  declared in `reasonix.toml`. Any OpenAI-compatible endpoint is a config entry,
  not new code.
- **Plugin-driven.** External tools run as subprocesses over stdio JSON-RPC
  (MCP-compatible). Built-in tools self-register at compile time.
- **Cache-aware.** Startup injects a small stable environment summary, stale
  tool output is pruned before compaction, and the built-in tool schema
  contract is documented for regression review.
- **Zero-friction distribution.** `CGO_ENABLED=0` single binary; cross-compile
  to six targets with one command. The only dependency is a TOML parser.

## Install

```sh
npm i -g reasonix                  # any OS; pulls the prebuilt native binary
brew install esengine/reasonix/reasonix   # macOS
```

Prebuilt archives (`darwin|linux|windows × amd64|arm64`) and `SHA256SUMS` are on
every [GitHub release](https://github.com/esengine/DeepSeek-Reasonix/releases).

### Code signing

Windows builds are code-signed with a free certificate provided by the
[SignPath Foundation](https://signpath.org/), with signing through
[SignPath.io](https://signpath.io/).

### Build from source

```sh
make build      # -> bin/reasonix(.exe)
make cross      # -> dist/ (darwin|linux|windows × amd64|arm64)
```

## Quick start

```sh
reasonix setup                      # config wizard → ./reasonix.toml
export DEEPSEEK_API_KEY=sk-...      # or let setup save it to Reasonix home .env
reasonix                            # then run /init to generate AGENTS.md (project memory)
reasonix run "implement the TODOs in main.go"
reasonix run --model deepseek-pro "add unit tests for this function"
echo "explain this code" | reasonix run
```

## Configuration

A minimal `reasonix.toml` — one provider and a default model — is enough to start:

```toml
default_model = "deepseek-flash"

[[providers]]
name        = "deepseek-flash"
kind        = "openai"
base_url    = "https://api.deepseek.com"
model       = "deepseek-v4-flash"
api_key_env = "DEEPSEEK_API_KEY"
```

Resolution order is **flag > `./reasonix.toml` > the user config file >
built-in defaults**; starting with **Reasonix v1.8.1**, the user file lives at
`~/.reasonix/config.toml` on macOS/Linux and
`%AppData%\reasonix\config.toml` on Windows. See
**[Configuration paths](./docs/CONFIG_PATHS.md)** for migration details and the
full `config.toml` / `.env` structure. Provider entries name secrets with
`api_key_env`; the secret values themselves live in Reasonix's global
`<Reasonix home>/.env`, shared by CLI and desktop. Project `.env` files are not
provider-key runtime fallbacks, but still feed workspace-scoped, non-provider
`${VAR}` expansion for MCP/plugin settings without importing Reasonix control
variables. Permissions, the sandbox, plugins (MCP), slash
commands, `@` references, and two-model setup are all in the
**[Guide](./docs/GUIDE.md)**.

## Documentation

- **[Guide](./docs/GUIDE.md)** — configuration, permissions & sandbox, plugins
  (MCP), slash commands, `@` references, two-model collaboration.
- **[Bot guide](./docs/BOT_GUIDE.md)** — connect Feishu, Lark, and WeChat bots
  from the desktop app, then use approvals, YOLO, and commands from IM.
- **[Spec](./docs/SPEC.md)** — engineering contract: architecture, registries,
  data types, and roadmap.
- **[Task contracts & pause policy](./docs/TASK_CONTRACT.md)** — structure
  complex requests with context, output boundaries, constraints, and when to ask.
- **[Tool contract](./docs/TOOL_CONTRACT.md)** — provider-visible built-in tool
  names, read-only flags, and schema snapshot guard.
- **[Migrating from 0.x](./docs/MIGRATING.md)** — moving from the legacy
  TypeScript releases to the 1.0 Go rewrite.
- **[Checkpoints & rewind](./docs/CHECKPOINTS.md)** — the snapshot-based edit
  safety net (Esc-Esc / `/rewind`).

<br/>
