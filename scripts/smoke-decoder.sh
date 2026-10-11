#!/usr/bin/env bash
# Quick smoke test before releases or after ecosystem wiring changes.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN="${SMOKE_DECODER_BIN:-/tmp/decoder-smoke}"
echo "== build =="
go build -mod=readonly -o "$BIN" ./cmd/decoder

echo "== decoder doctor =="
set +o pipefail
"$BIN" doctor >/dev/null 2>&1 || true
set -o pipefail

echo "== decoder ecosystem =="
"$BIN" ecosystem >/dev/null

echo "== decoder path =="
set +o pipefail
if ! PATH_OUT="$("$BIN" path 2>&1)"; then
  echo "path reported readiness problems — ordered checklist with fix commands:"
  echo "$PATH_OUT"
fi
set -o pipefail

echo "== ecosystem tests =="
go test ./internal/config/ -run TestFormatEcosystemPanel -count=1
go test ./cmd/ -run 'TestDoctor|TestHarrier|TestEcosystem|TestPath' -count=1
go test ./internal/config/ -run 'DeveloperPath|FormatEcosystemPanel' -count=1
go test ./internal/intelligence/memory/ -run 'FormatHarrier|ShouldAutoRemember' -count=1

echo "== verify developer path =="
./scripts/verify-developer-path.sh

echo "== smoke ok =="
