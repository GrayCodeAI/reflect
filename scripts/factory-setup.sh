#!/usr/bin/env bash
set -euo pipefail

reflect_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
workspace_dir=$(dirname "$reflect_dir")
factory_bin=${1:-"${XDG_DATA_HOME:-$HOME/.local/share}/graycode-factory/bin"}
go_bin=${GO_BIN:-go}
mkdir -p "$factory_bin"
factory_bin=$(cd "$factory_bin" && pwd)

build_go() {
  local repo=$1
  local binary=$2
  if [[ ! -f "$workspace_dir/$repo/go.mod" ]]; then
    echo "Missing sibling checkout: $workspace_dir/$repo" >&2
    exit 1
  fi
  echo "Building $binary from $repo"
  (cd "$workspace_dir/$repo" && GOWORK=off "$go_bin" build -o "$factory_bin/$binary" "./cmd/$binary")
}

if [[ ! -f "$workspace_dir/relay/go.mod" ]]; then
  echo "Missing sibling checkout: $workspace_dir/relay" >&2
  exit 1
fi
overlay_dir=$(mktemp -d "${TMPDIR:-/tmp}/graycode-relay-overlay-XXXXXX")
trap 'rm -r "$overlay_dir"' EXIT
cp "$reflect_dir/go.mod" "$overlay_dir/reflect.mod"
cp "$reflect_dir/go.sum" "$overlay_dir/reflect.sum"
echo "Building reflect with local relay from $workspace_dir/relay"
(cd "$reflect_dir" && GOWORK=off "$go_bin" mod edit -modfile="$overlay_dir/reflect.mod" -replace="github.com/GrayCodeAI/relay=$workspace_dir/relay")
(cd "$reflect_dir" && GOWORK=off "$go_bin" build -modfile="$overlay_dir/reflect.mod" -o "$factory_bin/reflect" ./cmd/reflect)
for product in resolve record recall rally rivet rein; do
  build_go "$product" "$product"
done

if [[ ${FACTORY_BUILD_RIDGE:-0} == 1 ]]; then
  if [[ ! -f "$workspace_dir/ridge/Cargo.toml" ]]; then
    echo "Missing sibling checkout: $workspace_dir/ridge" >&2
    exit 1
  fi
  (cd "$workspace_dir/ridge" && cargo build --release --locked)
  cp "$workspace_dir/ridge/target/release/ridge" "$factory_bin/ridge"
elif command -v ridge >/dev/null 2>&1; then
  ln -sf "$(command -v ridge)" "$factory_bin/ridge"
fi

echo "Factory binaries: $factory_bin"
echo "Add to your shell: export PATH=\"$factory_bin:\$PATH\""
echo "Then run: reflect factory doctor --repo /path/to/project"
echo "All checkouts: reflect factory workspace summary --root $workspace_dir"
echo "Verify one: reflect factory workspace verify NAME --root $workspace_dir"
