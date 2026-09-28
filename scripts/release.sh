#!/usr/bin/env bash
# release.sh - cut an ntcharts release across the root and nested modules.
#
# Usage: task release VERSION=v2.X.Y
#
# The nested picture/chartpicture and examples/shaders modules cannot carry a
# v2 tag because their module paths do not end in /v2, so they are tagged
# <dir>/v0.X.Y with the root's minor and patch. Consumers ignore the local
# `replace` directives (and examples/shaders has none, so that
# `go run .../examples/shaders@latest` works), so every nested go.mod must
# require the new root version.
#
# This script commits and tags but never pushes; it prints the push command.
set -euo pipefail

MODULE=github.com/NimbleMarkets/ntcharts/v2
VERSION="${1:-}"

if [[ ! "$VERSION" =~ ^v2\.([0-9]+)\.([0-9]+)$ ]]; then
	echo "usage: task release VERSION=v2.X.Y (got '${VERSION}')" >&2
	exit 2
fi
NESTED="v0.${BASH_REMATCH[1]}.${BASH_REMATCH[2]}"
NESTED_TAGS=("picture/chartpicture/${NESTED}" "examples/shaders/${NESTED}")

cd "$(git rev-parse --show-toplevel)"

if [[ -n "$(git status --porcelain)" ]]; then
	echo "release: working tree is dirty; commit or stash first" >&2
	exit 1
fi
for tag in "$VERSION" "${NESTED_TAGS[@]}"; do
	if git rev-parse -q --verify "refs/tags/${tag}" >/dev/null; then
		echo "release: tag ${tag} already exists" >&2
		exit 1
	fi
done
if ! grep -q "^## ${VERSION} (unreleased)" CHANGELOG.md; then
	echo "release: CHANGELOG.md has no '## ${VERSION} (unreleased)' heading" >&2
	exit 1
fi

echo "release: bumping nested modules to ${MODULE}@${VERSION}"
for dir in cmd examples examples/shaders picture/chartpicture; do
	(cd "$dir" && go mod edit -require="${MODULE}@${VERSION}")
done
(cd examples && go mod edit -require="${MODULE}/picture/chartpicture@${NESTED}")
go work sync

DATE=$(date +%Y-%m-%d)
sed -i.bak "s|^## ${VERSION} (unreleased)|## ${VERSION} (${DATE})|" CHANGELOG.md
rm -f CHANGELOG.md.bak

echo "release: running tests"
task test

git add -A
git commit -q -s -m "chore(release): ${VERSION}"
git tag "$VERSION"
for tag in "${NESTED_TAGS[@]}"; do
	git tag "$tag"
done

echo
echo "Tagged ${VERSION} ${NESTED_TAGS[*]} on $(git rev-parse --short HEAD)."
echo "Review the commit, then publish with:"
echo "  git push origin $(git branch --show-current) ${VERSION} ${NESTED_TAGS[*]}"
