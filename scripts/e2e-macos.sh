#!/usr/bin/env bash
# macOS End-to-End Test Suite for reflect
# Run: bash scripts/e2e-macos.sh
set -euo pipefail

PASS=0
FAIL=0

pass() { PASS=$((PASS+1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL+1)); echo "  FAIL: $1"; }

# Ensure binary exists
if [ ! -f ./reflect ]; then
    echo "Building reflect..."
    go build -o reflect ./cmd/reflect
fi

echo "=== macOS E2E Tests ==="
echo

# 1. Version check
echo "--- version ---"
./reflect version 2>&1 | grep -q "[0-9]\+\.[0-9]\+\.[0-9]\+" && pass "version output" || fail "version output"

# 2. Doctor / preflight
echo "--- doctor ---"
./reflect doctor 2>&1 | head -20 | grep -qi "reflect\|ok\|check" && pass "doctor runs" || fail "doctor failed"

# 3. Preflight
echo "--- preflight ---"
./reflect preflight 2>&1 | head -20 && pass "preflight runs" || fail "preflight failed"

# 4. No API keys in provider.json
echo "--- provider.json has no plaintext API keys ---"
if [ -f ~/.reflect/provider.json ]; then
    if grep -qi '"api_key"' ~/.reflect/provider.json 2>/dev/null; then
        fail "provider.json contains api_key field"
    else
        pass "provider.json has no plaintext API keys"
    fi
else
    pass "provider.json not present (no setup done)"
fi

# 5. Credential store (macOS Keychain)
echo "--- credentials list ---"
./reflect credentials list 2>&1 | head -10 && pass "credentials list runs" || fail "credentials list failed"

# 6. Shell completions generate
echo "--- shell completions ---"
./reflect completion bash 2>&1 | head -5 | grep -q "complete\|completion" && pass "bash completions generated" || fail "bash completions"
./reflect completion zsh 2>&1 | head -5 | grep -q "compdef\|completion" && pass "zsh completions generated" || fail "zsh completions"

# 7. Help output
echo "--- help ---"
./reflect --help 2>&1 | head -5 | grep -qi "reflect\|usage\|flag" && pass "help output" || fail "help output"

# 9. Config subcommands
echo "--- config commands ---"
./reflect config --help 2>&1 | head -5 | grep -qi "manage\|config\|edit" && pass "config help" || fail "config help"

# 10. Session list (no-op test)
echo "--- sessions list ---"
./reflect sessions list 2>&1 | head -5 && pass "sessions list runs" || fail "sessions list"

echo
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
