<h1 align="center">AI Coding Agent for Your Terminal</h1>

<p align="center">
  AI coding agent for your terminal — built for <strong>developers</strong>, not teams or enterprises (yet).
</p>

<p align="center">
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.27.2+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="License"></a>
  <a href="https://github.com/GrayCodeAI/decoder/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/GrayCodeAI/decoder/ci.yml?style=flat-square&label=tests" alt="CI"></a>
  <a href="https://github.com/GrayCodeAI/decoder/releases"><img src="https://img.shields.io/github/v/release/GrayCodeAI/decoder?style=flat-square&label=release&color=green" alt="Release"></a>
  <a href="https://pkg.go.dev/github.com/GrayCodeAI/decoder"><img src="https://img.shields.io/badge/godoc-reference-00ADD8?style=flat-square&logo=go" alt="GoDoc"></a>
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

## Why decoder

decoder is an AI-powered coding agent that lives in your terminal. It reads your codebase, writes and edits files, runs tests, and manages git — all through natural language. Unlike IDE-bound tools, decoder works over SSH and on any machine with a shell.

**Developer path:** one machine, keychain credentials, local memory. Run `decoder path` to check readiness.

- **Model-agnostic** — supports many first-class providers through [mux](https://github.com/GrayCodeAI/mux) (the exact count is dynamic — see `decoder --help`), including Anthropic, OpenAI, Gemini, Fireworks AI, Concentrate AI (pay-as-you-go), DeepSeek, and Ollama
- **Zero CGO** — single static binary, cross-compiled for linux/darwin/windows on amd64/arm64
- **Privacy-first** — your code never leaves your machine except to the LLM API you choose
- **Runs anywhere** — host execution on any machine with a shell, including over SSH
- **Extensible** — 40+ built-in tools, MCP server support, community skill registry

## Status

**Decoder is in active development.** Contributor source builds are the primary path today while we keep hardening the product in the open. Tagged releases and install assets may exist for validation, but they are not the recommended first path yet.

Follow [GrayCode](https://github.com/GrayCodeAI) for progress. When Decoder is ready to try, we will announce it on [graycodeai.com](https://graycodeai.com/changelog).

## Install

Prebuilt installs are **available from v0.3.0**, the first release published
under the `decoder` name. Until v0.3.0 appears on the
[releases page](https://github.com/GrayCodeAI/decoder/releases), build from source
(see [Quick Start](#quick-start-contributors--from-source)). The older
`v0.1.0`–`v0.2.0` tags predate the rename: they ship `hawk_*` archives under
the old module path and cannot be installed as `decoder`.

From v0.3.0:

```bash
# Script: installs ~/.decoder/bin/decoder-<version> and links it as ~/.decoder/bin/decoder.
# Always checks the archive's SHA-256 against checksums.txt; also verifies the
# cosign signature of checksums.txt when cosign is installed (it says so when
# it is not). Set DECODER_REQUIRE_COSIGN=1 to refuse installing without cosign.
curl -fsSL https://raw.githubusercontent.com/GrayCodeAI/decoder/v0.3.0/install.sh | sh

# Pin a version or choose the install prefix (see: sh install.sh --help)
curl -fsSL https://raw.githubusercontent.com/GrayCodeAI/decoder/v0.3.0/install.sh | sh -s -- --version 0.3.0 --prefix "$HOME/.decoder"

# Go toolchain (Go 1.27.2+): builds the tagged module from source
go install github.com/GrayCodeAI/decoder/cmd/decoder@latest
```

Each release carries `decoder_<version>_<os>_<arch>` archives for
linux/darwin/windows on amd64/arm64, `checksums.txt`, its cosign signature
bundle `checksums.txt.sigstore.json`, and SPDX SBOMs; the release notes show how
to verify them by hand. Homebrew and npm packages are **not available**: there
is no `graycodeai/tap` tap and nothing is published under `@graycodeai` on npm.

If `~/.decoder/bin` is not on your `PATH`, add it to your shell profile.

Then:

```bash
decoder            # interactive REPL (/config on first run: API key + model)
decoder path       # verify readiness
```

## Quick Start (contributors — from source)

This development revision uses `DECODER_*` environment variables, `.decoder`
state paths, and the `decoder` keychain namespace. Configure credentials and
migrate any existing settings to these names before using the new binary.


```bash
git clone https://github.com/GrayCodeAI/decoder && cd decoder
make setup   # generates go.work referencing sibling support repos in the graycode-eco workspace
go build -o decoder ./cmd/decoder
./decoder

# First run — paste API key in /config (stored in macOS Keychain / Linux keyring)
# Verify readiness
./decoder path
```

**No Docker or container runtime is required.** Decoder executes agent commands
directly on the host. Run `decoder path` (or `decoder doctor`) to see an onboarding
checklist: credentials → model → catalog → ecosystem.

See [docs/SECURITY-DEVELOPER.md](docs/SECURITY-DEVELOPER.md) for the credential model.
Do not put API keys in shell env or `.env` for decoder.

Optional: install the current `main` with the Go toolchain. Use `@main` until
v0.3.0 is tagged: `@latest` still resolves to the hawk-era v0.2.0, which does not
contain `cmd/decoder`.

```bash
go install github.com/GrayCodeAI/decoder/cmd/decoder@main
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

Export the latest or a selected Decoder session as validated graph nodes, edges,
and lifecycle events:

```bash
decoder graph export
decoder graph export <session-id>
decoder graph export --mission-dir /path/to/mission

# Explicitly privacy-normalize and sync the graph for a connected cloud project
decoder cloud graph sync <session-id>
decoder cloud graph sync --mission-dir /path/to/mission
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
usage, delivery context and graph sync. Decoder works fully without it and sends
nothing until you connect. It is alpha and still being rolled out: if
`decoder cloud login` cannot reach `cloud.graycodeai.com`, the hosted service is
not available to you yet, and nothing else in Decoder is affected.

```bash
decoder cloud login                     # browser device login
decoder cloud status                    # endpoint, device, project, token storage
decoder cloud context                   # sync repository, branch, CI and deployment context
decoder cloud graph sync <session-id>   # upload a privacy-normalized execution graph
```

- **Endpoint.** New connections use `https://cloud.graycodeai.com` unless you
  pass `--endpoint` or set `DECODER_CLOUD_URL` (for example a local Worker at
  `http://127.0.0.1:8787`). Only `https://` is accepted, except plain
  `http://` on `localhost`, `127.0.0.1` and `[::1]`. A saved connection keeps
  the endpoint that issued its token. `https://api.graycodeai.com` is the
  browser API and rejects device tokens.
- **What is sent.** Once connected, each `decoder exec` run reports one usage
  event (model, token counts, duration, status). This is fail-open: it never
  changes the run's result, and decoder waits at most 3 seconds for it after the
  output is written. `decoder cloud context` and `decoder cloud graph sync` are
  explicit, and any rejection is reported as an error.
- **Device token.** decoder keeps the token in the macOS Keychain or the Windows
  Credential Manager. Linux and other platforms have no credential-store
  integration yet, so the token is written to `.tokens` in the Decoder config
  directory (`~/.config/decoder/.tokens` by default, or `$DECODER_CONFIG_DIR/.tokens`)
  as plaintext with mode `0600`. `decoder cloud login`/`connect` print a warning
  when that happens and `decoder cloud status` shows where the token lives.
- **Manual connection.**
  `decoder cloud connect --device-id <id> --project-id <id> --token-stdin < token.txt`
  saves an existing device; without `--token-stdin` it prompts for the token
  without echo. `--token <value>` still works but is
  deprecated because it exposes the token in shell history and `ps`.

### Multi-Agent Mission Mode (optional)

For larger tasks, decompose work into parallel feature branches (power-user / future team workflows):

```bash
decoder mission "Add auth, rate limiting, and logging"
```

Each sub-agent runs in its own git worktree with full autonomy.

### Community Skills

Discover and install modular instruction packages for specialized workflows:

```bash
decoder skills search api        # Search community registry
decoder skills install go-review # Install from GitHub
decoder skills audit             # Security scan installed skills
```

### Permission Center

decoder exposes two independent chat command centers — trust tier and the
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
  writing real files to `.decoder/specs/<slug>/`, and blocks Write/Edit/Bash
  until you approve moving to implementation — at **any** trust tier,
  including Autonomous.

`/autonomy` and `/spec` are the main control surfaces for normal chat usage.
Older merged permission-mode chat commands have been removed in favor of
these two independent flows.

### MCP & LSP Support

Connect external tools via [Model Context Protocol](https://modelcontextprotocol.io/) and get code intelligence through Language Server Protocol. MCP also supports a WebSocket transport (opt-in) in addition to stdio/HTTP.

### Watch Mode (AI-comment loop)

`decoder --watch` watches your tree for `AI!` (do it now) and `AI?` (answer my question) comments and acts on them automatically — leave a directive in code, save, and decoder responds. Off by default; enabled via the `--watch` flag.

### CI / GitHub Action

A bundled GitHub Action (`.github/actions/decoder`) runs decoder in your pipeline: interactive mode on `@decoder` mentions in issue/PR comments, automation mode on labeled issues/PRs, and skill dispatch when a prompt begins with `/` (e.g. `/code-review`).

### Messaging Gateways (opt-in)

The daemon exposes Telegram, Discord, and Slack gateways so you can chat with decoder from your messaging app. Disabled by default; enabled per-channel via daemon config.

### AST Repo-Map & Codebase Analysis

An AST-based repository map (`internal/context/repomap`) gives the model a structural overview of your code. On first run, decoder can auto-analyze the codebase to seed context (default-off, opt-in).

### Auto-Lint / Auto-Fix Cycle

After edits, decoder can run the matching linter and iterate on fixes (bounded retries) before handing back. Opt-in; preserves existing behavior when disabled.

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
| Best-of-N fan-out | `decoder exec --fanout N` | Run the same prompt in N isolated worktrees, compare, merge winner |
| Completion notifications | `DECODER_NOTIFY_WEBHOOK_URL` / `DECODER_NOTIFY_TELEGRAM_TOKEN` + `_CHAT_ID` | Webhook or Telegram ping when a run finishes |
| Incremental system-context | `DECODER_INCREMENTAL_CONTEXT=1` | Reconcile dynamic sections instead of rebuilding the prompt |
| Tool-catalog shrink | `DECODER_TOOL_SHRINK=1` | Compress the tool catalog sent on every request |
| Compaction segments | `DECODER_COMPACTION_SEGMENT_DETAIL=verbose\|balanced\|minimal\|none` | Persist verbatim compacted turns to disk |
| Skill curator | `decoder skills curator status/run/pin/unpin/archive` + `DECODER_SKILL_CURATOR=1` | Auto-archive cold agent-created skills (recoverable) |
| Structural code match | `CodeMatch` tool | Tree-sitter query search over Go/Python/TS/TSX |
| Composable toolsets | `decoder toolset [name]` + `Toolset` tool | Named tool groups (research, dev, ops, full_stack) |
| App verification | `AppVerify` tool | Boot-smoke check with readiness polling and evidence artifacts |
| Media generation | `GenerateMedia` tool | Image/video generation with local persistence. Backend via `tool.SetMediaEngine`; an OpenAI-compatible client ships in `mux/client` (`ImageClient`), wired by the host (boundary-guarded — decoder routes through the mux facade) |
| Voice transcription | Telegram voice notes + `stt` package | Transcribe Telegram voice/audio into the prompt. Backend via `stt.SetTranscriber`; an OpenAI-compatible client ships in `mux/client` (`AudioClient`), wired by the host |
| Bounded autonomous budgets | `internal/engine` (`AutonomousBudget`) | Track turns/tokens/time/continuations; report why a run stopped (budget vs gate-passed vs error) |
| Agent family messaging | `internal/multiagent` (`FamilyMessenger`) | Direct parent/sibling/child messages with pending caps + rate limits |
| Path reservations | `internal/multiagent` ledger | Detect overlapping-file changes between parallel branches |
| Live agent status | `GET /v1/agent/status` (daemon) | Machine-readable working/idle/stale per session |
| X/Twitter search | `SearchX` tool | Live X search by forwarding a query to an xAI endpoint with server-side search; returns a cited summary. Requires `XAI_API_KEY` (or `GROK_API_KEY`) |
| Desktop computer-use | `ComputerUse` tool | snapshot/click/type/scroll/press/screenshot via a pluggable `tool.SetComputerBackend` seam (host wires a native macOS accessibility backend) |
| Token-cheaper file views | `Read` tool `--minify` | Read-only, comment-stripped, whitespace-dense file view (Go via `go/parser`; other languages string-aware; never touches disk) — fewer tokens per read |
| Classified provider hints | `internal/errhint` | Buckets provider errors (Auth/RateLimit/Connectivity/ModelNotFound/ContextOverflow) into a one-line fixable next step. Wired into TUI error rows (`friendlyErrorMessage`) and `decoder exec` CLI errors |
| Atomic install transactions | `internal/installtxn` | Cross-process staged install/remove with rollback. Wired into skill install (atomic `SKILL.md` publish) |
| Test command discovery | `internal/testrunner` | Auto-detect test/verify commands (Go/npm/bun/pnpm/yarn/pytest/cargo) and parse runner output into structured results. Wired into `decoder verify` |
| Circuit breaker | `internal/circuitbreaker` | Closed/open/half-open retry-storm protection with cooldown. Wired into auto-compaction (cooldown + half-open auto-retry) |
| Smart turn routing | `internal/smartrouting` | Deterministic simple/strong turn classifier with fail-toward-strong safety. Wired into per-turn model selection (`settings.smart_routing`) |
| Conversation arc | `internal/conversationarc` | Durable sidecar memory of goals/decisions/milestones/phase with a byte-stable summary. Wired into sessions (loaded on open, saved on close, injected into the system prompt) |
| Relevance pruning | `internal/relevanceprune` | Token-budgeted context pruning preserving recent turns/tool calls/errors. Wired into compaction as a `relevance` strategy |
| Tool-result clearing | `internal/engine` (`ClearOldToolResults`) | Two-tier context management: at 80% of the context window, stale tool-result content is replaced with `[output cleared]` placeholders (tool_use kept intact) before compacting — a gentler tier below compaction |
| Approval pause timing | `internal/permissions` | Approval requests record decision timestamp + human deliberation duration (`DecisionAt`/`PauseDuration`) for approval-latency observability |
| Graceful exhaustion | `internal/engine` (`SynthesisForExhaustion`) | When turn/token/time limits hit, one final tools-disabled LLM call synthesizes a coherent completion (accomplished/remaining/next steps) instead of a bare stop line. Opt-in via `DECODER_GRACEFUL_EXHAUSTION=1` |
| Deterministic replay cache | `internal/replaycache` (`DECODER_REPLAY_CACHE_DIR`) | Disk-persisted SHA-256-keyed cache of completions; identical requests replay stored responses for reproducible regression runs |

## Usage

### Interactive Mode

```bash
decoder                              # Start REPL
decoder -r abc123                    # Resume session
decoder -c                           # Continue latest session
decoder --provider openai --model gpt-4o  # Override provider
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
decoder -p "explain this repo"                    # Print response, exit
decoder -p "fix tests" --allowed-tools "Bash(go test:*) Edit Read"
decoder exec "refactor auth module"               # Full engine, non-interactive
decoder exec --auto full "add error handling"     # Full autonomy
decoder exec --worktree "add rate limiting"       # Isolated branch
decoder exec --agent reviewer "review last commit" # Custom persona
```

### Diagnostics & ecosystem

```bash
decoder path                   # Developer path readiness (setup + security)
decoder doctor                  # Full health report (mux + token pipeline panel)
decoder ecosystem               # Ecosystem panel only
decoder preflight               # Quick ready-to-chat check
make path                    # Developer path verification
make smoke                   # Build + quick verification script
```

See [docs/SECURITY-DEVELOPER.md](docs/SECURITY-DEVELOPER.md).

See [docs/ecosystem-message-flow.md](docs/ecosystem-message-flow.md) for how mux connects during a chat session, and [docs/ECOSYSTEM-WIRING.md](docs/ECOSYSTEM-WIRING.md) for the current-to-proposed architecture and repository boundaries.

In the TUI: `/path`, `/ecosystem`, `/memory` (AGENTS.md).

### Daemon Mode

```bash
decoder daemon start              # Background HTTP server on port 4590
decoder daemon status             # Check if running
decoder daemon stop               # Graceful shutdown
```

Endpoints: `GET /v1/health`, `GET /v1/ready` (dependency-aware readiness), `POST /v1/chat` (JSON or SSE streaming)

### Mission Mode

```bash
decoder mission "Add auth, rate limiting, and logging"
decoder mission --workers 6 "Refactor into microservices"
decoder mission --dry-run "What would this decompose into?"
decoder mission --from-tasks                 # Execute validated dependency waves
```

## Providers

decoder works with any LLM provider. **Developer path:** paste keys in `/config` (stored in OS keychain) — not shell env or `.env`. Use `decoder credentials status` to verify.

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

Provider routing, model resolution, and retries are handled by [mux](https://github.com/GrayCodeAI/mux).
For deployment-aware routing, set `"deployment_routing": true` in `.decoder/settings.json`
or export `DECODER_DEPLOYMENT_ROUTING=true`. Decoder will route canonical model IDs through
Mux's deployment catalog, so new models can be exposed by refreshing the catalog
instead of changing Decoder. In chat, run `/refresh-model-catalog` to fetch the latest
deployment-aware catalog into `~/.mux/model_catalog.json`.

## Architecture

decoder is built in Go with a modular, layered architecture:

```
decoder/
├── bin/                    # Built binaries (decoder, decoder_bin)
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
├── mux/              # LLM provider runtime
```

### Ecosystem

decoder is the main CLI/product and integrates these GrayCodeAI repositories in
three runtime layers plus optional tooling/platform services:

- **Primary product:** **decoder** is the only end-user product surface in this ecosystem.
- **Provider engine mounted by Decoder:** **mux** is the LLM provider runtime, consumed through its stable engine facade.
- **API consumers/extensions:** **graycode-skills** provides Decoder skills
  installed on demand (`decoder skills install`).
- **Tooling/platform:** **graycode-platform** contains the optional web/BFF/Decoder
  Cloud plane and is outside the Decoder Go runtime graph.

Local development uses:

- **`go.mod` modules:** pinned requirements for `mux`
- **Workspace + `go.work`:** sibling support repos are cloned in the `graycode-eco` workspace (as `../<repo>`); `go.work` resolves the module paths to those local checkouts
- **Module-mode builds:** standalone builds resolve the pinned `go.mod` versions from the module proxy (no workspace)

Cross-repo contracts now live in `internal/contracts` (vendored from the
removed `github.com/GrayCodeAI/eagle` module) so support repos do not depend
on Decoder internals. External consumers should vendor the needed DTOs from `internal/contracts`
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
| **decoder** | This repo | AI coding agent |
| **mux** | [GrayCodeAI/mux](https://github.com/GrayCodeAI/mux) | LLM provider runtime |
| **graycode-skills** | [GrayCodeAI/graycode-skills](https://github.com/GrayCodeAI/graycode-skills) | Community skill registry |
| **graycode-platform** | [GrayCodeAI/graycode-platform](https://github.com/GrayCodeAI/graycode-platform) | Web, BFF, and Decoder Cloud |

The canonical inventory of sibling repositories is
[`GrayCodeAI/graycode-eco/ecosystem.yaml`](https://github.com/GrayCodeAI/graycode-eco/blob/main/ecosystem.yaml),
not the `ecosystem.yaml` in this repo, which is deprecated and incomplete. The
inventory moved because a product must not own the registry of its siblings: it
lived here, `decoder` was renamed from `hawk` without signalling the other nine
repositories, and the list silently omitted `beam`. See
[graycode-eco/adr/0001](https://github.com/GrayCodeAI/graycode-eco/blob/main/adr/README.md).

`scripts/ecosystem-manifest.sh` reads the roof's file and falls back to the
deprecated local copy only when the roof is not checked out alongside.

`mux` is the only Go module dependency in the entire ecosystem; it is consumed
through its stable engine facade.

For the consolidated repo map and the current-vs-proposed architecture diagrams, see [docs/architecture/decoder-current-vs-proposed.md](docs/architecture/decoder-current-vs-proposed.md).
For execution-graph ownership, automatic capture seams, and export/sync
commands, see
[docs/architecture/execution-graph.md](docs/architecture/execution-graph.md).

## Development

### Prerequisites

- Go 1.27.2+

### Build & Test

```bash
go build ./cmd/decoder           # Build binary
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

decoder follows Go conventions: `cmd/` for entry points, `internal/` for private code, tests alongside source files. See [docs/architecture.md](docs/architecture.md) for details.

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
