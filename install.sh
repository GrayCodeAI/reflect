#!/bin/sh
# Install decoder (the GrayCode terminal AI coding agent) from a GitHub release.
#
#   curl -fsSL https://raw.githubusercontent.com/GrayCodeAI/decoder/v0.3.0/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/GrayCodeAI/decoder/v0.3.0/install.sh | sh -s -- --version 0.3.0 --prefix "$HOME/.decoder"
#
# Steps:
#   1. Resolve the version: --version / DECODER_VERSION, otherwise the latest
#      GitHub release. Releases before v0.3.0 predate the rename to decoder (they
#      ship hawk_* archives) and are refused.
#   2. Download decoder_<version>_<os>_<arch>.tar.gz (.zip on Windows) and
#      checksums.txt from the release.
#   3. When cosign is installed, verify checksums.txt against the release's
#      Sigstore bundle (checksums.txt.sigstore.json) with the exact identity of
#      this repository's release workflow for that tag. A missing or invalid
#      signature aborts the install. Without cosign the signature is NOT
#      verified and the script says so; DECODER_REQUIRE_COSIGN=1 makes cosign
#      mandatory.
#   4. Always verify the archive's SHA-256 against checksums.txt.
#   5. Install <prefix>/bin/decoder-<version> and atomically point <prefix>/bin/decoder
#      at it (a copy named decoder.exe on Windows).
#
# Options:
#   --version <ver>   install this version (e.g. 0.3.0 or v0.3.0)
#   --prefix <dir>    install under <dir>/bin (default: $DECODER_HOME or ~/.decoder)
#   -h, --help        show this help
#
# Environment:
#   DECODER_VERSION=<ver>        same as --version (the flag wins)
#   DECODER_HOME=<dir>           same as --prefix (the flag wins)
#   DECODER_REQUIRE_COSIGN=1     fail unless the cosign signature can be verified
#   DECODER_INSTALL_REPO=<o/r>   install from a fork's releases (default GrayCodeAI/decoder);
#                            the cosign identity follows the chosen repository
set -eu

MIN_VERSION="0.3.0"
BINARY="decoder"
OIDC_ISSUER="https://token.actions.githubusercontent.com"
REPO="${DECODER_INSTALL_REPO:-GrayCodeAI/decoder}"

say() { printf '%s\n' "$*"; }
warn() { printf 'WARNING: %s\n' "$*" >&2; }
die() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Install decoder from a GitHub release.

Usage: install.sh [--version <ver>] [--prefix <dir>]

  --version <ver>   install this version (e.g. 0.3.0); default: latest release
  --prefix <dir>    install under <dir>/bin; default: $DECODER_HOME or ~/.decoder
  -h, --help        show this help

Environment: DECODER_VERSION, DECODER_HOME, DECODER_REQUIRE_COSIGN=1, DECODER_INSTALL_REPO.
EOF
}

usage_error() {
  printf 'Error: %s\n\n' "$*" >&2
  usage >&2
  exit 2
}

# fetch URL DEST — HTTPS only, fail on HTTP errors.
fetch() {
  curl -fsSL --proto '=https' --tlsv1.2 --retry 3 -o "$2" "$1"
}

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  elif command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 "$1" | awk '{print $NF}'
  else
    die "need sha256sum, shasum or openssl to verify the download"
  fi
}

# version_core_ge A B — true when MAJOR.MINOR.PATCH of A >= that of B.
version_core_ge() {
  awk -v a="$1" -v b="$2" 'BEGIN {
    split(a, x, /[.+-]/); split(b, y, /[.+-]/)
    for (i = 1; i <= 3; i++) {
      if (x[i] + 0 > y[i] + 0) exit 0
      if (x[i] + 0 < y[i] + 0) exit 1
    }
    exit 0
  }'
}

