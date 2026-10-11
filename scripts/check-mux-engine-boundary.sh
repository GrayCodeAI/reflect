#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if command -v rg >/dev/null 2>&1; then
  mux_imports="$(
    rg -n '"github\.com/GrayCodeAI/mux/[^\"]+"' \
      --glob '*.go' --glob '!*_test.go' . || true
  )"
else
  mux_imports="$(
    grep -RInE --include='*.go' --exclude='*_test.go' \
      '"github\.com/GrayCodeAI/mux/[^\"]+"' . || true
  )"
fi
# Host contract surface is exactly four packages: engine (facade), llm (DTOs
# and the Provider port), graph (portable graph vocabulary), tools (tool-call
# contracts). See mux/README.md "Ecosystem Boundaries".
violations="$(printf '%s\n' "$mux_imports" | grep -vE '"github\.com/GrayCodeAI/mux/(engine|llm|graph|tools)(/|\")' || true)"

if [[ -n "$violations" ]]; then
  echo "direct production imports below the mux/engine facade found:"
  echo "$violations"
  echo
  echo "route every Decoder production integration through github.com/GrayCodeAI/mux/engine"
  exit 1
fi

if command -v rg >/dev/null 2>&1; then
  credential_symbols="$(rg -n '\b(apiKeys|SetAPIKey|SetAPIKeys)\b' internal/engine --glob '*.go' --glob '!*_test.go' || true)"
else
  credential_symbols="$(grep -RInE --include='*.go' --exclude='*_test.go' '(^|[^[:alnum:]_])(apiKeys|SetAPIKey|SetAPIKeys)([^[:alnum:]_]|$)' internal/engine || true)"
fi
if [[ -n "$credential_symbols" ]]; then
	printf '%s\n' "$credential_symbols"
  echo
  echo "provider credentials must not enter Decoder's agent/session layer"
  exit 1
fi

echo "mux engine boundary passed (zero lower-level production imports)"
