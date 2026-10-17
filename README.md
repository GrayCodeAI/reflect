<h1 align="center">AI Coding Agent for Your Terminal</h1>

<p align="center">
  AI coding agent for your terminal — built for <strong>developers</strong>, not teams or enterprises (yet).
</p>

<p align="center">
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.27.2+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="License"></a>
  <a href="https://github.com/GrayCodeAI/reflect/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/GrayCodeAI/reflect/ci.yml?style=flat-square&label=tests" alt="CI"></a>
  <a href="https://github.com/GrayCodeAI/reflect/releases"><img src="https://img.shields.io/github/v/release/GrayCodeAI/reflect?style=flat-square&label=release&color=green" alt="Release"></a>
  <a href="https://pkg.go.dev/github.com/GrayCodeAI/reflect"><img src="https://img.shields.io/badge/godoc-reference-00ADD8?style=flat-square&logo=go" alt="GoDoc"></a>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> ·
  <a href="#features">Features</a> ·
  <a href="#usage">Usage</a> ·
  <a href="#skills">Skills</a> ·
  <a href="#tools">Tools</a> ·
  <a href="#architecture">Architecture</a> ·
  <a href="#performance--benchmarks">Benchmarks</a> ·
  <a href="#contributing">Contributing</a>
</p>

---

## Why reflect

reflect is an AI-powered coding agent that lives in your terminal. It reads your codebase, writes and edits files, runs tests, and manages git — all through natural language. Unlike IDE-bound tools, reflect works over SSH and on any machine with a shell.

**Developer path:** one machine, keychain credentials, local memory. Run `reflect path` to check readiness.

