# Reflect Dependency Rules

`graycode-eco` is only a local parent folder. `reflect` is the main CLI and the
single orchestration root. Repository names below use directory/module names;
product labels are shown in parentheses where they differ.

## Required graph

```text
sparrow / robin / wren ── Reflect daemon API ──> reflect <── skill API ── starling
                                             │
                                             ├── relay/engine
                                             ├── harrier       (Harrier)
                                             ├── shrike        (Shrike)
                                             ├── swift/cli     (Swift)
                                             ├── kestrel       (Kestrel)
                                             ├── merlin        (Merlin)
                                             └── eagle

harrier / kestrel / merlin ──> falcon ──> mark3labs/mcp-go
engines ──> eagle when a shared contract is required
```

The canonical list of repositories is in
[`reflect/ecosystem.yaml`](../../ecosystem.yaml). It is the single source of
truth; other documents must not maintain a second repository list.

## Contract edges

An engine may depend on Eagle only when it produces or consumes a cross-repo
contract such as a finding, severity, event, or graph fact. Current Go module
consumers are tracked by `ecosystem.yaml` and checked for Eagle version parity.

Falcon is a separate foundation for shared MCP transport and handler patterns.
It remains upstream-only and must not import Reflect, Eagle, or an engine.

## Forbidden graph

```text
engine -> engine
engine -> reflect/internal/*
engine -> reflect/shared/*
SDK -> engine
skills -> engine
any Go module -> graycode-platform code
```

## Rules

### 1. Reflect is the orchestrator

Only Reflect coordinates the support engines and owns the user-facing CLI,
daemon, sessions, tools, policy, and product workflows.

### 2. Engines are peers

Engines may share concepts through Eagle contracts, but may not import each
other. Product names do not change this rule: `harrier` is Harrier, `shrike` is
Shrike, `swift` is Swift, `kestrel` is Kestrel, and `merlin` is Merlin.

### 3. Provider logic stays behind Relay

Reflect production code reaches provider credentials, catalogs, routing, and
transport only through `github.com/GrayCodeAI/relay/engine`. This is enforced
by shell and AST/package-graph guards.

### 4. Reflect schemas stay Reflect-owned

Reflect conversation persistence and CLI/JSON output are explicit projections,
not aliases or direct serialization of Relay DTOs.

### 5. Graph integrations are explicit surfaces

Reflect currently imports selected engine-owned graph/projection packages for
memory, token, review, and verification capture. These are integration
exceptions, not peer-engine dependencies. If replaceability requires stronger
isolation, move those calls behind engine-owned facades; do not expose more
storage or implementation types.

### 6. Cloud is runtime-only

`graycode-platform` is outside the Reflect Go module graph. Reflect may call the
deployed `graycode-cloud` Worker over authenticated HTTP for optional usage or
explicit graph synchronization. Platform failures must not block local CLI
execution.

### 7. Engine configuration is instance-scoped

Reflect supplies effective gateway settings while constructing an Relay Engine.
Product code must not mutate Relay process-global gateway state.

## Enforcement

Reflect CI runs manifest validation, no-local-replace checks, Eagle parity,
support-repository coupling, Relay facade checks, internal-layer checks, and
the AST package-boundary audit. Support repositories run their own boundary
guards. SDK contract tests compare their OpenAPI snapshots with Reflect's public
daemon contract.
