#!/usr/bin/env bash
set -euo pipefail

# Retain the old entrypoint, using an explicit, repo-local module overlay.
# Never create, remove, or rewrite a workspace in the parent directory.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ $# -ne 1 || -z "$1" || ! -f "$1/go.mod" ]]; then
  echo "usage: $0 /path/to/relay (creates optional go.local.mod; ordinary builds need no sibling checkout)" >&2
  exit 2
fi
RELAY_DIR="$(cd "$1" && pwd)"
if [[ "$(awk '$1 == "module" {print $2; exit}' "${RELAY_DIR}/go.mod")" != github.com/GrayCodeAI/relay ]]; then
  echo "the supplied checkout must declare module github.com/GrayCodeAI/relay" >&2
  exit 2
fi
if [[ -e "${ROOT_DIR}/go.local.mod" || -e "${ROOT_DIR}/go.local.sum" ]]; then
  echo "go.local.mod or go.local.sum already exists; edit the existing overlay explicitly" >&2
  exit 1
fi

stage="$(mktemp -d "${ROOT_DIR}/.relay-overlay.XXXXXX")"
trap 'rm -rf "$stage"' EXIT
cp "${ROOT_DIR}/go.mod" "${stage}/go.local.mod"
cp "${ROOT_DIR}/go.sum" "${stage}/go.local.sum"
GOWORK=off go mod edit -modfile="${stage}/go.local.mod" -replace="github.com/GrayCodeAI/relay=${RELAY_DIR}"
mv "${stage}/go.local.sum" "${ROOT_DIR}/go.local.sum"
mv "${stage}/go.local.mod" "${ROOT_DIR}/go.local.mod"
echo "Created go.local.mod; run GOWORK=off go test -modfile=go.local.mod ./..."
