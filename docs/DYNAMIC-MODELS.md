# Dynamic Model Discovery — Decoder Developer Guide

## Ownership and flow

Decoder is the product face; Mux is the provider engine. Decoder owns the CLI/TUI,
model-picker presentation, user intent, and output compatibility. The
`mux/engine` facade alone owns provider registry details, credentials,
discovery, catalog/cache policy, model aliases, deployment routing, and chat
transport.

```text
provider APIs / remote catalog / local cache
                     |
                     v
            mux/engine.Engine
          catalog + credentials + routing
                     |
          stable host DTOs and methods
                     v
       Decoder composition and presentation
          /config | models | conversation
```

Production Decoder packages must not import Mux packages below
`github.com/GrayCodeAI/mux/engine`. Shell and AST guards enforce a
zero-exception boundary.

## Decoder composition boundary

`internal/config` is Decoder's control-plane composition root. It creates an
Mux engine and projects engine models into Decoder UI and command contracts:

```go
models, err := config.ListEngineModels(ctx, "anthropic", false)
live, err := config.ListEngineModels(ctx, "anthropic", true)
public, err := config.ListPublicEngineModels(ctx, "xiaomi_mimo_payg")
```

Conversation construction goes through Decoder's `internal/engine` adapter. The
adapter translates Decoder-owned message, tool, usage, and stream DTOs to the
Mux engine facade; conversation history, WAL, resume, approvals, and tool
execution remain Decoder-owned.

## Catalog and live discovery

The Mux engine combines its provider registry, provider-scoped live
discovery, and the cache at `~/.mux/model_catalog.json` by default. Override
the cache with `MUX_MODEL_CATALOG_PATH`. Credential and state dependencies
are injected into each Engine instance, so discovery does not need hidden
process-global credentials.

Use the normal cache-backed path for repeatable UI and automation:

```bash
decoder models list anthropic
decoder models list anthropic --json
```

Use a provider-scoped live request when current connectivity and credentials
must be checked:

```bash
decoder models list anthropic --live
decoder models list anthropic --live --json
decoder models list anthropic --live --raw
```

`decoder preflight` reports **local readiness**: usable local state, a selected
model, and presence of the required stored credential. It is intentionally
cheap and does not prove that a remote provider accepts that credential.
Treat a successful provider-scoped `--live` request (or `/config` live
validation) as **live verified**.

## Stable command output

`decoder models list --json` is a Decoder-owned compatibility contract, not a direct
serialization of Mux's evolving `engine.Model` DTO. Its stable fields are:

```text
id, input_price_per_1m, output_price_per_1m, context_window, max_output,
server_tools, display_name, description, owner, live_metadata
```

New fields must be additive. `--raw` returns provider-native
`live_metadata` objects when available; for cache/public rows without native
metadata, it returns the stable Decoder compatibility row instead of `null`.

## Custom gateways

Decoder converts effective `custom_providers` settings into
`engine.Options.CustomGateways` at its composition root. Custom gateway
metadata is snapshotted per Engine instance. Do not register custom gateways
in Mux process-global state: tests, parallel sessions, and future multi-tenant
hosts must be isolated from one another.

## Adding or changing a provider

1. Implement registry, discovery, credentials, aliases, and transport behavior
   behind Mux's engine facade.
2. Add Mux tests for cache and live discovery, credential status, selection,
   and generation/streaming.
3. Commit and verify standalone Mux.
4. Advance Decoder's `../mux` sibling checkout to that exact commit, then update
   Decoder's module version when the Mux revision is published.
5. Verify both the workspace (`go.work`) and published-module
   (`GOWORK=off`) build modes.

Decoder changes are needed only for a new product behavior or an additive
Decoder-owned presentation field—not for provider-specific mechanics.

## Checks

```bash
decoder models refresh
decoder models status
decoder preflight
make mux-engine-guard
go test ./cmd ./internal/config ./internal/engine -count=1
```
