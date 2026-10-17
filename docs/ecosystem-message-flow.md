# Message flow (relay + embedded token engine)

How one user message travels through reflect.

## Overview

```
User prompt (TUI or reflect exec)
        │
        ▼
┌───────────────────┐
│  buildSystemPrompt │  AGENTS.md + prompt templates (practices.md)
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐     ┌─────────────┐
│  local memory      │◄────│ ~/.reflect/    │  conventions, decisions, lessons
│  recall            │     │ memories    │
└─────────┬─────────┘     └─────────────┘
          │
          ▼
┌───────────────────┐     ┌─────────────┐
│  token budget      │     │ embedded    │  CountTokens, CompressForContext
│  (context sizing)  │     │ library     │
└─────────┬─────────┘     └─────────────┘
          │
          ▼
┌───────────────────┐     ┌─────────────┐
│  Reflect ChatClient   │────►│ relay/engine│  catalog, credentials, routing
│  port + adapter    │     │ generate/   │────► provider API
└─────────┬─────────┘     │ stream      │
                          └─────────────┘
          │
          ▼
    Tool calls (Read, Edit, Bash, …)
          │
          ▼
    Response to user
          │
          ▼ (when context grows)
┌───────────────────┐
│  token Compress    │  fast path before LLM summarization
│  + relay compact   │
└───────────────────┘
```

## Step by step

### 1. Session start (`reflect` or `reflect exec`)

- **relay**: The Reflect composition root creates an `relay/engine.Engine` with
  Relay-owned state paths, an injected secret store, and per-engine custom
  gateway metadata. The engine loads provider state and the model catalog, then
  builds transport behind Reflect's `ChatClient` port.
- **memory**: `configureSession` initializes the local memory manager
  (core, auto, evolving, zen, retrieval metrics, continuity). If the state
  directory is unavailable, reflect runs without persistent memory.
- **token**: No startup step — embedded at compile time.

### 2. System prompt assembly

- Reflect templates (`internal/prompts/templates/*.md`) define behavior, tools, and practices.
- Project `AGENTS.md` is appended via `reflectconfig.BuildContextWithDirs`.
- **memory**: `Memory.Recall` injects relevant stored memories into the system prompt.

### 3. User message → agent loop (`internal/engine/stream.go`)

Each turn:

1. **memory** — recall memories matching the latest user message (token budget ~2000).
2. **relay** — Reflect's adapter calls engine generate/stream with Reflect-owned tool
   definitions; Relay normalizes provider events and tool requests.
3. Tools run with the session's memory service in context.
4. **memory** — post-session bookkeeping records the session goal and outcome.

### 4. Context pressure

When messages exceed limits (`internal/engine/compact.go`):

1. **token** — `token.Compress()` tries a fast compression path for summaries.
2. **relay** — if token reduction is insufficient, reflect calls the LLM to summarize, then keeps recent messages.

### 5. Token accounting

- `internal/engine/token/tokenizer.go` wraps the embedded token engine for
  precise and fast estimates used in budget UI and compaction decisions.

## Verify locally

```bash
reflect doctor              # ecosystem panel + relay preflight
./scripts/smoke-reflect.sh  # build + quick tests
```

## Module layout

| Module | Role in reflect | Required? |
|--------|----------------|-----------|
| **relay** | LLM APIs, catalog, credentials, routing | Yes |
| **internal/token** | Token estimate + context compression + secrets + usage | Yes (embedded, no config) |

`relay` is an independent sibling checkout; the parent `go.work` wires the
local Go workspace.

Production Reflect code imports Relay only through `relay/engine`. Conversation
history, WAL/resume, permissions, tool execution, and memory remain in Reflect;
provider credentials, discovery, selection, transport, resilience, and
normalized streaming remain in Relay.