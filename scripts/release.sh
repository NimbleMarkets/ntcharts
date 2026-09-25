#!/usr/bin/env bash
# release.sh - cut an ntcharts release across the root and nested modules.
#
# Usage: task release VERSION=v2.X.Y
#
# The nested picture/chartpicture module cannot carry a v2 tag because its
# module path does not end in /v2, so it is tagged picture/chartpicture/v0.X.Y
# with the root's minor and patch. Consumers ignore the local `replace`
# directives, so every nested go.mod must require the new root version.
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
NESTED_TAG="picture/chartpicture/${NESTED}"

cd "$(git rev-parse --show-toplevel)"

if [[ -n "$(git status --porcelain)" ]]; then
	echo "release: working tree is dirty; commit or stash first" >&2
	exit 1
fi
for tag in "$VERSION" "$NESTED_TAG"; do
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
git tag "$NESTED_TAG"

echo
echo "Tagged ${VERSION} and ${NESTED_TAG} on $(git rev-parse --short HEAD)."
echo "Review the commit, then publish with:"
echo "  git push origin $(git branch --show-current) ${VERSION} ${NESTED_TAG}"
