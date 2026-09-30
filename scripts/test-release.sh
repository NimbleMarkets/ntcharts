#!/usr/bin/env bash
# Exercise release/tidy/CI behavior with tiny modules and real Git tags.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
export PATH="$(go env GOROOT)/bin:$PATH"
GO_VERSION=$(go env GOVERSION)
GO_VERSION=${GO_VERSION#go}
SCRATCH=$(mktemp -d "${TMPDIR:-/tmp}/ntcharts-release-test.XXXXXX")
SCRATCH=$(cd "$SCRATCH" && pwd -P)
trap 'chmod -R u+w "$SCRATCH"; rm -rf "$SCRATCH"' EXIT
REPO="$SCRATCH/repo"
MODULE=github.com/NimbleMarkets/ntcharts
# Fixture versions must never exist for real: go work sync reads the original
# version's go.mod from the module cache or proxy even when a workspace
# replaces it by directory, and verifies it against any go.sum entry present.
# A real tag with the fixture's number would make that check compare fake
# content against the published release.
RELEASE=v2.999.0
NEXT=v2.999.1
mkdir -p "$REPO/scripts" "$REPO/picture/chartpicture" "$REPO/spec/echarts" "$REPO/examples/quickstart" "$REPO/examples/shaders" "$REPO/cmd"
cp scripts/{release,check-release,tidy}.sh scripts/sortsum.go "$REPO/scripts/"
cp Taskfile.yml "$REPO/"
cd "$REPO"
export GOWORK="$REPO/go.work"
git init -q
git config user.name 'Release regression test'
git config user.email release-test@example.invalid
git config commit.gpgsign false
git config tag.gpgsign false
git config core.hooksPath /dev/null
printf '# Changelog\n\n## %s (unreleased)\n' "$RELEASE" > CHANGELOG.md
printf 'module %s/v2\n\ngo %s\n' "$MODULE" "$GO_VERSION" > go.mod
printf 'package ntcharts\nconst Name = "ntcharts"\n' > charts.go
for dir in picture/chartpicture spec/echarts examples examples/shaders cmd; do
	printf 'module %s/%s/v2\n\ngo %s\n\nrequire %s/v2 %s\n' \
		"$MODULE" "$dir" "$GO_VERSION" "$MODULE" "$RELEASE" > "$dir/go.mod"
	: > "$dir/go.sum"
done
printf '\nrequire %s/picture/chartpicture/v2 %s\n' "$MODULE" "$RELEASE" >> examples/go.mod
printf '\nrequire %s/spec/echarts/v2 %s\n' "$MODULE" "$RELEASE" >> examples/go.mod
printf 'package chartpicture\nimport "%s/v2"\nconst Name = ntcharts.Name\n' "$MODULE" > picture/chartpicture/chart.go
printf 'package echarts\nimport "%s/v2"\nconst Name = ntcharts.Name\n' "$MODULE" > spec/echarts/echarts.go
printf 'package main\nimport ("fmt"; "%s/v2"; "%s/picture/chartpicture/v2"; "%s/spec/echarts/v2")\nfunc main() { fmt.Println(ntcharts.Name, chartpicture.Name, echarts.Name) }\n' \
	"$MODULE" "$MODULE" "$MODULE" > examples/quickstart/main.go
for dir in examples/shaders cmd; do
	printf 'package main\nimport ("fmt"; "%s/v2")\nfunc main() { fmt.Println(ntcharts.Name) }\n' "$MODULE" > "$dir/main.go"
done
: > go.sum
cat > go.work <<EOF
go $GO_VERSION
use (
 .
 ./picture/chartpicture
 ./spec/echarts
 ./examples
 ./examples/shaders
 ./cmd
)
replace $MODULE/v2 $RELEASE => .
replace $MODULE/picture/chartpicture/v2 $RELEASE => ./picture/chartpicture
replace $MODULE/spec/echarts/v2 $RELEASE => ./spec/echarts
EOF
cp go.work wasm.work
git add -A
git commit -qm 'Initial fixture'

# go.sum must keep the go command's order (path, semantic version, then the
# /go.mod suffix), or tidy and go commands flip lines back and forth.
printf '%s\n' \
	'example.com/m v1.10.0/go.mod h1:b' \
	'example.com/m v1.9.0 h1:a' \
	'example.com/p v1.0.0 h1:x' \
	'example.com/m v1.10.0 h1:b' \
	'example.com/a v0.0.0-20200101000000-abcdef123456/go.mod h1:c' \
	'example.com/m v1.9.0/go.mod h1:a' \
	'example.com/p v1.0.0-rc.1 h1:y' \
	'example.com/m v1.9.0 h1:a' \
	> "$SCRATCH/scrambled.sum"
printf '%s\n' \
	'example.com/a v0.0.0-20200101000000-abcdef123456/go.mod h1:c' \
	'example.com/m v1.9.0 h1:a' \
	'example.com/m v1.9.0/go.mod h1:a' \
	'example.com/m v1.10.0 h1:b' \
	'example.com/m v1.10.0/go.mod h1:b' \
	'example.com/p v1.0.0-rc.1 h1:y' \
	'example.com/p v1.0.0 h1:x' \
	> "$SCRATCH/expected.sum"
go run scripts/sortsum.go "$SCRATCH/scrambled.sum"
if ! cmp -s "$SCRATCH/scrambled.sum" "$SCRATCH/expected.sum"; then
	echo 'FAIL: sortsum does not produce the go command order' >&2
	diff "$SCRATCH/expected.sum" "$SCRATCH/scrambled.sum" >&2 || true
	exit 1
fi
echo 'PASS: sortsum orders go.sum like the go command'

check() {
	local log=$1
	shift
	if ! "$@" > "$SCRATCH/$log" 2>&1; then
		cat "$SCRATCH/$log" >&2
		return 1
	fi
}
check release.log ./scripts/release.sh $RELEASE
for dir in picture/chartpicture spec/echarts examples examples/shaders cmd; do
	[[ $(grep -c "^$MODULE/v2 $RELEASE" "$dir/go.sum") == 2 ]]
	cp "$dir/go.sum" "$SCRATCH/${dir//\//_}.sum"
done
[[ $(grep -c "^$MODULE/picture/chartpicture/v2 $RELEASE" examples/go.sum) == 2 ]]
[[ $(grep -c "^$MODULE/spec/echarts/v2 $RELEASE" examples/go.sum) == 2 ]]
HEAD_COMMIT=$(git rev-parse HEAD)
for prefix in '' picture/chartpicture/ spec/echarts/ examples/ examples/shaders/; do
	[[ $(git rev-parse "${prefix}$RELEASE") == "$HEAD_COMMIT" ]]
done
# Fetch the actual final tags with a new cache: preparation must not have
# recorded hashes for intermediate module contents.
check published.log ./scripts/check-release.sh $RELEASE
echo 'PASS: final release tags contain valid sibling hashes and standalone modules'

printf '# Changelog\n\n## %s (unreleased)\n\n## %s (released)\n' "$NEXT" "$RELEASE" > CHANGELOG.md
check tidy.log ./scripts/tidy.sh
for dir in picture/chartpicture spec/echarts examples examples/shaders cmd; do
	cmp "$dir/go.sum" "$SCRATCH/${dir//\//_}.sum"
done
git add CHANGELOG.md
git commit -qm "Open $NEXT, keep dependencies pinned to $RELEASE"
check next.log ./scripts/check-release.sh
echo 'PASS: tidy retains published sums; next changelog version resolves previous tags'

# A check must fail on missing historical sums, not silently repair them.
awk -v prefix="$MODULE/" 'index($1, prefix) != 1' examples/go.sum > "$SCRATCH/missing.sum"
cp "$SCRATCH/missing.sum" examples/go.sum
if ./scripts/check-release.sh > "$SCRATCH/missing.log" 2>&1; then
	echo 'FAIL: missing historical checksums were not rejected' >&2
	exit 1
fi
grep -q 'missing go.sum entry' "$SCRATCH/missing.log"
cp "$SCRATCH/examples.sum" examples/go.sum
echo 'PASS: readonly checks reject missing historical checksums'

git clone --quiet --depth=1 --no-tags "file://$REPO" "$SCRATCH/ci"
cd "$SCRATCH/ci"
export GOWORK="$SCRATCH/ci/go.work"
if ./scripts/check-release.sh > "$SCRATCH/shallow.log" 2>&1; then
	echo 'FAIL: tag-less fixture unexpectedly resolved the previous release' >&2
	exit 1
fi
grep -Eq 'unknown revision|invalid version' "$SCRATCH/shallow.log"
git fetch --quiet --unshallow --tags origin
check complete.log ./scripts/check-release.sh
echo 'PASS: full history and tags fix the CI checkout regression'
