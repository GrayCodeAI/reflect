# GrayCode Ecosystem Architecture

> **Historical.** This document describes the pre-2026-09 multi-engine
> ecosystem. Decoder now depends only on `mux` and embeds its own token engine;
> the harrier, kestrel, merlin, shrike, and swift integrations have been
> removed. Kept for record.

This document describes how the independent repositories around the Decoder main
CLI connect. `graycode-eco` is only the local parent folder; it is not a Git
repository, product, or runtime module.

## Repository boundary

| Repository | Responsibility | Connection to Decoder |
|---|---|---|
| `decoder` | Main CLI, daemon, orchestration, policy, and public API | Main product and integration root |
| `eagle` | Neutral cross-repository contracts | Go dependency of Decoder and selected engines |
| `falcon` | Shared MCP transport and handler scaffolding | Go dependency of MCP-serving engines |
| `mux` | Provider credentials, catalog, routing, transport, streaming | Decoder consumes `mux/engine` |
| `harrier` | Harrier memory and retrieval engine | Decoder integration |
| `shrike` | Shrike token budgeting and context compression | Decoder integration |
| `swift` | Swift provenance and replay engine | Decoder consumes `swift/cli` |
| `kestrel` | Kestrel source-review engine | Decoder integration |
| `merlin` | Merlin verification/target-inspection engine | Decoder integration |
| `sparrow` | Go SDK | Decoder daemon API/OpenAPI consumer |
| `robin` | Python SDK | Decoder daemon API/OpenAPI consumer |
| `wren` | TypeScript SDK | Decoder daemon API/OpenAPI consumer |
| `starling` | Community skills and extensions | Decoder skill/plugin consumer |
| `owl` | Architecture visualization tooling | Reads the generated manifest/artifacts |
| `graycode-platform` | Web application, browser BFF, and Decoder Cloud Worker | Optional HTTP/Service Binding plane |

Product names are labels, not directory names: `harrier` is Harrier, `shrike` is
Shrike, `swift` is Swift, `kestrel` is Kestrel, and `merlin` is Merlin.

## Dependency and runtime topology

```text
sparrow / robin / wren ── HTTP/OpenAPI ──> decoder <── skill surface ── starling
                                             │
                                             ├── mux/engine
                                             ├── harrier       (Harrier)
                                             ├── shrike        (Shrike)
                                             ├── swift/cli     (Swift)
                                             ├── kestrel       (Kestrel)
                                             ├── merlin        (Merlin)
                                             └── eagle

engines ──> eagle when a shared contract is required
harrier / kestrel / merlin ──> falcon ──> mark3labs/mcp-go

decoder ── optional authenticated HTTP ──> graycode-platform/apps/worker
web ──> graycode-platform/apps/bff ── private Service Binding ──> worker
worker ──> D1 + Queue + R2

owl <── decoder/ecosystem.yaml via generated ecosystem.json and analysis artifacts
```

The Worker deployment is named `graycode-cloud`, but it is not a separate
repository. It lives in `graycode-platform/apps/worker`; browser identity and
the gateway live in `graycode-platform/apps/bff`.

## Workspace policy

The parent `go.work` connects the nine Go repositories marked `workspace: true`
in `decoder/ecosystem.yaml`: Decoder, Eagle, Falcon, and the six engines. The SDKs,
Starling, Owl, and GrayCode Platform are independent non-Go/API/tooling repos.

Local sibling resolution is a development convenience. Each repository keeps
its own Git history, version, CI, and release cadence. Standalone Decoder builds
use published module versions with `GOWORK=off`.

## Architecture invariants

1. Decoder is the only main product CLI and orchestration root.
2. Engines do not import Decoder internals or peer engines.
3. Shared neutral Go vocabulary belongs in Eagle.
4. SDKs and skills consume Decoder public surfaces, never engine internals.
5. No Go module imports `graycode-platform`.
6. Decoder may use the platform only through authenticated, optional runtime HTTP.
7. Local execution remains usable when the platform is unavailable.
8. The canonical repository list is `decoder/ecosystem.yaml`; Owl is a projection.

## Current versus proposed

The proposed repository shape is already present locally. Remaining work is
release and boundary refinement: publish the Eagle-compatible Mux revision,
remove its transitional legacy contract dependency from the standalone graph,
and decide whether Decoder's graph/projection integrations should be hidden behind
engine-owned facades.

The executable checklist is maintained in
`docs/architecture/ecosystem-implementation-plan.md`.
