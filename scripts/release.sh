#!/usr/bin/env bash
# release.sh - cut an ntcharts release across the root and nested modules.
#
# Usage: task release VERSION=v2.X.Y
#
# Published modules share one version, with directory-prefixed tags. cmd is
# internal tooling: it follows the root dependency but gets no release tag.
# Consumer checks run against temporary Git tags before this checkout is tagged.
#
# This script commits and tags but never pushes; it prints the push command.
set -euo pipefail

MODULE=github.com/NimbleMarkets/ntcharts/v2
CHART_MODULE=github.com/NimbleMarkets/ntcharts/picture/chartpicture/v2
ECHARTS_MODULE=github.com/NimbleMarkets/ntcharts/spec/echarts/v2
VERSION="${1:-}"

if [[ ! "$VERSION" =~ ^v2\.([0-9]+)\.([0-9]+)$ ]]; then
	echo "usage: task release VERSION=v2.X.Y (got '${VERSION}')" >&2
	exit 2
fi
NESTED_TAGS=("picture/chartpicture/${VERSION}" "spec/echarts/${VERSION}" "examples/${VERSION}" "examples/shaders/${VERSION}")

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
OLD_ROOT=$(awk -v module="$MODULE" '{sub(/^require[ \t]+/, "")} $1 == module {print $2}' examples/go.mod)
OLD_CHART=$(awk -v module="$CHART_MODULE" '{sub(/^require[ \t]+/, "")} $1 == module {print $2}' examples/go.mod)
OLD_ECHARTS=$(awk -v module="$ECHARTS_MODULE" '{sub(/^require[ \t]+/, "")} $1 == module {print $2}' examples/go.mod)
for dir in cmd examples examples/shaders picture/chartpicture spec/echarts; do
	(cd "$dir" && go mod edit -require="${MODULE}@${VERSION}")
done
(cd examples && go mod edit -require="${CHART_MODULE}@${VERSION}" -require="${ECHARTS_MODULE}@${VERSION}")
for workfile in go.work wasm.work; do
	GOWORK="$PWD/$workfile" go work edit \
		-dropreplace="${MODULE}@${OLD_ROOT}" \
		-dropreplace="${CHART_MODULE}@${OLD_CHART}" \
		-dropreplace="${ECHARTS_MODULE}@${OLD_ECHARTS}" \
		-replace="${MODULE}@${VERSION}=." \
		-replace="${CHART_MODULE}@${VERSION}=./picture/chartpicture" \
		-replace="${ECHARTS_MODULE}@${VERSION}=./spec/echarts"
done
task go-tidy

DATE=$(date +%Y-%m-%d)
sed -i.bak "s|^## ${VERSION} (unreleased)|## ${VERSION} (${DATE})|" CHANGELOG.md
rm -f CHANGELOG.md.bak

echo "release: running tests"
task test
echo "release: verifying consumer downloads and builds in a scratch clone"
./scripts/check-release.sh "$VERSION" --write-sums

git add -A
git commit -q -s -m "chore(release): ${VERSION}"
git tag "$VERSION"
for tag in "${NESTED_TAGS[@]}"; do
	git tag "$tag"
done

echo
echo "Tagged ${VERSION} ${NESTED_TAGS[*]} on $(git rev-parse --short HEAD)."
echo "Review the commit, then publish with:"
echo "  git push --atomic origin $(git branch --show-current) ${VERSION} ${NESTED_TAGS[*]}"