- **Model-agnostic** — supports many first-class providers through [relay](https://github.com/GrayCodeAI/relay) (the exact count is dynamic — see `reflect --help`), including Anthropic, OpenAI, Gemini, Fireworks AI, Concentrate AI (pay-as-you-go), DeepSeek, and Ollama
- **Zero CGO** — single static binary, cross-compiled for linux/darwin/windows on amd64/arm64
- **Privacy-first** — your code never leaves your machine except to the LLM API you choose
- **Runs anywhere** — host execution on any machine with a shell, including over SSH
- **Extensible** — 40+ built-in tools, MCP server support, community skill registry

## Status

**Reflect is in active development.** Contributor source builds are the primary path today while we keep hardening the product in the open. Tagged releases and install assets may exist for validation, but they are not the recommended first path yet.

Follow [GrayCode](https://github.com/GrayCodeAI) for progress. When Reflect is ready to try, we will announce it on [graycodeai.com](https://graycodeai.com/changelog).

## Install

The next Reflect release uses the version in [`VERSION`](./VERSION).
Until a release with `reflect_*` archives appears on the
[releases page](https://github.com/GrayCodeAI/reflect/releases), build from source
(see [Quick Start](#quick-start-contributors--from-source)). Historical
`v0.1.0`, `v0.1.1` and `v0.2.0` tags ship `hawk_*` archives under the old module
path and cannot be installed as `reflect`.

After a Reflect release is published:

```bash
# Script: installs ~/.reflect/bin/reflect-<version> and links it as ~/.reflect/bin/reflect.
# Always checks the archive's SHA-256 against checksums.txt; also verifies the
# cosign signature of checksums.txt when cosign is installed (it says so when
# it is not). Set REFLECT_REQUIRE_COSIGN=1 to refuse installing without cosign.
curl -fsSL https://raw.githubusercontent.com/GrayCodeAI/reflect/main/install.sh | sh

# Pin a version or choose the install prefix (see: sh install.sh --help)
curl -fsSL https://raw.githubusercontent.com/GrayCodeAI/reflect/main/install.sh | sh -s -- --version "$(cat VERSION)" --prefix "$HOME/.reflect"

# Go toolchain (Go 1.27.2+): builds the tagged module from source
go install github.com/GrayCodeAI/reflect/cmd/reflect@v$(cat VERSION)
```

Each release carries `reflect_<version>_<os>_<arch>` archives for
linux/darwin/windows on amd64/arm64, `checksums.txt`, its cosign signature
bundle `checksums.txt.sigstore.json`, and SPDX SBOMs; the release notes show how
to verify them by hand. Homebrew and npm packages are **not available**: there
is no `graycodeai/tap` tap and nothing is published under `@graycodeai` on npm.

If `~/.reflect/bin` is not on your `PATH`, add it to your shell profile.

Then:

```bash
reflect            # interactive REPL (/config on first run: API key + model)
reflect path       # verify readiness
```

## Quick Start (contributors — from source)

This development revision uses `REFLECT_*` environment variables, `.reflect`
state paths, and the `reflect` keychain namespace. Configure credentials and
migrate any existing settings to these names before using the new binary.

```bash
git clone https://github.com/GrayCodeAI/reflect && cd reflect
make setup   # generates go.work referencing sibling support repos in the graycode-eco workspace
go build -o reflect ./cmd/reflect
./reflect

# First run — paste API key in /config (stored in macOS Keychain / Linux keyring)
# Verify readiness
./reflect path
```

**No Docker or container runtime is required.** Reflect executes agent commands
directly on the host. Run `reflect path` (or `reflect doctor`) to see an onboarding
checklist: credentials → model → catalog → ecosystem.

See [docs/SECURITY-DEVELOPER.md](docs/SECURITY-DEVELOPER.md) for the credential model.
Do not put API keys in shell env or `.env` for reflect.

Optional: install the current `main` with the Go toolchain. Historical
higher-numbered hawk tags mean `@latest` cannot be relied on for the reset
version series. Use `@main` or an explicit Reflect release tag.
```bash
go install github.com/GrayCodeAI/reflect/cmd/reflect@main
```

## Features

### Interactive Terminal UI

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for a smooth, keyboard-driven experience with vim-style keybindings.

### 40+ Built-in Tools

| Category | Tools |
|---|---|
| **Files** | `Read`, `Write`, `Edit`, `LS`, `Glob`, `Grep` |
| **Shell** | `Bash`, `PowerShell`, `CronCreate`, `CronDelete` |
| **Git** | `GitCommit`, `SmartCommit`, `EnterWorktree`, `ExitWorktree` |
| **Web** | `WebFetch`, `WebSearch`, `CodeSearch` |
| **Tasks** | `TodoWrite`, `TaskCreate`, `TaskList`, `TaskUpdate` |
| **Code** | `LSP` diagnostics, `CodeSearch`, `NotebookEdit`, `SQL` (read-only DB exploration) |
| **MCP** | `ListMcpResources`, `ReadMcpResource` |

### Portable Execution Graph

Export the latest or a selected Reflect session as validated graph nodes, edges,
and lifecycle events:

```bash
reflect graph export
reflect graph export <session-id>
reflect graph export --mission-dir /path/to/mission

# Explicitly privacy-normalize and sync the graph for a connected cloud project
reflect cloud graph sync <session-id>
reflect cloud graph sync --mission-dir /path/to/mission
```

Syncing needs a [GrayCode Cloud](#graycode-cloud-optional) connection.

The export contains metadata and hashes, not prompts, tool arguments/results,
policy reasons, verification evidence, or runtime output. Persisted chat
sessions automatically append privacy-safe permission, enabled approval-gate,
and `VerifyPlanExecution` summaries for subsequent graph exports. Mission runs
also persist a portable `mission-graph.json`; the mission form is validated and
synchronized explicitly with the `--mission-dir` variants above.

### GrayCode Cloud (optional)

GrayCode Cloud is the optional, opt-in hosted control plane for device login,
usage, delivery context and graph sync. Reflect works fully without it and sends
nothing until you connect. It is alpha and still being rolled out: if
`reflect cloud login` cannot reach `cloud.graycodeai.com`, the hosted service is
not available to you yet, and nothing else in Reflect is affected.

```bash
reflect cloud login                     # browser device login
reflect cloud status                    # endpoint, device, project, token storage
reflect cloud context                   # sync repository, branch, CI and deployment context
reflect cloud graph sync <session-id>   # upload a privacy-normalized execution graph
```

- **Endpoint.** New connections use `https://cloud.graycodeai.com` unless you
  pass `--endpoint` or set `REFLECT_CLOUD_URL` (for example a local Worker at
  `http://127.0.0.1:8787`). Only `https://` is accepted, except plain
  `http://` on `localhost`, `127.0.0.1` and `[::1]`. A saved connection keeps
  the endpoint that issued its token. `https://api.graycodeai.com` is the
  browser API and rejects device tokens.
- **What is sent.** Once connected, each `reflect exec` run reports one usage
  event (model, token counts, duration, status). This is fail-open: it never
  changes the run's result, and reflect waits at most 3 seconds for it after the
  output is written. `reflect cloud context` and `reflect cloud graph sync` are
  explicit, and any rejection is reported as an error.
- **Device token.** reflect keeps the token in the macOS Keychain or the Windows
  Credential Manager. Linux and other platforms have no credential-store
  integration yet, so the token is written to `.tokens` in the Reflect config
  directory (`~/.config/reflect/.tokens` by default, or `$REFLECT_CONFIG_DIR/.tokens`)
  as plaintext with mode `0600`. `reflect cloud login`/`connect` print a warning
  when that happens and `reflect cloud status` shows where the token lives.
- **Manual connection.**
  `reflect cloud connect --device-id <id> --project-id <id> --token-stdin < token.txt`
  saves an existing device; without `--token-stdin` it prompts for the token
  without echo. `--token <value>` still works but is
  deprecated because it exposes the token in shell history and `ps`.

### Multi-Agent Mission Mode (optional)

For larger tasks, decompose work into parallel feature branches (power-user / future team workflows):

```bash
reflect mission "Add auth, rate limiting, and logging"
```

Each sub-agent runs in its own git worktree with full autonomy.

### Community Skills

Discover and install modular instruction packages for specialized workflows:

```bash
reflect skills search api        # Search community registry
reflect skills install go-review # Install from GitHub
reflect skills audit             # Security scan installed skills
```

### Permission Center

reflect exposes two independent chat command centers — trust tier and the
spec-driven workflow gate — rather than one merged permission mode:

```text
/autonomy
/autonomy tier <scout|builder|operator|autonomous>
/autonomy dry-run <on|off>
/autonomy allow <rule>
/autonomy deny <rule>
/autonomy rules
/autonomy reset
/autonomy save [project|global]

/spec
/spec [what to build]
/spec status
/spec reset
```

The model is:

- `Tier` controls autonomy (bare `/autonomy` opens a picker for this):
  - `Always Ask` — prompts for permission on every tool call
  - `Scout`
  - `Builder`
  - `Operator`
  - `Autonomous`
- `Dry-run` is a kill switch: denies every tool call unconditionally,
  regardless of tier or spec stage.
- `Rules` control explicit allow/deny exceptions.
- `Spec` is a separate, independent workflow gate (bare `/spec` opens a
  picker): starting it walks the model through `Specify → Plan → Tasks`,
  writing real files to `.reflect/specs/<slug>/`, and blocks Write/Edit/Bash
  until you approve moving to implementation — at **any** trust tier,
  including Autonomous.

`/autonomy` and `/spec` are the main control surfaces for normal chat usage.
Older merged permission-mode chat commands have been removed in favor of
these two independent flows.

### MCP & LSP Support

Connect external tools via [Model Context Protocol](https://modelcontextprotocol.io/) and get code intelligence through Language Server Protocol. MCP also supports a WebSocket transport (opt-in) in addition to stdio/HTTP.

### Watch Mode (AI-comment loop)

`reflect --watch` watches your tree for `AI!` (do it now) and `AI?` (answer my question) comments and acts on them automatically — leave a directive in code, save, and reflect responds. Off by default; enabled via the `--watch` flag.

### CI / GitHub Action

A bundled GitHub Action (`.github/actions/reflect`) runs reflect in your pipeline: interactive mode on `@reflect` mentions in issue/PR comments, automation mode on labeled issues/PRs, and skill dispatch when a prompt begins with `/` (e.g. `/code-review`).

### Messaging Gateways (opt-in)

The daemon exposes Telegram, Discord, and Slack gateways so you can chat with reflect from your messaging app. Disabled by default; enabled per-channel via daemon config.

### AST Repo-Map & Codebase Analysis

An AST-based repository map (`internal/context/repomap`) gives the model a structural overview of your code. On first run, reflect can auto-analyze the codebase to seed context (default-off, opt-in).

### Auto-Lint / Auto-Fix Cycle

After edits, reflect can run the matching linter and iterate on fixes (bounded retries) before handing back. Opt-in; preserves existing behavior when disabled.

### Image / Multimodal Context

Feed screenshots and images into the conversation for vision-capable models (`internal/engine/vision.go`).

### Plan & Explore Sub-Agents

Read-only sub-agent modes: `plan` decomposes a task into steps, `explore` investigates the codebase with a configurable thoroughness budget (`quick` / `medium` / `very-thorough`).

### Conventional-Commit Generation

`SmartCommit` and the diff summarizer generate Conventional Commit messages from your staged changes.

### Durable Workflows & Approval Gates

LangGraph-style durable workflows with named, resumable step checkpoints and optional human-in-the-loop approval gates that persist the gate decision.

### Structured Output

Request JSON-Schema-constrained responses; results are validated against the schema and retried once on mismatch.

### YAML Agents & Tasks

Define personas and eval tasks in YAML (in addition to markdown personas), including per-agent display `color` and lifecycle `hooks`.

### IT-Managed Policy Tier

An optional, non-excludable org-policy rule tier (highest precedence) with HTML-comment stripping of rule files for IT-managed deployments.

### Adopted Capabilities (env-gated)

Features adopted from open-source agent projects. All are off by default unless explicitly enabled; see each section for details.

| Feature | Flag / Command | What it does |
|---|---|---|
| Best-of-N fan-out | `reflect exec --fanout N` | Run the same prompt in N isolated worktrees, compare, merge winner |
| Completion notifications | `REFLECT_NOTIFY_WEBHOOK_URL` / `REFLECT_NOTIFY_TELEGRAM_TOKEN` + `_CHAT_ID` | Webhook or Telegram ping when a run finishes |
| Incremental system-context | `REFLECT_INCREMENTAL_CONTEXT=1` | Reconcile dynamic sections instead of rebuilding the prompt |
| Tool-catalog shrink | `REFLECT_TOOL_SHRINK=1` | Compress the tool catalog sent on every request |
| Compaction segments | `REFLECT_COMPACTION_SEGMENT_DETAIL=verbose\|balanced\|minimal\|none` | Persist verbatim compacted turns to disk |
| Skill curator | `reflect skills curator status/run/pin/unpin/archive` + `REFLECT_SKILL_CURATOR=1` | Auto-archive cold agent-created skills (recoverable) |
| Structural code match | `CodeMatch` tool | Tree-sitter query search over Go/Python/TS/TSX |
| Composable toolsets | `reflect toolset [name]` + `Toolset` tool | Named tool groups (research, dev, ops, full_stack) |
| App verification | `AppVerify` tool | Boot-smoke check with readiness polling and evidence artifacts |
| Media generation | `GenerateMedia` tool | Image/video generation with local persistence. Backend via `tool.SetMediaEngine`; an OpenAI-compatible client ships in `relay/client` (`ImageClient`), wired by the host (boundary-guarded — reflect routes through the relay facade) |
| Voice transcription | Telegram voice notes + `stt` package | Transcribe Telegram voice/audio into the prompt. Backend via `stt.SetTranscriber`; an OpenAI-compatible client ships in `relay/client` (`AudioClient`), wired by the host |
| Bounded autonomous budgets | `internal/engine` (`AutonomousBudget`) | Track turns/tokens/time/continuations; report why a run stopped (budget vs gate-passed vs error) |
| Agent family messaging | `internal/multiagent` (`FamilyMessenger`) | Direct parent/sibling/child messages with pending caps + rate limits |
| Path reservations | `internal/multiagent` ledger | Detect overlapping-file changes between parallel branches |
| Live agent status | `GET /v1/agent/status` (daemon) | Machine-readable working/idle/stale per session |
| X/Twitter search | `SearchX` tool | Live X search by forwarding a query to an xAI endpoint with server-side search; returns a cited summary. Requires `XAI_API_KEY` (or `GROK_API_KEY`) |
| Desktop computer-use | `ComputerUse` tool | snapshot/click/type/scroll/press/screenshot via a pluggable `tool.SetComputerBackend` seam (host wires a native macOS accessibility backend) |
| Token-cheaper file views | `Read` tool `--minify` | Read-only, comment-stripped, whitespace-dense file view (Go via `go/parser`; other languages string-aware; never touches disk) — fewer tokens per read |
| Classified provider hints | `internal/errhint` | Buckets provider errors (Auth/RateLimit/Connectivity/ModelNotFound/ContextOverflow) into a one-line fixable next step. Wired into TUI error rows (`friendlyErrorMessage`) and `reflect exec` CLI errors |
| Atomic install transactions | `internal/installtxn` | Cross-process staged install/remove with rollback. Wired into skill install (atomic `SKILL.md` publish) |
| Test command discovery | `internal/testrunner` | Auto-detect test/verify commands (Go/npm/bun/pnpm/yarn/pytest/cargo) and parse runner output into structured results. Wired into `reflect verify` |
| Circuit breaker | `internal/circuitbreaker` | Closed/open/half-open retry-storm protection with cooldown. Wired into auto-compaction (cooldown + half-open auto-retry) |
| Smart turn routing | `internal/smartrouting` | Deterministic simple/strong turn classifier with fail-toward-strong safety. Wired into per-turn model selection (`settings.smart_routing`) |
| Conversation arc | `internal/conversationarc` | Durable sidecar memory of goals/decisions/milestones/phase with a byte-stable summary. Wired into sessions (loaded on open, saved on close, injected into the system prompt) |
| Relevance pruning | `internal/relevanceprune` | Token-budgeted context pruning preserving recent turns/tool calls/errors. Wired into compaction as a `relevance` strategy |
| Tool-result clearing | `internal/engine` (`ClearOldToolResults`) | Two-tier context management: at 80% of the context window, stale tool-result content is replaced with `[output cleared]` placeholders (tool_use kept intact) before compacting — a gentler tier below compaction |
| Approval pause timing | `internal/permissions` | Approval requests record decision timestamp + human deliberation duration (`DecisionAt`/`PauseDuration`) for approval-latency observability |
| Graceful exhaustion | `internal/engine` (`SynthesisForExhaustion`) | When turn/token/time limits hit, one final tools-disabled LLM call synthesizes a coherent completion (accomplished/remaining/next steps) instead of a bare stop line. Opt-in via `REFLECT_GRACEFUL_EXHAUSTION=1` |
| Deterministic replay cache | `internal/replaycache` (`REFLECT_REPLAY_CACHE_DIR`) | Disk-persisted SHA-256-keyed cache of completions; identical requests replay stored responses for reproducible regression runs |

## Usage

### Interactive Mode

```bash
reflect                              # Start REPL
reflect -r abc123                    # Resume session
reflect -c                           # Continue latest session
reflect --provider openai --model gpt-4o  # Override provider
```

### Permission Examples

```bash
# Inside the TUI
/autonomy
/autonomy tier builder
/autonomy allow Bash(git:*)
/autonomy deny Bash(rm -rf *)
/autonomy save project

/spec add dark mode support
```

### Non-Interactive Mode

```bash
reflect -p "explain this repo"                    # Print response, exit
reflect -p "fix tests" --allowed-tools "Bash(go test:*) Edit Read"
reflect exec "refactor auth module"               # Full engine, non-interactive
reflect exec --auto full "add error handling"     # Full autonomy
reflect exec --worktree "add rate limiting"       # Isolated branch
reflect exec --agent reviewer "review last commit" # Custom persona
```

### Diagnostics & ecosystem

```bash
reflect path                   # Developer path readiness (setup + security)
reflect doctor                  # Full health report (relay + token pipeline panel)
reflect ecosystem               # Ecosystem panel only
reflect preflight               # Quick ready-to-chat check
make path                    # Developer path verification
make smoke                   # Build + quick verification script
```

See [docs/SECURITY-DEVELOPER.md](docs/SECURITY-DEVELOPER.md).

See [docs/ecosystem-message-flow.md](docs/ecosystem-message-flow.md) for how relay connects during a chat session, and [docs/ECOSYSTEM-WIRING.md](docs/ECOSYSTEM-WIRING.md) for the current-to-proposed architecture and repository boundaries.

In the TUI: `/path`, `/ecosystem`, `/memory` (AGENTS.md).

### Daemon Mode

```bash
reflect daemon start              # Background HTTP server on port 4590
reflect daemon status             # Check if running
reflect daemon stop               # Graceful shutdown
```

Endpoints: `GET /v1/health`, `GET /v1/ready` (dependency-aware readiness), `POST /v1/chat` (JSON or SSE streaming)

### Mission Mode

```bash
reflect mission "Add auth, rate limiting, and logging"
reflect mission --workers 6 "Refactor into microservices"
reflect mission --dry-run "What would this decompose into?"
reflect mission --from-tasks                 # Execute validated dependency waves
```

## Providers

reflect works with any LLM provider. **Developer path:** paste keys in `/config` (stored in OS keychain) — not shell env or `.env`. Use `reflect credentials status` to verify.

| Provider | ID | Key (via `/config`) |
|---|---|---|
| Anthropic | `anthropic` | `ANTHROPIC_API_KEY` |
| OpenAI | `openai` | `OPENAI_API_KEY` |
| Google Gemini | `gemini` | `GEMINI_API_KEY` |
| OpenRouter | `openrouter` | `OPENROUTER_API_KEY` |
| Fireworks AI | `fireworks` | `FIREWORKS_API_KEY` |
| xAI (Grok) | `grok` | `XAI_API_KEY` |
| Z.AI | `z-ai` | `ZAI_API_KEY` |
| CanopyWave | `canopywave` | `CANOPYWAVE_API_KEY` |
| OpenCode Go | `opencodego` | `OPENCODEGO_API_KEY` |
| Kimi (Moonshot) | `kimi` | `MOONSHOT_API_KEY` |
| Xiaomi (MiMo) Pay-as-you-go | `xiaomi_mimo_payg` | `XIAOMI_MIMO_PAYG_API_KEY` |
| Xiaomi (MiMo) Token Plan | `xiaomi_mimo_token_plan` | `XIAOMI_MIMO_TOKEN_PLAN_API_KEY` (pick region in `/config`) |
| Ollama (local) | `ollama` | `OLLAMA_BASE_URL` (no API key) |

Provider routing, model resolution, and retries are handled by [relay](https://github.com/GrayCodeAI/relay).
For deployment-aware routing, set `"deployment_routing": true` in `.reflect/settings.json`
or export `REFLECT_DEPLOYMENT_ROUTING=true`. Reflect will route canonical model IDs through
Relay's deployment catalog, so new models can be exposed by refreshing the catalog
instead of changing Reflect. In chat, run `/refresh-model-catalog` to fetch the latest
deployment-aware catalog into `~/.relay/model_catalog.json`.

## Architecture

reflect is built in Go with a modular, layered architecture:

```
reflect/
├── bin/                    # Built binaries (reflect, reflect_bin)
├── cmd/                    # CLI entry point (Cobra + Bubble Tea TUI)
├── internal/
│   ├── engine/             # Agent loop, compaction, self-improvement
│   │   └── lifecycle/      # Self-improvement loop, limits tracking
│   ├── tool/               # 40+ built-in tools with safety layer
│   │   └── codegen_builtins.go  # Code generation templates
│   ├── config/             # Settings, budget tracking, agent personas
│   ├── session/            # Persistence (JSONL, WAL, checkpoints)
│   ├── api/                # HTTP API server
│   ├── daemon/             # Background HTTP/SSE server
│   ├── token/              # Token counting, compression, usage, secrets
│   ├── permissions/        # User approval system with auto-learning
│   ├── hooks/              # Event-driven plugin system
│   ├── mcp/                # Model Context Protocol client
│   ├── intelligence/       # Code intelligence (repomap, memory, planner)
│   ├── multiagent/         # Mission orchestration, parallel execution
│   ├── observability/      # Analytics, metrics, logging, tracing
│   ├── resilience/         # Circuit breaker, rate limiting, retries
│   ├── feature/            # eval, fingerprint, voice, IDE integration
│   ├── bridge/             # External bridges (sessioncapture)
│   ├── provider/           # Provider routing
│   └── system/             # Bus, cron, retention, shutdown
├── docs/                   # Architecture, security, integration docs
└── testdata/               # Test fixtures

Ecosystem sibling repos (independent Git repos in the `graycode-eco` parent
folder):
├── relay/              # LLM provider runtime
```

### Ecosystem

reflect is the main CLI/product and integrates these GrayCodeAI repositories in
three runtime layers plus optional tooling/platform services:

- **Primary product:** **reflect** is the only end-user product surface in this ecosystem.
- **Provider engine mounted by Reflect:** **relay** is the LLM provider runtime, consumed through its stable engine facade.
- **API consumers/extensions:** **graycode-skills** provides Reflect skills
  installed on demand (`reflect skills install`).
- **Tooling/platform:** **graycode-platform** contains the optional web/BFF/Reflect
  Cloud plane and is outside the Reflect Go runtime graph.

Local development uses:

- **`go.mod` modules:** pinned requirements for `relay`
- **Workspace + `go.work`:** sibling support repos are cloned in the `graycode-eco` workspace (as `../<repo>`); `go.work` resolves the module paths to those local checkouts
- **Module-mode builds:** standalone builds resolve the pinned `go.mod` versions from the module proxy (no workspace)

Cross-repo contracts now live in `internal/contracts` (vendored from the
removed `github.com/GrayCodeAI/eagle` module) so support repos do not depend
on Reflect internals. External consumers should vendor the needed DTOs from `internal/contracts`
until a published contracts module exists.

Current contract packages (`internal/contracts/`):

- `types` — severity, findings
- `graph` — portable graph vocabulary: nodes, edges, events, provenance
- `agent` — typed subagent spawn DTOs and hook events
- `policy` — permission and policy verdict contracts
- `contracts/review` — normalized review findings, comments, stats, results
- `contracts/verify` — normalized verification findings, stats, reports
- `events` — tool, trace, and usage events
- `harness` — harness evaluation reports and dimension scores

You may keep a **personal** parent **`go.work`** that lists alternate clones on disk for multi-repo development.

| Component | Repository | Purpose |
|---|---|---|
| **reflect** | This repo | AI coding agent |
| **relay** | [GrayCodeAI/relay](https://github.com/GrayCodeAI/relay) | LLM provider runtime |
| **graycode-skills** | [GrayCodeAI/graycode-skills](https://github.com/GrayCodeAI/graycode-skills) | Community skill registry |
| **graycode-platform** | [GrayCodeAI/graycode-platform](https://github.com/GrayCodeAI/graycode-platform) | Web, BFF, and Reflect Cloud |

The canonical inventory of sibling repositories is
[`GrayCodeAI/graycode-eco/ecosystem.yaml`](https://github.com/GrayCodeAI/graycode-eco/blob/main/ecosystem.yaml),
not the `ecosystem.yaml` in this repo, which is deprecated and incomplete. The
inventory moved because a product must not own the registry of its siblings: it
lived here, `reflect` was renamed from `hawk` without signalling the other nine
repositories, and the list silently omitted `beam`. See
[graycode-eco/adr/0001](https://github.com/GrayCodeAI/graycode-eco/blob/main/adr/README.md).

`scripts/ecosystem-manifest.sh` reads the roof's file and falls back to the
deprecated local copy only when the roof is not checked out alongside.

`relay` is the only Go module dependency in the entire ecosystem; it is consumed
through its stable engine facade.

For the consolidated repo map and the current-vs-proposed architecture diagrams, see [docs/architecture/reflect-current-vs-proposed.md](docs/architecture/reflect-current-vs-proposed.md).
For execution-graph ownership, automatic capture seams, and export/sync
commands, see
[docs/architecture/execution-graph.md](docs/architecture/execution-graph.md).

## Development

### Prerequisites

- Go 1.27.2+

### Build & Test

```bash
go build ./cmd/reflect           # Build binary
go test -race ./...           # Run all tests with race detector
make ci                       # Run full CI suite (lint, test, security)
make cover                    # Generate coverage report
```

### Performance & Benchmarks

Published, reproducible CPU benchmarks (session save/load, repo-map size/tokens)
live in [docs/BENCHMARKS.md](docs/BENCHMARKS.md), recorded with machine + commit so
numbers are comparable across runs. Reproduce with the existing `go test -bench`
targets there; nothing gates releases on them.

Headline numbers (Go 1.26.6, AMD EPYC 7543P, `go test -bench -benchmem -count=3`):

| Benchmark | Result |
|---|---|
| Session save (1000 msgs) | ~790 µs, 256 KB, 2,056 allocs |
| Session load (1000 msgs) | ~5.3 ms |
| Repo-map generate (100-file tree, 2,500 symbols) | ~2.2 ms, ~21.6k est. tokens, ~364 KB |
| Session save (100 msgs) | ~183 µs, 32 KB, 256 allocs |

### Project Structure

reflect follows Go conventions: `cmd/` for entry points, `internal/` for private code, tests alongside source files. See [docs/architecture.md](docs/architecture.md) for details.

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, commit conventions, and the PR process.

Quick start:

1. Fork and create a branch: `git checkout -b feat/short-description`
2. Make changes in small, focused commits
3. Run `make ci` locally
4. Open a pull request

Use [Conventional Commits](https://www.conventionalcommits.org/) for commit messages. Releases are cut by hand from a release PR; see [docs/RELEASING.md](docs/RELEASING.md).

## License

MIT — see [LICENSE](LICENSE) for details.

© 2026 GrayCode AI

## Project version

[`VERSION`](./VERSION) is the single version source. Edit it once, then run
`make version-sync` to regenerate package and OpenAPI metadata. CI and release
workflows run `make version-check` to reject stale metadata. Plain Go builds
also carry the canonical version when run outside the checkout.
