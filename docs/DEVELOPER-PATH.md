# Reflect Developer Path

This guide explains what `reflect path` checks and how to get a fresh developer machine ready to use Reflect safely.

## What "developer path" means

For Reflect, the developer path is the minimum local setup required to chat, edit code, and keep credentials off disk:

- A provider credential stored in the OS secret store
- A model selected in Reflect settings
- A local model catalog available through relay
- No plaintext API keys left in Relay's configured `provider.json` or legacy env files
- Safe defaults for Bash execution and filesystem access
- Optional but healthy local memory and token-pipeline state

Run the report at any time:

```bash
reflect path
reflect path --strict
reflect doctor
reflect preflight
```

## Setup checklist

### 1. Build and workspace setup

If you are contributing from source, clone Reflect as the main CLI. The support
repositories are independent sibling checkouts when you need the full local
workspace; they are not nested under Reflect:

```bash
mkdir graycode-eco && cd graycode-eco
git clone https://github.com/GrayCodeAI/reflect
git clone https://github.com/GrayCodeAI/relay
cd reflect
make setup
go build -o reflect ./cmd/reflect
```

`make setup` validates the canonical repository manifest and regenerates the
parent `../go.work` from the local Go repositories marked `workspace: true`.
Reflect can also be built as a standalone checkout with `GOWORK=off go build ./cmd/reflect`;
the sibling workspace is only required for cross-repository development and
boundary checks.

### 2. Configure credentials

Start Reflect and use `/config` to paste an API key or configure a local provider like Ollama.

Reflect stores credentials in the macOS Keychain or Linux secret store. It should not rely on shell env vars, `.env`, or plaintext config files for provider secrets.

Useful checks:

```bash
reflect credentials status
reflect preflight
```

`reflect preflight` is a local-ready check; it does not contact a provider. To
live-verify the selected provider credential and connectivity, use `/config`
validation or `reflect models list <provider> --live`.

### 3. Select a model

Pick a model in `/config`. Reflect stores the selected model in settings and uses relay for provider routing and catalog resolution.

If the catalog is missing or empty:

```bash
reflect models refresh
```

## Security checks

`reflect path` treats these as important security conditions:

- Relay's resolved `provider.json` must not contain secret fields
- sensitive files like provider config and SSH paths should be blocked from agent reads

Relay resolves provider state from `RELAY_CONFIG_DIR` first, then the
platform user-config directory.
Reflect protects that resolved path even when it is customized or symlinked.

If Reflect detects secret fields in `provider.json`, back up the file, remove those fields manually, and save your keys again through `/config`.

Read the full credential and isolation model in [SECURITY-DEVELOPER.md](./SECURITY-DEVELOPER.md).

## Execution model

Reflect executes agent commands directly on the host. No Docker daemon or
container runtime is required, and there is no container fallback to configure.

## Ecosystem checks

`reflect path` also verifies the provider layer behind Reflect:

- `relay` for provider routing and local preflight readiness
- the embedded token pipeline for estimation and compression

If you want the broader status summary:

```bash
reflect ecosystem
reflect doctor
```

## Typical recovery path

If `reflect path` says you are not ready, this is the intended order:

1. Run `reflect`
2. Open `/config`
3. Paste an API key or configure Ollama
4. Pick a model
5. Re-run `reflect preflight`
6. Re-run `reflect path`

If security items still fail, fix those before treating the machine as ready.
