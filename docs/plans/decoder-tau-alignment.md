# Plan: Decoder — Tau-style TUI restyle + architecture tightening

Status: in progress
Branch: `feat/rename-decoder`

## Context

Decoder (formerly Hawk) is a Go terminal coding agent (Bubble Tea + Lip Gloss,
~17.7k LOC TUI across 107 files, 138 tools, 64 commands). Tau is a small
Python/Textual teaching agent with a strict 3-layer split
(`tau_ai → tau_agent → tau_coding`).

"Make Decoder like Tau" cannot mean a language port or a Textual clone. This plan
does the achievable parts:

1. **Tau-style visual restyle** — adopt Tau's restrained, low-chrome look in
   Decoder's existing Bubble Tea TUI.
2. **Tau-style architecture discipline** — make `engine.StreamEvent` the hard
   frontend contract; document and enforce the boundary.
3. **Verify twice** — targeted tests + lint, then a second independent pass.

## Phase 0 — Rename completion (done)

- [x] Content rename hawk → decoder (0 refs remain except intentional shims)
- [x] File/dir renames (`cmd/decoder`, `internal/decodererr`, npm packages)
- [x] Module path `github.com/GrayCodeAI/decoder`
- [x] Compatibility shims: `env.GetenvDecoder`, `storage.resolveAppDir`,
      legacy `HAWK_*_DIR` overrides
- [x] Committed + pushed to `feat/rename-decoder`

## Phase 1 — Tau-style theme (visual)

Tau's palette is a restrained dark theme: near-black panel, single muted
accent, generous dim text, no heavy borders.

- [x] Add a `tau` palette to `internal/theme/theme_palettes.go`
      (panel `#0d1117`, ink `#e6edf3`, muted `#8b949e`, accent `#7aa2f7`,
      subtle line `#21262d`).
- [x] Register it in `themeRegistry`; make it selectable via `/theme tau`.
- [x] Reduce chrome: prefer single-line separators over box borders where the
      theme is `tau`.
- [x] Keep every existing palette intact (no regressions).

## Phase 2 — Tau-style layout (visual)

- [ ] Flatten the footer: one status line + one input line (Tau shows a
      minimal bottom bar, not a multi-row panel).
      NOTE: `renderStatusBar` intentionally renders two rows at width >= 100
      and `TestRenderStatusBar_TwoLineAtWidth120` asserts it. Flattening is a
      product/visual decision that needs sign-off before changing that test.
- [ ] Dim the input border; show the mode as a short prefix rather than a
      separate row.
- [ ] Tighten the welcome block: wordmark + one hint line.
- [ ] Preserve all existing information (model, ctx, cost) in the compact row.

## Phase 3 — Architecture tightening (contract)

- [x] Document the frontend contract in `docs/architecture/frontend-contract.md`:
      `engine.StreamEvent` is the only channel; TUI/daemon/ACP/print are
      consumers and must not reach into engine internals.
- [x] Add a testaudit guard asserting `cmd/` TUI files do not import
      `internal/engine/*` subpackages beyond the public facade types.

## Phase 4 — Verification (twice)

- [x] Pass 1: `gofumpt v0.10.0`, `goimports`, `go build ./...`, `go vet ./...`,
      `golangci-lint`, targeted tests for touched packages, boundary guards.
- [x] Pass 2: independent re-run of the same, plus a fresh-eyes grep for
      stale references and a manual TUI render check.

## Out of scope (explicitly)

- Rewriting Decoder in Python / cloning Textual.
- Deleting Decoder's tool/command surface (separate decision).
- Changing the daemon/ACP wire protocols.