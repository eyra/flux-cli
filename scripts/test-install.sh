#!/bin/bash
#
# Checks that the goreleaser archives and install.sh agree: every archive in
# dist/ must contain the binary and skill/SKILL.md, and installing the archive
# for this machine through install.sh must put both in place.
#
# Usage: scripts/test-install.sh [dist-dir]
#   Build the archives first: goreleaser release --snapshot --clean
#
# Runs with a temporary HOME, INSTALL_DIR and SKILL_DIR, from a directory
# without go.mod, so it never touches the real installation.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$(cd "${1:-$ROOT/dist}" && pwd)"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

shopt -s nullglob
ARCHIVES=("$DIST"/flux_*.tar.gz)
[ ${#ARCHIVES[@]} -gt 0 ] || fail "no flux_*.tar.gz archives in $DIST"

for a in "${ARCHIVES[@]}"; do
  CONTENTS="$(tar -tzf "$a")"
  grep -qx "flux" <<<"$CONTENTS" || fail "$(basename "$a") has no flux binary"
  grep -qx "skill/SKILL.md" <<<"$CONTENTS" || fail "$(basename "$a") has no skill/SKILL.md"
  echo "ok: $(basename "$a") contains flux and skill/SKILL.md"
done

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
[ "$ARCH" = "x86_64" ] && ARCH="amd64"
[ "$ARCH" = "aarch64" ] && ARCH="arm64"
ARCHIVE="$DIST/flux_${OS}_${ARCH}.tar.gz"
[ -f "$ARCHIVE" ] || fail "no archive for this machine: $ARCHIVE"

SANDBOX="$(mktemp -d)"
trap 'rm -rf "$SANDBOX"' EXIT
mkdir -p "$SANDBOX/home" "$SANDBOX/work"

(
  cd "$SANDBOX/work"
  [ ! -f go.mod ] || fail "work directory has a go.mod"
  HOME="$SANDBOX/home" FLUX_ARCHIVE="$ARCHIVE" bash "$ROOT/install.sh"
)

BIN="$SANDBOX/home/.local/bin/flux"
SKILL="$SANDBOX/home/.claude/skills/flux/SKILL.md"
[ -x "$BIN" ] || fail "install.sh did not install $BIN"
[ -f "$SKILL" ] || fail "install.sh did not install $SKILL"
cmp -s "$SKILL" "$ROOT/skill/SKILL.md" || fail "installed SKILL.md differs from skill/SKILL.md"
"$BIN" --version >/dev/null 2>&1 || "$BIN" --help >/dev/null || fail "installed flux does not run"

echo "ok: install.sh installed $(basename "$ARCHIVE") (binary and skill) under a temporary HOME"
