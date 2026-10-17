# Dynamic Model Discovery — Reflect Developer Guide

## Ownership and flow

Reflect is the product face; Relay is the provider engine. Reflect owns the CLI/TUI,
model-picker presentation, user intent, and output compatibility. The
`relay/engine` facade alone owns provider registry details, credentials,
discovery, catalog/cache policy, model aliases, deployment routing, and chat
transport.

```text
provider APIs / remote catalog / local cache
                     |
                     v
            relay/engine.Engine
          catalog + credentials + routing
                     |
          stable host DTOs and methods
                     v
       Reflect composition and presentation
          /config | models | conversation
```

Production Reflect packages must not import Relay packages below
`github.com/GrayCodeAI/relay/engine`. Shell and AST guards enforce a
zero-exception boundary.

## Reflect composition boundary

`internal/config` is Reflect's control-plane composition root. It creates an
Relay engine and projects engine models into Reflect UI and command contracts:

```go
models, err := config.ListEngineModels(ctx, "anthropic", false)
live, err := config.ListEngineModels(ctx, "anthropic", true)
public, err := config.ListPublicEngineModels(ctx, "xiaomi_mimo_payg")
```

Conversation construction goes through Reflect's `internal/engine` adapter. The
adapter translates Reflect-owned message, tool, usage, and stream DTOs to the
Relay engine facade; conversation history, WAL, resume, approvals, and tool
execution remain Reflect-owned.

## Catalog and live discovery

The Relay engine combines its provider registry, provider-scoped live
discovery, and the cache at `~/.relay/model_catalog.json` by default. Override
the cache with `RELAY_MODEL_CATALOG_PATH`. Credential and state dependencies
are injected into each Engine instance, so discovery does not need hidden
process-global credentials.

Use the normal cache-backed path for repeatable UI and automation:

```bash
reflect models list anthropic
reflect models list anthropic --json
```

Use a provider-scoped live request when current connectivity and credentials
must be checked:

```bash
reflect models list anthropic --live
reflect models list anthropic --live --json
reflect models list anthropic --live --raw
```

`reflect preflight` reports **local readiness**: usable local state, a selected
model, and presence of the required stored credential. It is intentionally
cheap and does not prove that a remote provider accepts that credential.
Treat a successful provider-scoped `--live` request (or `/config` live
validation) as **live verified**.

## Stable command output

`reflect models list --json` is a Reflect-owned compatibility contract, not a direct
serialization of Relay's evolving `engine.Model` DTO. Its stable fields are:

```text
id, input_price_per_1m, output_price_per_1m, context_window, max_output,
server_tools, display_name, description, owner, live_metadata
```

New fields must be additive. `--raw` returns provider-native
`live_metadata` objects when available; for cache/public rows without native
metadata, it returns the stable Reflect compatibility row instead of `null`.

## Custom gateways

Reflect converts effective `custom_providers` settings into
`engine.Options.CustomGateways` at its composition root. Custom gateway
metadata is snapshotted per Engine instance. Do not register custom gateways
in Relay process-global state: tests, parallel sessions, and future multi-tenant
hosts must be isolated from one another.

## Adding or changing a provider

1. Implement registry, discovery, credentials, aliases, and transport behavior
   behind Relay's engine facade.
2. Add Relay tests for cache and live discovery, credential status, selection,
   and generation/streaming.
3. Commit and verify standalone Relay.
4. Advance Reflect's `../relay` sibling checkout to that exact commit, then update
   Reflect's module version when the Relay revision is published.
5. Verify both the workspace (`go.work`) and published-module
   (`GOWORK=off`) build modes.

Reflect changes are needed only for a new product behavior or an additive
Reflect-owned presentation field—not for provider-specific mechanics.

## Checks

```bash
reflect models refresh
reflect models status
reflect preflight
make relay-engine-guard
go test ./cmd ./internal/config ./internal/engine -count=1
```
