#!/usr/bin/env bash
# Fetch a snapshot through Go's Git resolver with no workspace or replacements.
# All commits, tags, downloads and installed binaries stay in a temporary folder.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
VERSION="${1:-$(awk '/^## v2\./ { print $2; exit }' CHANGELOG.md)}"
WRITE_SUMS="${2:-}"
if [[ ! "$VERSION" =~ ^v2\.[0-9]+\.[0-9]+$ || ( -n "$WRITE_SUMS" && "$WRITE_SUMS" != --write-sums ) || $# -gt 2 ]]; then
	echo "usage: $0 [v2.X.Y [--write-sums]]" >&2
	exit 2
fi
ROOT=$PWD
NEW_RELEASE=true
if git rev-parse -q --verify "refs/tags/$VERSION" >/dev/null; then
	NEW_RELEASE=false
	if [[ -n "$WRITE_SUMS" ]]; then
		echo 'release check: cannot write new checksums for an existing release' >&2
		exit 1
	fi
fi
SCRATCH=$(mktemp -d "${TMPDIR:-/tmp}/ntcharts-release-check.XXXXXX")
trap 'chmod -R u+w "$SCRATCH"; rm -rf "$SCRATCH"' EXIT

git clone --quiet --no-hardlinks "$ROOT" "$SCRATCH/repo"
git diff --binary HEAD > "$SCRATCH/changes.patch"
if [[ -s "$SCRATCH/changes.patch" ]]; then
	git -C "$SCRATCH/repo" apply "$SCRATCH/changes.patch"
fi
while IFS= read -r -d '' file; do
	mkdir -p "$SCRATCH/repo/$(dirname "$file")"
	cp "$file" "$SCRATCH/repo/$file"
done < <(git ls-files --others --exclude-standard -z)
snapshot() {
	git -C "$SCRATCH/repo" add -A
	git -C "$SCRATCH/repo" -c user.name='Release check' -c user.email='release-check@example.invalid' \
		-c commit.gpgsign=false -c core.hooksPath=/dev/null commit --quiet --allow-empty -m 'Release check snapshot'
	if [[ "$NEW_RELEASE" == true ]]; then
		for prefix in '' picture/chartpicture/ spec/echarts/ examples/ examples/shaders/; do
			git -C "$SCRATCH/repo" -c tag.gpgsign=false tag -f "${prefix}${VERSION}" >/dev/null
		done
	fi
}
snapshot

# Reuse cached third-party downloads, but never cache these unpublished versions
# in the developer's normal module cache or contact GitHub for this repository.
CACHED_PROXY="file://$(go env GOMODCACHE)/cache/download"
# Keep the toolchain already selected by this checkout when leaving its workspace.
export PATH="$(go env GOROOT)/bin:$PATH"
export GOPROXY="$CACHED_PROXY,$(go env GOPROXY)"
export GOMODCACHE="$SCRATCH/root-cache" GOBIN="$SCRATCH/bin" GOWORK=off GOFLAGS=
export GONOPROXY=github.com/NimbleMarkets/ntcharts
export GONOSUMDB=github.com/NimbleMarkets/ntcharts
export GIT_CONFIG_COUNT=2
export GIT_CONFIG_KEY_0="url.file://$SCRATCH/repo.insteadOf"
export GIT_CONFIG_VALUE_0=https://github.com/NimbleMarkets/ntcharts
export GIT_CONFIG_KEY_1="$GIT_CONFIG_KEY_0"
export GIT_CONFIG_VALUE_1=https://github.com/NimbleMarkets/ntcharts.git

# Finalize sibling hashes in dependency order. Nested modules are excluded from
# the root module zip, so changing their sums cannot change the root checksum.
# Only candidate-version hashes may be regenerated. Historical release sums must
# already be present and are checked unchanged by the readonly builds below.
candidate_sums() {
	local dir=$1 module version
	while read -r module version; do
		[[ "$NEW_RELEASE" == true && "$version" == "$VERSION" ]] || continue
		(
			cd "$SCRATCH/repo/$dir"
			awk -v module="$module" -v version="$version" \
				'!($1 == module && ($2 == version || $2 == version "/go.mod"))' go.sum > "$SCRATCH/filtered.sum"
			cp "$SCRATCH/filtered.sum" go.sum
			go mod download "$module@$version"
		)
	done < <(awk '{sub(/^require[ \t]+/, "")} $1 ~ /^github.com\/NimbleMarkets\/ntcharts\// && $2 ~ /^v/ {print $1, $2}' "$SCRATCH/repo/$dir/go.mod")
}
candidate_sums picture/chartpicture
candidate_sums spec/echarts
snapshot
# Go caches Git tag resolutions as well as module zips. Use a fresh cache after
# finalizing chartpicture and spec/echarts so examples hash their updated
# go.sum files, not the first tag.
export GOMODCACHE="$SCRATCH/nested-cache"
for dir in examples examples/shaders cmd; do
	candidate_sums "$dir"
done
snapshot
# Verify the final tags independently of all intermediate snapshots.
export GOMODCACHE="$SCRATCH/modcache"

echo 'release check: standalone builds, verification, and vendoring (readonly module files)'
for dir in . picture/chartpicture spec/echarts examples examples/shaders cmd; do
	(
		cd "$SCRATCH/repo/$dir"
		case "$dir" in
			.|picture/chartpicture|spec/echarts) go build -mod=readonly ./... ;;
			*)
				mkdir -p "$SCRATCH/build/$dir"
				go build -mod=readonly -o "$SCRATCH/build/$dir/" ./...
				;;
		esac
		go mod verify
		if [[ "$dir" != . ]]; then
			go mod vendor -o "$SCRATCH/vendor/$dir"
		fi
	)
