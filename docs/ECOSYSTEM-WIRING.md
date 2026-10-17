# GrayCode ecosystem wiring

> **STALE (2026-09-18):** Diagrams below predate `eagle` removal (2026-09-04) and list 14 repos / 6 engines.
> NOTE (2026-09-04): `eagle` contracts vendored into `reflect/internal/contracts`.
> **INVENTORY MOVED (2026-10-02):** the canonical machine-readable inventory is
> [`GrayCodeAI/graycode-eco/ecosystem.yaml`](https://github.com/GrayCodeAI/graycode-eco/blob/main/ecosystem.yaml)
> (10 products, including `beam`). The `ecosystem.yaml` in this repo is
> deprecated and incomplete. Use that file for workspace and boundary checks;
> `scripts/ecosystem-manifest.sh` already reads it.

This document is the historical implementation contract.

## Current state before this change

```mermaid
flowchart LR
  Lists[Hard-coded repository lists] --> CI[Per-repo CI]
  Lists --> Scripts[Workspace and release scripts]
  Lists --> Owl[Owl catalog]

  Reflect[Reflect product] --> Engines[Six engine repositories]
  Engines --> Eagle[Eagle contracts]
  Reflect --> Eagle

  ReflectSpec[Reflect OpenAPI] -. manual snapshots .-> Sparrow
  ReflectSpec -. manual snapshots .-> Robin
  ReflectSpec -. no snapshot .-> Wren

  Browser[GrayCode browser] --> Mixed[One mixed Worker]
  Mixed --> Identity[(Identity tables)]
  Mixed --> Cloud[(Reflect Cloud tables)]
  Mixed -. fire-and-forget .-> Queue[Usage Queue]
```

The runtime engine integrations were mostly sound, but the surrounding wiring
could drift: repository names appeared independently in shell scripts and Owl,
SDK contracts were inconsistent, browser identity and the Reflect control plane
shared a Worker and D1 binding, and a successful usage write did not guarantee a
durable Queue-delivery intent.

## Implemented architecture

```mermaid
flowchart TB
  Manifest[GrayCodeAI/graycode-eco/ecosystem.yaml\ncanonical repo and contract inventory]
  Manifest --> Workspace[generated root go.work]
  Manifest --> Guards[boundary and release parity guards]
  Manifest --> OwlSnapshot[owl/ecosystem.json]

  subgraph Local[Local-first runtime]
    User[CLI / daemon user] --> Reflect[Reflect composition root]
    Reflect --> Relay[Relay facade]
    Reflect --> Harrier[Harrier / Harrier facade]
    Reflect --> Shrike[Shrike / Shrike facade]
    Reflect --> Swift[Swift / Swift facade]
    Reflect --> Kestrel[Kestrel / Kestrel facade]
    Reflect --> Merlin[Merlin / Merlin facade]
    Reflect --> Eagle[Eagle neutral contracts]
    Relay --> Eagle
    Harrier --> Eagle
    Shrike --> Eagle
    Swift --> Eagle
    Kestrel --> Eagle
    Merlin --> Eagle
  end

  Reflect --> Daemon[Reflect daemon API]
  Daemon --> OpenAPI[api/openapi.yaml]
  OpenAPI --> Sparrow[Sparrow Go SDK snapshot]
  OpenAPI --> Robin[Robin Python SDK snapshot]
  OpenAPI --> Wren[Wren TypeScript SDK snapshot]

  subgraph Hosted[Optional hosted plane]
    Browser[GrayCode browser] --> BFF[GrayCode identity/BFF Worker]
    BFF --> Identity[(Identity D1)]
    BFF -->|private typed Service Binding| Cloud[Reflect Cloud Worker]
    Cloud --> Control[(Control-plane D1)]
    Cloud -->|same D1 transaction| Outbox[(Usage outbox)]
    Outbox -->|immediate + scheduled retry| Queue[Cloudflare Queue]
    Queue --> Rollups[(Idempotent rollups)]
    Queue --> Archive[(R2 archive)]
  end

  Reflect -. explicit opt-in; fail-open .-> Cloud
```

## Ownership rules

| Boundary | Owner | Rule |
|---|---|---|
| Product orchestration | Reflect | Engines never orchestrate Reflect or one another. |
| Provider runtime | Relay | Reflect imports its supported `engine` facade. |
| Shared data contracts | Eagle | Neutral types only; no product behavior. |
| Portable local graph | Source engine/Reflect | Emit bounded `reflect.graph/v1` facts without raw secrets or prompts. |
| Daemon HTTP API | Reflect | `reflect/api/openapi.yaml` is authoritative. |
| SDK endpoint support | Each SDK | Exact contract snapshot plus an explicit decision for every path. |
| Browser identity | GrayCode BFF | Users, sessions, email, API keys, and UI activity remain in identity D1. |
| Hosted control plane | Reflect Cloud | Organizations, projects, devices, usage, billing, graph ledger, and audit. |
| Cloud delivery | D1 outbox + Queue | Business state and delivery intent commit together; consumers are idempotent. |
| Architecture discovery | Owl | Generated projection of the canonical manifest, never a second inventory. |

## Repository identity

Repository directories are repository and Go module identities. Product names
remain the user-facing labels. In particular, `relay` is Relay. Scripts and CI
must use `directory`/`github_repo`; UI copy may use `product_name`.

## Contract and event flow

1. Engine packages expose only the facade declared in the canonical inventory
   (`GrayCodeAI/graycode-eco/ecosystem.yaml`); the manifest validator rejects a
   missing or out-of-module facade.
2. Eagle commit `9d358dde4ad8` is the cross-repository contract revision, pinned
   through its reachable Go pseudo-version until the next semver tag is published;
   the parity gate checks every declared consumer.
3. Local graph projections use `reflect.graph/v1`. Reflect Cloud independently
   validates and privacy-normalizes them into `graycode-cloud.graph/v1`.
4. Usage ingestion commits its idempotency claim, budget counter, event,
   billing ledger row, daily activity projection, and `usage.recorded.v1`
   outbox record in one ordered D1 batch.
5. Queue publication is at-least-once. A successful send marks the outbox row;
   failures back off and the cron drain retries them. Queue consumers dedupe on
   `eventId`, so a send/mark race cannot double-count.

## Release and development workflow

- Run `make workspace` in Reflect to regenerate the parent `go.work`.
- Run `make boundaries` to validate the inventory, module isolation, Eagle
  parity, facade locations, and support-repository coupling.
- Root `go.mod` files never contain local `replace` directives. Local sibling
  substitutions exist only in the generated, uncommitted parent workspace.
- SDK CI checks its snapshot byte-for-byte against Reflect and separately checks
  that every OpenAPI path has a support decision.
- Run `owl/scripts/sync-ecosystem.sh` after changing the canonical inventory;
  Owl CI rejects drift.
- Standalone/module-mode tests remain the release truth; workspace tests are an
  additional integration pass, not a substitute.

### Coordinated Go publication gate

The compatible Eagle-migrated Relay source is currently ahead of Relay's
published `origin/main`. Do not merge or release Reflect against the older Relay
pseudo-version: it still exposes the retired `reflect-core-contracts` types and is
not type-compatible with Reflect's Eagle boundary.

1. Merge and publish the Relay ecosystem-wiring branch first.
2. Resolve that final remote commit to its canonical Go pseudo-version with
   `go list -m github.com/GrayCodeAI/relay@<commit>`.
3. Update Reflect's Relay requirement, run `GOWORK=off go mod tidy`, and remove any
   transition excludes no longer required by the published engine graphs.
4. Require Reflect's `public-modules` and `release-parity` CI jobs to pass before
   merging or tagging Reflect.

This publication is intentionally not performed by the source implementation:
remote branch merges and tags are externally visible release operations.

## Deployment sequence for the Worker split

1. Create or automatically provision the `reflect-identity` D1 database and
   confirm its binding in `apps/bff/wrangler.jsonc`.
2. Export the existing identity tables from the legacy mixed database and
   import them into the identity database; verify row counts and login flows.
3. Deploy `graycode-cloud` with the named `ReflectCloudService` entrypoint and
   apply migration `0023_usage_outbox.sql`.
4. Deploy `reflect-bff` with its D1 binding and private service binding.
5. Point the web frontend API hostname at the BFF. Reflect/CLI device start and
   poll traffic continues to target Reflect Cloud.
6. After a rollback window, remove the legacy identity tables from the old
   cloud D1 using a separately reviewed data-retirement migration.

The database move is intentionally an operator-controlled deployment step:
creating Cloudflare resources or deleting the legacy copy is not safe to infer
from source-code changes alone.
