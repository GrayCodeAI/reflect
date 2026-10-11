#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if command -v rg >/dev/null 2>&1; then
  violations="$(
    rg -n '"github\.com/GrayCodeAI/mux/client(?:/[^\"]*)?"' \
      --glob '*.go' --glob '!*_test.go' . || true
  )"
else
  violations="$(
    grep -RInE --include='*.go' --exclude='*_test.go' \
      '"github\.com/GrayCodeAI/mux/client(/[^\"]*)?"' . || true
  )"
fi

if [[ -n "${violations}" ]]; then
  echo "forbidden direct imports of github.com/GrayCodeAI/mux/client found:"
  echo "${violations}"
  echo
  echo "Decoder production code must go through github.com/GrayCodeAI/mux/engine"
  exit 1
fi

echo "mux/client boundary guard passed (zero production imports)"