done
if [[ -n "$(git -C "$SCRATCH/repo" status --porcelain)" ]]; then
	echo 'release check: standalone commands changed module files' >&2
	git -C "$SCRATCH/repo" diff --stat >&2
	exit 1
fi

mkdir "$SCRATCH/consumer"
cd "$SCRATCH/consumer"
go mod init release-check.example/consumer
MODULE=github.com/NimbleMarkets/ntcharts
echo "release check: core library and dependency isolation"
go get "$MODULE/v2/...@$VERSION"
go list -m all > "$SCRATCH/core-modules"
if grep -Eq 'go-analyze/|go-echarts/|gogpu/|go-webgpu/|go-booba|ntcharts/(picture/chartpicture|spec/echarts|examples)' "$SCRATCH/core-modules"; then
	echo 'release check: optional dependencies leaked into the core module' >&2
	exit 1
fi
go build "$MODULE/v2/..."

echo "release check: chartpicture consumer"
go get "$MODULE/picture/chartpicture/v2@$VERSION"
go build "$MODULE/picture/chartpicture/v2/..."

echo "release check: spec/echarts consumer"
go get "$MODULE/spec/echarts/v2@$VERSION"
go build "$MODULE/spec/echarts/v2/..."

echo "release check: install all published demos at $VERSION"
go install "$MODULE/examples/v2/...@$VERSION"
go install "$MODULE/examples/shaders/v2@$VERSION"
echo "release check: run shader command without a terminal or GPU"
go run "$MODULE/examples/shaders/v2@$VERSION" -list

# @latest is what the gallery advertises. Check both module boundaries.
for module in "$MODULE/examples/v2" "$MODULE/examples/shaders/v2"; do
	latest=$(go list -m -f '{{.Version}}' "$module@latest")
	if [[ "$latest" != "$VERSION" ]]; then
		echo "release check: $module@latest selected $latest, expected $VERSION" >&2
		exit 1
	fi
done

echo "release check: internal tools build without the workspace"
(cd "$SCRATCH/repo/cmd" && go test -mod=readonly ./...)
if [[ -n "$WRITE_SUMS" ]]; then
	for dir in picture/chartpicture spec/echarts examples examples/shaders cmd; do
		cp "$SCRATCH/repo/$dir/go.sum" "$ROOT/$dir/go.sum"
	done
fi
echo "release check: passed (nothing published)"
