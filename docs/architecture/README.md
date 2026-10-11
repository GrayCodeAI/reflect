# Decoder Architecture Plan

This directory holds the implementation and design docs for Decoder, the main
model-agnostic AI coding-agent CLI. `graycode-eco` is only the local parent
folder for independent repositories.

Documents:

- `decoder-current-vs-proposed.md` - current workspace shape vs target Decoder-centered repository architecture
- `graycode-ecosystem-summary.md` - one-page repo role, dependency, and future cloud summary
- `decoder-product-architecture.md` - target architecture and runtime flow
- `decoder-repo-roles.md` - role of each Decoder repo in the product ecosystem
- `decoder-dependency-rules.md` - import and ownership boundaries
- `eagle-spec.md` - shared contracts layer and current status
- `decoder-provider-abstraction.md` - provider/runtime abstraction design
- `decoder-review-verify-lifecycle.md` - review and verification lifecycle
- `decoder-architecture-v1-definition-of-done.md` - realistic shipping bar for architecture v1
- `adr/ADR-0004-file-first-session-history.md` - canonical session history and SQLite projection boundary
- `adr/ADR-0001-graycode-platform-telemetry-edge.md` - constrained optional
  runtime edge from Decoder to GrayCode Platform
- `tasks.md` - historical implementation checklist from the initial architecture pass (superseded by the definition-of-done doc; kept for record)
- `adr/` - accepted architecture decision records, e.g. exceptions to the dependency rules above
  - `ADR-0003-grok-behavioral-port-go-multirepo.md` - Year 0 Grok behavioral port keeps Go multi-repo
- Related (historical) execution track: `docs/plans/YEAR-0-ACTIVE.md`

Core rule:

`decoder` is the main CLI/product. The other repositories are capabilities,
contracts, integrations, tooling, or optional platform services that connect
to it through the boundaries documented here.

Final target shape:

- `decoder` is the orchestrator and only primary product surface
- `mux` sits below Decoder as the provider engine, consumed through its stable
  engine facade
- Decoder's token/context engine is embedded (`internal/token`), not a peer repo
- shared vocabulary lives in Decoder's vendored `internal/contracts`
- SDKs and community skills sit above Decoder as consumers of Decoder public surfaces
- `graycode-platform` stays outside the Decoder runtime module graph and exposes
  the optional web/BFF/Decoder Cloud plane
