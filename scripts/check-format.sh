#!/usr/bin/env bash
# Include new source files, but leave ignored build artifacts alone.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
failed=false
while IFS= read -r -d '' file; do
	[[ -f "$file" ]] || continue
	unformatted=$(gofmt -l "$file")
	if [[ -n "$unformatted" ]]; then
		echo "$file needs gofmt" >&2
		failed=true
	fi
done < <(git ls-files --cached --others --exclude-standard -z -- '*.go')
if [[ "$failed" == true ]]; then
	exit 1
fi
