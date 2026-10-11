# Decoder Product Architecture

## Product statement

Decoder is the model-agnostic main AI coding-agent CLI from GrayCodeAI.

`graycode-eco` is only the local parent folder used to develop the independent
repositories. It is not a monorepo or a runtime product.

Decoder is the only primary product surface in the graycode-eco ecosystem. The support repos exist to power Decoder, not to compete with it as standalone products.

For model execution specifically: **Decoder is the face and composition layer;
Mux is the engine.** Decoder owns the conversation and product experience while
the `mux/engine` facade owns the complete provider path from credential and
catalog state through model selection and normalized generation/streaming.

## Goals

- Keep Decoder model agnostic.
- Keep Decoder CLI-first and local-first.
- Keep provider integration pluggable.
- Keep the provider engine isolated behind a stable facade.
- Make review and verification first-class.
- Keep the design ready for a future hosted layer without making cloud a dependency.

## Target repo set

- `decoder`
- `mux`
- `owl`
- `graycode-platform` (outside the Decoder runtime module graph)

Directory names are authoritative for dependencies: `mux` is Mux. The
product label remains useful in CLI and user-facing documentation.

## Runtime architecture

```text
Users / SDKs / Skills
        |
        v
      DECODER
        |
        +----------------------+
        |                      |
        v                      v
 core execution          embedded token engine
 mux/engine            internal/token
        |                      |
        +----------+-----------+
                   |
                   v
            internal/contracts
```

## Current vs proposed

Current implementation in the workspace:

```text
decoder
  -> mux/engine
  -> internal/token (embedded)
  -> internal/contracts

x-> Decoder internals from any external module
```

Proposed steady-state architecture:

```text
SDKs / Skills / future integrations
                |
                v
              Decoder
                |
   +------------+------------+
   |                         |
   v                         v
 Mux/engine          internal/token
   |                         |
   +------------+------------+
                |
                v
        internal/contracts
```

This means:

- Decoder is the product and orchestration boundary
- the provider engine stays behind its stable facade
- shared vocabulary lives in Decoder's vendored contracts
- SDKs and community skills consume Decoder, not the engine directly
- `graycode-platform` provides the optional hosted plane through HTTP and a
  private Service Binding; it is not imported by Decoder

## Decoder responsibilities

Decoder owns:

- CLI entrypoints
- session lifecycle
- orchestration
- workflow control
- tool routing
- permission and policy enforcement
- user model preference and task-semantic model intent
- engine coordination
- public integration surfaces

Decoder does not own:

- provider-specific implementation details
- engine-specific business logic
- future company-wide auth/billing/platform concerns

## Engine responsibilities

### `mux`
- stable host control and generation facade (`mux/engine`)
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

1. User invokes `decoder`.
2. Decoder loads product settings, policy, and workspace state, then creates an
   Mux Engine with effective per-instance custom gateway settings.
3. Decoder creates or resumes a session.
4. Decoder assembles context through the embedded token engine.
5. Decoder recalls relevant local memories.
6. Decoder routes provider execution through `mux`.
7. Decoder invokes tools and records graph observations.
8. Decoder persists results and returns output to the user.

At step 6, Decoder passes intent and Decoder-owned conversation DTOs through its
adapter. Mux loads provider/catalog/credential state, resolves the gateway,
and returns normalized events. No production Decoder package imports a lower
Mux package, and no Mux engine DTO is used as Decoder's persistent or CLI
schema.

## Implementation phases

### Phase 1
- freeze architecture rules
- document repo roles
- define shared contracts inventory

### Phase 2
- add shared contracts
- move shared types out of Decoder internals

Status:
- completed
- shared contracts now exist for `types`, `contracts/review`, `contracts/verify`, `tools`, `events`, and `policy`

### Phase 3
- remove engine imports of Decoder internals
- remove engine-to-engine coupling

Status:
- completed for current workspace boundaries
- local/CI guards now block support-repo imports of `decoder/internal/*`

### Phase 4
- harden orchestration boundaries in Decoder
- formalize provider, review, and verify integration points

Status:
- completed for the local runtime boundary
- Decoder owns runtime DTOs and review/verify product-boundary contracts
- Decoder's `ChatClient` anti-corruption port translates only to `mux/engine`
- all lower-level Mux production imports are forbidden by shell guards and meta-audit tests

### Phase 5
- align SDKs and skills to Decoder public interfaces only

Status:
- policy is now explicit and guarded in Decoder docs
- broader non-Go consumer enforcement remains future work

## Done criteria

The architecture is in good shape when:

- `decoder` is the only product surface
- the provider engine is consumed only through its stable facade
- shared types no longer live in Decoder internals as a cross-repo API
- provider abstraction is stable
- review and verification are part of the standard runtime flow
- deprecated compatibility surfaces have a documented removal path and active guardrails
