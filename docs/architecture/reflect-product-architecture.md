# Reflect Product Architecture

## Product statement

Reflect is the model-agnostic main AI coding-agent CLI from GrayCodeAI.

`graycode-eco` is only the local parent folder used to develop the independent
repositories. It is not a monorepo or a runtime product.

Reflect is the only primary product surface in the graycode-eco ecosystem. The support repos exist to power Reflect, not to compete with it as standalone products.

For model execution specifically: **Reflect is the face and composition layer;
Relay is the engine.** Reflect owns the conversation and product experience while
the `relay/engine` facade owns the complete provider path from credential and
catalog state through model selection and normalized generation/streaming.

## Goals

- Keep Reflect model agnostic.
- Keep Reflect CLI-first and local-first.
- Keep provider integration pluggable.
- Keep the provider engine isolated behind a stable facade.
- Make review and verification first-class.
- Keep the design ready for a future hosted layer without making cloud a dependency.

## Target repo set

- `reflect`
- `relay`
- `owl`
- `graycode-platform` (outside the Reflect runtime module graph)

Directory names are authoritative for dependencies: `relay` is Relay. The
product label remains useful in CLI and user-facing documentation.

## Runtime architecture

```text
Users / SDKs / Skills
        |
        v
      REFLECT
        |
        +----------------------+
        |                      |
        v                      v
 core execution          embedded token engine
 relay/engine            internal/token
        |                      |
        +----------+-----------+
                   |
                   v
            internal/contracts
```

## Current vs proposed

Current implementation in the workspace:

```text
reflect
  -> relay/engine
  -> internal/token (embedded)
  -> internal/contracts

x-> Reflect internals from any external module
```

Proposed steady-state architecture:

```text
SDKs / Skills / future integrations
                |
                v
              Reflect
                |
   +------------+------------+
   |                         |
   v                         v
 Relay/engine          internal/token
   |                         |
   +------------+------------+
                |
                v
        internal/contracts
```

This means:

- Reflect is the product and orchestration boundary
- the provider engine stays behind its stable facade
- shared vocabulary lives in Reflect's vendored contracts
- SDKs and community skills consume Reflect, not the engine directly
- `graycode-platform` provides the optional hosted plane through HTTP and a
  private Service Binding; it is not imported by Reflect

## Reflect responsibilities

Reflect owns:

- CLI entrypoints
- session lifecycle
- orchestration
- workflow control
- tool routing
- permission and policy enforcement
- user model preference and task-semantic model intent
- engine coordination
- public integration surfaces

Reflect does not own:

- provider-specific implementation details
- engine-specific business logic
- future company-wide auth/billing/platform concerns

## Engine responsibilities

### `relay`
- stable host control and generation facade (`relay/engine`)
- credential storage and safe credential status
- catalog discovery and model metadata
- concrete provider/deployment selection
- provider adapters
- request/response normalization
- streaming
- retries/timeouts/fallbacks
- low-level provider registry and compatibility logic behind the engine facade

### `internal/token` (embedded)
- token budgeting and estimation
- context ranking, packing, and truncation
- compression, secret detection, and usage tracking
- model-ready context assembly inputs

## Primary runtime flow

1. User invokes `reflect`.
2. Reflect loads product settings, policy, and workspace state, then creates an
   Relay Engine with effective per-instance custom gateway settings.
3. Reflect creates or resumes a session.
4. Reflect assembles context through the embedded token engine.
5. Reflect recalls relevant local memories.
6. Reflect routes provider execution through `relay`.
7. Reflect invokes tools and records graph observations.
8. Reflect persists results and returns output to the user.

At step 6, Reflect passes intent and Reflect-owned conversation DTOs through its
adapter. Relay loads provider/catalog/credential state, resolves the gateway,
and returns normalized events. No production Reflect package imports a lower
Relay package, and no Relay engine DTO is used as Reflect's persistent or CLI
schema.

## Implementation phases

### Phase 1
- freeze architecture rules
- document repo roles
- define shared contracts inventory

### Phase 2
- add shared contracts
- move shared types out of Reflect internals

Status:
- completed
- shared contracts now exist for `types`, `contracts/review`, `contracts/verify`, `tools`, `events`, and `policy`

### Phase 3
- remove engine imports of Reflect internals
- remove engine-to-engine coupling

Status:
- completed for current workspace boundaries
- local/CI guards now block support-repo imports of `reflect/internal/*`

### Phase 4
- harden orchestration boundaries in Reflect
- formalize provider, review, and verify integration points

Status:
- completed for the local runtime boundary
- Reflect owns runtime DTOs and review/verify product-boundary contracts
- Reflect's `ChatClient` anti-corruption port translates only to `relay/engine`
- all lower-level Relay production imports are forbidden by shell guards and meta-audit tests

### Phase 5
- align SDKs and skills to Reflect public interfaces only

Status:
- policy is now explicit and guarded in Reflect docs
- broader non-Go consumer enforcement remains future work

## Done criteria

The architecture is in good shape when:

- `reflect` is the only product surface
- the provider engine is consumed only through its stable facade
- shared types no longer live in Reflect internals as a cross-repo API
- provider abstraction is stable
- review and verification are part of the standard runtime flow
- deprecated compatibility surfaces have a documented removal path and active guardrails
