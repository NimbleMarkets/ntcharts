#!/usr/bin/env bash
# Tidy against sibling source, including versions that are not published yet.
# Replacements exist only in temporary modfiles, never in published go.mod files.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
ROOT=$PWD
SCRATCH=$(mktemp -d "${TMPDIR:-/tmp}/ntcharts-tidy.XXXXXX")
trap 'rm -rf "$SCRATCH"' EXIT
for dir in . picture/chartpicture examples examples/shaders cmd; do
	(
		cd "$ROOT/$dir"
		cp go.mod "$SCRATCH/go.mod"
		cp go.sum "$SCRATCH/go.sum"
		export GOWORK=off
		go mod edit -modfile="$SCRATCH/go.mod" \
			-replace="github.com/NimbleMarkets/ntcharts/v2=$ROOT" \
			-replace="github.com/NimbleMarkets/ntcharts/picture/chartpicture/v2=$ROOT/picture/chartpicture"
		go mod tidy -modfile="$SCRATCH/go.mod"
		go mod edit -modfile="$SCRATCH/go.mod" \
			-dropreplace=github.com/NimbleMarkets/ntcharts/v2 \
			-dropreplace=github.com/NimbleMarkets/ntcharts/picture/chartpicture/v2
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
		LC_ALL=C sort -u "$SCRATCH/go.sum" > "$SCRATCH/sorted.sum"
		cp "$SCRATCH/go.mod" go.mod
		cp "$SCRATCH/sorted.sum" go.sum
	)
done
go work sync
