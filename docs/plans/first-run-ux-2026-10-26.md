# First successful task: Reflect and Relay

Status: in progress, based on the 2026-10-26 Linux first-run walkthrough.

Implemented locally: `reflect setup` rejects invalid provider choices, accepts
opaque nonempty keys, and prompts for a cached model after catalog discovery.
Reflect preflight requests a provider-scoped tool capability check from Relay
and reports supported, unsupported, or unknown metadata. Setup keeps a curated
seven-provider menu, using Relay's credential mappings, and prefers models with
known tool support. The full provider catalog remains available in `/config`.
Live preflight does not send a chat request. The
startup welcome now uses a compact text header, and permission prompts put
one-time allow/deny first while naming the scope of remembered rules.

## User outcome

A new user installs Reflect, chooses a provider and model, and receives one
successful coding-agent answer without editing JSON or guessing which readiness
check to trust. When setup fails, the next action is visible in the CLI.

Measure time from first `reflect` invocation to first completed answer, and
record which step needed manual intervention. Test the journey with a fresh user
config and a disposable credential store, never a real credential in test logs.

## Observed failure

The Linux walkthrough reached `reflect preflight --live` with an Anthropic key
and `anthropic/claude-sonnet-5-5`; preflight reported ready, but the first
`reflect -p "Say hello"` failed with `model does not satisfy requested
capabilities`. Relay's old cached Anthropic offering lacked `function_calling`
metadata. The local Relay backfill fix resolved that instance. The gap remains:
preflight checks model availability and provider access, not the coding agent's
required capabilities or an actual chat turn.

The `reflect setup` command is a separate seven-provider wizard. It accepts an
unknown menu answer as Anthropic, rejects some API keys by prefix, and refreshes
the catalog without selecting a model. Its command help currently promises
provider and model configuration. The in-chat `/config` path has the fuller
workflow. These paths should lead to the same final state.

## Work sequence

### 1. One setup journey in Reflect

- Keep `reflect setup` focused on seven common providers and expose the full
  provider catalog through `/config`. Derive credential mappings and model
  metadata from Relay so both routes reach the same saved selection.
- Preserve already stored credentials. Treat an empty or invalid selection as a
  retry or explicit cancel, never silently select Anthropic.
- Accept opaque provider keys after basic nonempty input validation. Let Relay
  perform provider-specific live authentication; do not infer validity from a
  key prefix.
- Finish with the selected provider/model, the exact verification command,
  and a clear next step on failure. Never print the key.

### 2. Capability-aware readiness in Relay

- Expose whether the active provider's selected offering satisfies the coding
  agent's required capabilities through Relay's host-facing engine facade.
  Keep provider/catalog internals inside Relay.
- Distinguish **supported**, **unsupported**, and **unknown** metadata. An
  unknown capability must not be silently reported as supported. Give the
  user an actionable model or catalog-refresh suggestion.
- Keep local readiness and provider-list verification distinct from an actual
  chat request. If an end-to-end smoke check is added, make it an explicit
  command because it can incur provider usage.
- Ensure a stale cache that is missing known Anthropic tool metadata is
  covered by a regression test. Add equivalent provider fixtures only where
  provider behavior is established by source or live contract evidence.

### 3. Clear result in Reflect

- Translate Relay capability failures into a provider/model-specific message
  that points to `/config` and `reflect models refresh` when appropriate.
- Label `preflight --live` as provider/model-list verification. Say "chat
  verified" only after a successful chat turn.
- Show one answer and one aggregated usage line in print mode; retain the
  existing regression coverage for this recently fixed path.

## Acceptance journey

1. On a fresh Linux profile, configure an already stored key or paste a new
   one, select a tool-capable model, and complete `reflect -p "Say hello"`.
2. With a catalog entry missing tool metadata, setup/preflight reports an
   unknown capability or offers a repair action before claiming coding-agent
   readiness.
3. With an explicitly unsupported model, Reflect explains the selection
   problem without exposing credentials or a raw Relay error as the only help.
4. With an invalid key, live verification fails with an authentication result;
   key format alone does not block saving it to the OS secret store.
5. On a repeat run, setup reuses the credential and lets the user change the
   model without pasting the key again.

## Later improvements

Once the first task works consistently, simplify the visible `plan`/`act`
choice and show a compact task timeline with changed files, tests, and a
resume point. OpenCode's documented plan/build agents show the value of a
clear mode choice; Entire's Git-native sessions show a useful continuity
pattern. Reflect already has sessions and checkpoints, so evaluate their
surface before adding another storage system.

References: [OpenCode](https://github.com/anomalyco/opencode),
[Entire CLI](https://github.com/entireio/cli),
[Pi](https://github.com/earendil-works/pi).