# --- Arguments -----------------------------------------------------------------
VERSION_PIN="${DECODER_VERSION:-}"
PREFIX="${DECODER_HOME:-}"
while [ $# -gt 0 ]; do
  case "$1" in
    --version)
      [ $# -ge 2 ] && [ -n "$2" ] || usage_error "--version needs a value"
      VERSION_PIN=$2
      shift 2
      ;;
    --version=*)
      VERSION_PIN=${1#--version=}
      [ -n "$VERSION_PIN" ] || usage_error "--version needs a value"
      shift
      ;;
    --prefix)
      [ $# -ge 2 ] && [ -n "$2" ] || usage_error "--prefix needs a directory"
      PREFIX=$2
      shift 2
      ;;
    --prefix=*)
      PREFIX=${1#--prefix=}
      [ -n "$PREFIX" ] || usage_error "--prefix needs a directory"
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      usage_error "unknown argument: $1"
      ;;
  esac
done

if [ -z "$PREFIX" ]; then
  [ -n "${HOME:-}" ] || die "HOME is not set; pass --prefix <dir>"
  PREFIX="$HOME/.decoder"
fi

if ! printf '%s' "$REPO" | grep -Eq '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$'; then
  die "DECODER_INSTALL_REPO must look like owner/repo (got: $REPO)"
fi

command -v curl >/dev/null 2>&1 || die "curl is required"

# --- Platform --------------------------------------------------------------------
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux) OS=linux ;;
  darwin) OS=darwin ;;
  mingw* | msys* | cygwin*) OS=windows ;;
  *) die "unsupported operating system: $os (release archives exist for linux, darwin and windows)" ;;
esac
arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) ARCH=amd64 ;;
  aarch64 | arm64) ARCH=arm64 ;;
  *) die "unsupported architecture: $arch (release archives exist for amd64 and arm64)" ;;
esac
if [ "$OS" = windows ]; then
  ARCHIVE_EXT=zip
  BIN_NAME="${BINARY}.exe"
else
  ARCHIVE_EXT=tar.gz
  BIN_NAME="$BINARY"
fi

