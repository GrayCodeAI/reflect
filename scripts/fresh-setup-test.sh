#!/usr/bin/env bash
# Fresh first-run /config test — build decoder and optional isolated ~/.decoder.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "==> Ensuring ecosystem repos are present (make setup)..."
make setup

echo ""
echo "==> Building decoder..."
go build -o decoder ./cmd/decoder/
echo "    $(./decoder --version 2>/dev/null || echo built)"

echo ""
echo "==> Credential status (macOS Keychain / secret service):"
./decoder credentials status || true

echo ""
echo "==> Optional: isolated config dir (sessions/settings separate from ~/.decoder)"
ISOLATED="${DECODER_FRESH_CONFIG_DIR:-$(mktemp -d)/decoder-fresh}"
mkdir -p "$ISOLATED"
rm -f "$ISOLATED/learned_credential_prefixes.json"
rm -f "$ISOLATED/settings.json"
echo "    DECODER_CONFIG_DIR=$ISOLATED"

echo ""
echo "To match first-run Setup (no API key in Keys tab):"
echo "  - Remove stored keys: ./decoder credentials remove <provider>   (e.g. anthropic, openai)"
echo "  - This script already clears the isolated settings.json so model selection starts fresh"
echo ""
echo "Run TUI (from $ROOT):"
echo "  export DECODER_CONFIG_DIR=\"$ISOLATED\""
echo "  ./decoder"
echo ""
echo "In Setup: Keys → Add key · <Gateway> → paste → one probe → Models."
echo "After setup: ./decoder path  and  ./decoder preflight"
