# Decoder developer security model

This document describes how decoder and mux handle API keys and agent isolation for an individual developer on macOS or Linux (no Vault, no proxy). Teams and enterprise deployment models come later.

## Goals

- API keys live only in the OS secret store (macOS Keychain / Linux GNOME Keyring or KWallet).
- Decoder does not read API keys from `.env`, shell env, or plaintext files.
- Mux's `provider.json` holds routing and deployment metadata only — never secrets on disk.
- Decoder talks to mux without putting keys in JSON or chat messages.
- Agent commands run on the host under the permission engine; file tools cannot
  read credential paths.

## Credential storage

| Write | Read | Remove |
|-------|------|--------|
| `/config` paste flow → `mux/engine.Engine.SaveCredential` | `Engine.ResolveCredential` (secret store only) | `/config key remove` or `decoder credentials remove` |

Decoder does not import keys from legacy `~/.decoder/env` / `~/.decoder/.env` files or
from secret fields in `provider.json`. Save every key through `/config`.

Check status: `decoder credentials status`, `decoder path`, or `decoder preflight`.

## First-run flow (`/config`)

```
User pastes API key in /config
        |
        v
Decoder /config -> Mux engine credential service (OS secret store)
        |
        v
Mux engine discover/apply (credentials from store, not JSON body)
        |
        v
SetupUI JSON (display_name + canonical_id per model)
        |
        v
User picks model -> Mux provider.json (canonical id only)
```

Remove a stored key: `/config key remove` (interactive picker).

## Decoder to Mux

- **Control plane**: Decoder calls only `mux/engine`; no lower Mux package is a
  production import.
- **Discovery/apply**: credentials are resolved from the Engine's injected
  secret store; provider state and request bodies remain sanitized.
- **Chat**: Decoder sends model intent, messages, and tool definitions; Mux
  resolves the gateway and reads secrets internally.

## Agent execution

```
+------------------+
|  Decoder TUI/host   |
|  Keychain access |
|  /config paste   |
+------------------+
         |
         |  permission engine (tier + rules + approval gate)
         v
   Host command execution
```

Decoder executes agent commands directly on the host. Every tool call passes
through the permission engine (autonomy tier, allow/deny rules, approval gate,
and dry-run kill switch) before it runs. No Docker daemon or container runtime
is required.

### Blocked for agents

- **Read** tool: legacy Decoder env files, Mux's configured `provider.json`,
  `~/.ssh/*`, etc.
- **Bash**: `printenv`, `env`, reading decoder env paths, echoing `*_API_KEY` variables.

## Secrets left on disk

- **provider.json secrets**: `decoder path` fails its `provider.json` security
  check when the file still holds secret fields. Decoder does not remove them
  automatically: back up the file, delete the secret fields, and save the keys
  again through `/config`.
- **Provider state writes**: the Mux engine sanitizes provider state and
  writes it atomically, so secret fields are never written back to disk.

## Provider state path

Mux owns the provider-state path. Resolution order is:

1. `MUX_CONFIG_DIR/provider.json`
2. the platform user-config directory under `mux/provider.json`

Decoder's Read/Edit/Write and Bash safety checks protect the resolved path,
including a custom or symlinked `MUX_CONFIG_DIR`; protection is not limited
to the historical default provider-state location.

## Environment variables

Non-secret overrides only (decoder does not load provider API keys from env):

| Variable | Meaning |
|----------|---------|
| `DECODER_CONFIG_DIR` | Override decoder config directory |
| `MUX_CONFIG_DIR` | Override Mux provider-state directory; takes precedence for `provider.json` |
| `OPENAI_MODEL` | Override default OpenAI model |
| `OLLAMA_BASE_URL` | Ollama server URL (also saved via `/config` for Ollama) |

## Related code

- Decoder: `internal/config/engine.go`, `internal/tool/safety.go`,
  `internal/storage/paths.go`, `cmd/credentials.go`
- Mux public host boundary: `engine/`
- Daemon HTTP surface: [`docs/DAEMON-PORT-THREAT-MODEL.md`](DAEMON-PORT-THREAT-MODEL.md)
