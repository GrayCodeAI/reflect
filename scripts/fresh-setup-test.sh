#!/usr/bin/env bash
# Fresh first-run /config test — build reflect and optional isolated ~/.reflect.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "==> Ensuring ecosystem repos are present (make setup)..."
make setup

echo ""
echo "==> Building reflect..."
go build -o reflect ./cmd/reflect/
echo "    $(./reflect --version 2>/dev/null || echo built)"

echo ""
echo "==> Credential status (macOS Keychain / secret service):"
./reflect credentials status || true

echo ""
echo "==> Optional: isolated config dir (sessions/settings separate from ~/.reflect)"
ISOLATED="${REFLECT_FRESH_CONFIG_DIR:-$(mktemp -d)/reflect-fresh}"
mkdir -p "$ISOLATED"
rm -f "$ISOLATED/learned_credential_prefixes.json"
rm -f "$ISOLATED/settings.json"
echo "    REFLECT_CONFIG_DIR=$ISOLATED"

echo ""
echo "To match first-run Setup (no API key in Keys tab):"
echo "  - Remove stored keys: ./reflect credentials remove <provider>   (e.g. anthropic, openai)"
echo "  - This script already clears the isolated settings.json so model selection starts fresh"
echo ""
echo "Run TUI (from $ROOT):"
echo "  export REFLECT_CONFIG_DIR=\"$ISOLATED\""
echo "  ./reflect"
echo ""
echo "In Setup: Keys → Add key · <Gateway> → paste → one probe → Models."
echo "After setup: ./reflect path  and  ./reflect preflight"
