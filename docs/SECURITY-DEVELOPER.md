# Reflect developer security model

This document describes how reflect and relay handle API keys and agent isolation for an individual developer on macOS or Linux (no Vault, no proxy). Teams and enterprise deployment models come later.

## Goals

- API keys live only in the OS secret store (macOS Keychain / Linux GNOME Keyring or KWallet).
- Reflect does not read API keys from `.env`, shell env, or plaintext files.
- Relay's `provider.json` holds routing and deployment metadata only — never secrets on disk.
- Reflect talks to relay without putting keys in JSON or chat messages.
- Agent commands run on the host under the permission engine; file tools cannot
  read credential paths.

## Credential storage

| Write | Read | Remove |
|-------|------|--------|
| `/config` paste flow → `relay/engine.Engine.SaveCredential` | `Engine.ResolveCredential` (secret store only) | `/config key remove` or `reflect credentials remove` |

Reflect does not import keys from legacy `~/.reflect/env` / `~/.reflect/.env` files or
from secret fields in `provider.json`. Save every key through `/config`.

Check status: `reflect credentials status`, `reflect path`, or `reflect preflight`.

## First-run flow (`/config`)

```
User pastes API key in /config
        |
        v
Reflect /config -> Relay engine credential service (OS secret store)
        |
        v
Relay engine discover/apply (credentials from store, not JSON body)
        |
        v
SetupUI JSON (display_name + canonical_id per model)
        |
        v
User picks model -> Relay provider.json (canonical id only)
```

Remove a stored key: `/config key remove` (interactive picker).

## Reflect to Relay

- **Control plane**: Reflect calls only `relay/engine`; no lower Relay package is a
  production import.
- **Discovery/apply**: credentials are resolved from the Engine's injected
  secret store; provider state and request bodies remain sanitized.
- **Chat**: Reflect sends model intent, messages, and tool definitions; Relay
  resolves the gateway and reads secrets internally.

## Agent execution

```
+------------------+
|  Reflect TUI/host   |
|  Keychain access |
|  /config paste   |
+------------------+
         |
         |  permission engine (tier + rules + approval gate)
         v
   Host command execution
```

Reflect executes agent commands directly on the host. Every tool call passes
through the permission engine (autonomy tier, allow/deny rules, approval gate,
and dry-run kill switch) before it runs. No Docker daemon or container runtime
is required.

### Blocked for agents

- **Read** tool: legacy Reflect env files, Relay's configured `provider.json`,
  `~/.ssh/*`, etc.
- **Bash**: `printenv`, `env`, reading reflect env paths, echoing `*_API_KEY` variables.

## Secrets left on disk

- **provider.json secrets**: `reflect path` fails its `provider.json` security
  check when the file still holds secret fields. Reflect does not remove them
  automatically: back up the file, delete the secret fields, and save the keys
  again through `/config`.
- **Provider state writes**: the Relay engine sanitizes provider state and
  writes it atomically, so secret fields are never written back to disk.

## Provider state path

Relay owns the provider-state path. Resolution order is:

1. `RELAY_CONFIG_DIR/provider.json`
2. the platform user-config directory under `relay/provider.json`

Reflect's Read/Edit/Write and Bash safety checks protect the resolved path,
including a custom or symlinked `RELAY_CONFIG_DIR`; protection is not limited
to the historical default provider-state location.

## Environment variables

Non-secret overrides only (reflect does not load provider API keys from env):

| Variable | Meaning |
|----------|---------|
| `REFLECT_CONFIG_DIR` | Override reflect config directory |
| `RELAY_CONFIG_DIR` | Override Relay provider-state directory; takes precedence for `provider.json` |
| `OPENAI_MODEL` | Override default OpenAI model |
| `OLLAMA_BASE_URL` | Ollama server URL (also saved via `/config` for Ollama) |

## Related code

- Reflect: `internal/config/engine.go`, `internal/tool/safety.go`,
  `internal/storage/paths.go`, `cmd/credentials.go`
- Relay public host boundary: `engine/`
- Daemon HTTP surface: [`docs/DAEMON-PORT-THREAT-MODEL.md`](DAEMON-PORT-THREAT-MODEL.md)
