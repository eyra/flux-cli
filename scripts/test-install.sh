#!/bin/bash
#
# Checks that the goreleaser archives and install.sh agree: every archive in
# dist/ must contain the binary and skill/SKILL.md, and installing the archive
# for this machine through install.sh must put both in place.
#
# Usage: scripts/test-install.sh [dist-dir]
#   Build the archives first: goreleaser release --snapshot --clean
#
# Runs with a temporary HOME, and INSTALL_DIR and SKILL_DIR unset so they
# default into it, from a directory without go.mod, so it never touches the
# real installation.

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

# run_install: install $ARCHIVE through install.sh inside the sandbox. A
# developer's exported INSTALL_DIR or SKILL_DIR must never leak in.
run_install() {
  (
    cd "$SANDBOX/work"
    [ ! -f go.mod ] || fail "work directory has a go.mod"
    env -u INSTALL_DIR -u SKILL_DIR HOME="$SANDBOX/home" FLUX_ARCHIVE="$ARCHIVE" \
      bash "$ROOT/install.sh"
  )
}

run_install

BIN="$SANDBOX/home/.local/bin/flux"
SKILL_DIR="$SANDBOX/home/.claude/skills/flux"
SKILL="$SKILL_DIR/SKILL.md"
[ -x "$BIN" ] || fail "install.sh did not install $BIN"
[ -f "$SKILL" ] || fail "install.sh did not install $SKILL"
cmp -s "$SKILL" "$ROOT/skill/SKILL.md" || fail "installed SKILL.md differs from skill/SKILL.md"
"$BIN" --version >/dev/null 2>&1 || "$BIN" --help >/dev/null || fail "installed flux does not run"

echo "ok: install.sh installed $(basename "$ARCHIVE") (binary and skill) under a temporary HOME"

# Reinstalling over an unchanged skill makes no backup.
run_install >/dev/null
BACKUPS=("$SKILL_DIR"/SKILL.md.bak-*)
[ ${#BACKUPS[@]} -eq 0 ] || fail "reinstall over an unchanged skill made a backup"
echo "ok: reinstall over an unchanged skill makes no backup"

# Reinstalling over a locally edited skill backs it up first.
echo "local edit" >>"$SKILL"
cp "$SKILL" "$SANDBOX/edited.md"
OUT="$(run_install)"
BACKUPS=("$SKILL_DIR"/SKILL.md.bak-*)
[ ${#BACKUPS[@]} -eq 1 ] || fail "expected one backup of the edited skill, found ${#BACKUPS[@]}"
cmp -s "${BACKUPS[0]}" "$SANDBOX/edited.md" || fail "backup does not hold the edited skill"
cmp -s "$SKILL" "$ROOT/skill/SKILL.md" || fail "edited skill was not replaced by the release skill"
grep -qF "${BACKUPS[0]}" <<<"$OUT" || fail "install.sh did not print the backup path"
echo "ok: reinstall over an edited skill backs it up to $(basename "${BACKUPS[0]}")"
