# Reflect Repo Roles

> **Historical.** This document describes the pre-2026-09 multi-engine
> ecosystem. Reflect now depends only on `relay` and embeds its own token engine;
> the harrier, kestrel, merlin, shrike, and swift integrations have been
> removed. Kept for record.

## Product repo

### `reflect`
Main CLI and product repository. It is the orchestration root; `graycode-eco`
is only the local parent folder for the independent repositories.

Owns:

- CLI
- daemon/API surface
- agent loop
- session orchestration
- tool execution flow
- policy and permissions
- engine coordination
- user-facing model/configuration presentation and stable CLI schemas

Reflect is the only primary product surface. Users interact with Reflect, not with six
separate end-user products.

## Support engines

### `relay`
Reflect provider engine. Its public host boundary is `relay/engine`, which owns
credentials, provider state, catalog discovery, model/deployment selection,
transport, resilience, and normalized generation/streaming. Reflect production
code has zero imports of Relay's lower-level packages.

### `harrier` (Harrier)
Reflect memory engine.

### `shrike` (Shrike)
Reflect context and token engine.

### `swift` (Swift)
Reflect audit and replay engine.

### `kestrel` (Kestrel)
Reflect review engine.

### `merlin` (Merlin)
Reflect verification engine.

All six support engines are peers:

- `relay`
- `harrier` (Harrier)
- `shrike` (Shrike)
- `swift` (Swift)
- `kestrel` (Kestrel)
- `merlin` (Merlin)

They should stay isolated from each other and are coordinated by Reflect;
they may depend on `eagle` where a shared vocabulary is required.

## Ecosystem repos

### `sparrow`
Go integration surface for Reflect public APIs/contracts.

### `robin`
Python integration surface for Reflect public APIs/contracts.

### `wren`
TypeScript integration surface for Reflect public APIs/contracts.

### `starling`
Reusable Reflect skills, recipes, and extension packs.

## Shared foundation

### `eagle`
Shared types, events, findings, policies, and engine request/response contracts.

This repo should stay small, stable, and implementation-free.

### `falcon`
Shared MCP server scaffolding wrapping `mark3labs/mcp-go` — construction,
transports, and handler helpers that MCP-serving engines (`kestrel`, `merlin`)
would otherwise duplicate.

Like `eagle`, it sits below the engines: it must not import
engines, reflect, or graycode-platform.

## Tooling and platform

### `owl`
Read-only architecture visualization tooling. It consumes the generated
projection of `reflect/ecosystem.yaml`; it is not a runtime dependency.

### `graycode-platform`
Separate web, browser-BFF, and Reflect Cloud repository. Its Worker deployment is
named `graycode-cloud`, but it connects to Reflect only through authenticated
runtime HTTP/Service Binding and is never a Go dependency.

## Role rules

- Users should feel they are using `reflect`, not six unrelated tools.
- Engines are internal capabilities from a product perspective.
- Engines can stay in separate repos for isolation, testing, and replacement.
- Engines must not import each other.
- SDKs and skills extend Reflect, but should not bypass Reflect to reach engines directly.