# --- Version ---------------------------------------------------------------------
if [ -n "$VERSION_PIN" ]; then
  VERSION=${VERSION_PIN#v}
else
  latest_json=$(curl -fsSL --proto '=https' --tlsv1.2 --retry 3 \
    -H 'Accept: application/vnd.github+json' \
    "https://api.github.com/repos/${REPO}/releases/latest") ||
    die "could not query the latest release of ${REPO}; pass --version <ver>"
  tag=$(printf '%s\n' "$latest_json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)
  [ -n "$tag" ] || die "could not determine the latest release of ${REPO}; pass --version <ver>"
  VERSION=${tag#v}
fi

if ! printf '%s' "$VERSION" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'; then
  die "invalid version '$VERSION' (expected MAJOR.MINOR.PATCH, e.g. 0.3.0)"
fi
if ! version_core_ge "$VERSION" "$MIN_VERSION"; then
  die "v${VERSION} predates the rename to decoder: releases before v${MIN_VERSION} ship hawk_* archives that this script cannot install. Install v${MIN_VERSION} or later, or build from source (https://github.com/${REPO}#readme)."
fi

TAG="v${VERSION}"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
ARCHIVE_NAME="${BINARY}_${VERSION}_${OS}_${ARCH}.${ARCHIVE_EXT}"

TMP=$(mktemp -d 2>/dev/null || mktemp -d -t decoder-install)
trap 'rm -rf "$TMP"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

say "Downloading decoder ${TAG} for ${OS}/${ARCH}..."
fetch "${BASE_URL}/${ARCHIVE_NAME}" "$TMP/$ARCHIVE_NAME" ||
  die "could not download ${BASE_URL}/${ARCHIVE_NAME} (does release ${TAG} exist for ${OS}/${ARCH}?)"
fetch "${BASE_URL}/checksums.txt" "$TMP/checksums.txt" ||
  die "could not download checksums.txt for ${TAG}"

# --- Signature (cosign) ----------------------------------------------------------
IDENTITY="https://github.com/${REPO}/.github/workflows/release.yml@refs/tags/${TAG}"
if command -v cosign >/dev/null 2>&1; then
  say "Verifying the checksums.txt signature with cosign..."
  fetch "${BASE_URL}/checksums.txt.sigstore.json" "$TMP/checksums.txt.sigstore.json" ||
    die "release ${TAG} has no checksums.txt.sigstore.json signature bundle; refusing to install an unsigned release while cosign is available"
  if ! cosign verify-blob \
    --new-bundle-format \
    --bundle "$TMP/checksums.txt.sigstore.json" \
    --certificate-identity "$IDENTITY" \
    --certificate-oidc-issuer "$OIDC_ISSUER" \
    "$TMP/checksums.txt" >"$TMP/cosign.log" 2>&1; then
    cat "$TMP/cosign.log" >&2
    die "cosign could not verify checksums.txt for ${TAG} against ${IDENTITY}; the release may have been tampered with (cosign v2.4 or later is required, v3 recommended)"
  fi
  SIGNATURE_STATUS="verified with cosign (${IDENTITY})"
elif [ "${DECODER_REQUIRE_COSIGN:-0}" = 1 ]; then
  die "DECODER_REQUIRE_COSIGN=1 but cosign is not installed (see https://docs.sigstore.dev/cosign/system_config/installation/)"
else
  warn "cosign is not installed, so the release signature was NOT verified."
  warn "The SHA-256 check only detects corrupted downloads, not a tampered release."
  warn "Install cosign (https://docs.sigstore.dev) or set DECODER_REQUIRE_COSIGN=1 to require it."
  SIGNATURE_STATUS="NOT verified (cosign not installed)"
fi

# --- Checksum (always) -----------------------------------------------------------
# Exact field match, not a regex: '.' in the archive name must not act as a
# wildcard, and exactly one line may match.
EXPECTED=$(awk -v f="$ARCHIVE_NAME" '$2 == f || $2 == "*" f { print $1 }' "$TMP/checksums.txt")
[ -n "$EXPECTED" ] || die "checksums.txt has no entry for ${ARCHIVE_NAME}"
[ "$(printf '%s\n' "$EXPECTED" | wc -l | tr -d ' ')" = 1 ] || die "checksums.txt has more than one entry for ${ARCHIVE_NAME}"
ACTUAL=$(sha256_of "$TMP/$ARCHIVE_NAME")
if [ "$ACTUAL" != "$EXPECTED" ]; then
  printf '  expected: %s\n  actual:   %s\n' "$EXPECTED" "$ACTUAL" >&2
  die "SHA-256 mismatch for ${ARCHIVE_NAME}"
fi
say "Checksum verified."

# --- Extract ---------------------------------------------------------------------
mkdir -p "$TMP/extract"
if [ "$ARCHIVE_EXT" = zip ]; then
  command -v unzip >/dev/null 2>&1 || die "unzip is required to install Windows release archives"
  unzip -q "$TMP/$ARCHIVE_NAME" -d "$TMP/extract"
else
  tar -xzf "$TMP/$ARCHIVE_NAME" -C "$TMP/extract"
fi
[ -f "$TMP/extract/$BIN_NAME" ] || die "${ARCHIVE_NAME} does not contain ${BIN_NAME}"

# --- Versioned install -----------------------------------------------------------
# A binary is never overwritten in place. On macOS (and any codesigned
# platform) replacing a file that a running process has mmap'd invalidates the
# kernel's code-signature cache and the kernel then SIGKILLs that process.
# Installing into a per-version file and swapping a symlink keeps running decoder
# processes on the old inode. The symlink is created under a temporary name and
# renamed into place so the swap is atomic. Windows lacks reliable non-admin
# symlinks, so the launcher there is a plain copy.
BINDIR="$PREFIX/bin"
mkdir -p "$BINDIR"
if [ "$OS" = windows ]; then
  mv -f "$TMP/extract/$BIN_NAME" "$BINDIR/decoder-${VERSION}.exe"
  cp -f "$BINDIR/decoder-${VERSION}.exe" "$BINDIR/decoder.exe"
  INSTALLED="$BINDIR/decoder-${VERSION}.exe (launcher: $BINDIR/decoder.exe)"
else
  mv -f "$TMP/extract/$BIN_NAME" "$BINDIR/decoder-${VERSION}"
  chmod 0755 "$BINDIR/decoder-${VERSION}"
  rm -f "$BINDIR/decoder.tmp"
  ln -s "decoder-${VERSION}" "$BINDIR/decoder.tmp"
  mv -f "$BINDIR/decoder.tmp" "$BINDIR/decoder"
  INSTALLED="$BINDIR/decoder-${VERSION} (linked: $BINDIR/decoder)"
fi

say ""
say "Installed decoder ${TAG} to ${INSTALLED}"
say "  checksum:  verified (SHA-256 from checksums.txt)"
say "  signature: ${SIGNATURE_STATUS}"
say ""
case ":${PATH:-}:" in
  *":$BINDIR:"*) ;;
  *)
    say "Add $BINDIR to your PATH, e.g."
    say "  export PATH=\"\$PATH:$BINDIR\""
    say ""
    ;;
esac
say "Restart running decoder sessions to pick up the new binary; they keep running"
say "the previous version until restarted."
