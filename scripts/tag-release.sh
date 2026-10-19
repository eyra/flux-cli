#!/usr/bin/env bash
# Tags HEAD with the next version, bumped by the commit subjects since the
# latest vX.Y.Z tag: feat/feature → minor, fix/refactor → patch, anything else
# (chore, docs, ci, merge commits) → nothing. Only the highest bump counts.
# Major bumps are tagged by hand.
# Writes tag=<tag> to $GITHUB_OUTPUT when it tagged, so the release can run.
set -euo pipefail

minor_re='^(feat|feature)(\([^)]*\))?!?:'
patch_re='^(fix|refactor)(\([^)]*\))?!?:'

latest="$(git tag --list 'v*' --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -n 1 || true)"
if [ -n "$latest" ]; then
  range="$latest..HEAD"
else
  latest=v0.0.0
  range=HEAD
fi

bump=none
while IFS= read -r subject; do
  if [[ "$subject" =~ $minor_re ]]; then
    bump=minor
    break
  elif [[ "$subject" =~ $patch_re ]]; then
    bump=patch
  fi
done < <(git log --format=%s "$range")

IFS=. read -r major minor patch <<< "${latest#v}"
case "$bump" in
  minor) tag="v$major.$((minor + 1)).0" ;;
  patch) tag="v$major.$minor.$((patch + 1))" ;;
  *)
    echo "No feat, feature, fix or refactor commits since $latest, nothing to release"
    exit 0
    ;;
esac

git tag "$tag" HEAD
git push origin "refs/tags/$tag"
echo "Tagged $tag ($bump bump from $latest)"
echo "tag=$tag" >> "${GITHUB_OUTPUT:-/dev/null}"
