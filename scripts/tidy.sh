#!/usr/bin/env bash
# Tidy against sibling source, including versions that are not published yet.
# Replacements exist only in temporary modfiles, never in published go.mod files.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
ROOT=$PWD
SCRATCH=$(mktemp -d "${TMPDIR:-/tmp}/ntcharts-tidy.XXXXXX")
trap 'rm -rf "$SCRATCH"' EXIT
for dir in . picture/chartpicture spec/echarts examples examples/shaders cmd; do
	(
		cd "$ROOT/$dir"
		cp go.mod "$SCRATCH/go.mod"
		cp go.sum "$SCRATCH/go.sum"
		export GOWORK=off
		go mod edit -modfile="$SCRATCH/go.mod" \
			-replace="github.com/NimbleMarkets/ntcharts/v2=$ROOT" \
			-replace="github.com/NimbleMarkets/ntcharts/picture/chartpicture/v2=$ROOT/picture/chartpicture" \
			-replace="github.com/NimbleMarkets/ntcharts/spec/echarts/v2=$ROOT/spec/echarts"
		go mod tidy -modfile="$SCRATCH/go.mod"
		go mod edit -modfile="$SCRATCH/go.mod" \
			-dropreplace=github.com/NimbleMarkets/ntcharts/v2 \
			-dropreplace=github.com/NimbleMarkets/ntcharts/picture/chartpicture/v2 \
			-dropreplace=github.com/NimbleMarkets/ntcharts/spec/echarts/v2
		# Local replacements make tidy discard sibling checksums. Retain both
		# hashes for versions still required; never invent hashes of local source.
		awk '
			NR == FNR {
				sub(/^require[ \t]+/, "")
				if ($1 ~ /^github.com\/NimbleMarkets\/ntcharts\// && $2 ~ /^v/) {
					keep[$1 " " $2] = 1
					keep[$1 " " $2 "/go.mod"] = 1
				}
				next
			}
			($1 " " $2) in keep
		' "$SCRATCH/go.mod" go.sum >> "$SCRATCH/go.sum"
		# Keep the go command's own ordering so go.sum does not churn between
		# this script and go commands that rewrite the file.
		go run "$ROOT/scripts/sortsum.go" "$SCRATCH/go.sum"
		cp "$SCRATCH/go.mod" go.mod
		cp "$SCRATCH/go.sum" go.sum
	)
done
# No `go work sync` here on purpose. Between releases the nested modules
# require the root at the upcoming, not-yet-tagged version; go1.27 makes
# `go work sync` fetch that version's go.mod from the proxy even though the
# module is `use`d and replaced in go.work, and fails with "version
# constraints conflict ... unknown revision" (go1.26 tolerated it). The
# per-module tidy above, with sibling replacements, already leaves every
# go.mod consistent; scripts/test-release.sh proves the flow end to end.
