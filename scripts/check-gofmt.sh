#!/usr/bin/env bash
set -euo pipefail

# H101-77 closes the gap that let an unformatted testdata fixture reach
# main (6a3d4fb): every agent on this floor has run `gofmt -l .` locally
# and reported it, but nothing enforced it, so the discipline was a habit
# rather than a guard. `gofmt -l` always exits 0 — it lists unformatted
# files on stdout but never fails on their account — so a bare `gofmt -l .`
# CI step would silently pass no matter what it lists. This script is the
# difference: it treats any listed file as a failure. It intentionally
# does not exclude testdata; that exclusion (implicit, via `go test ./...`
# never reaching a fixture outside the build) is exactly what let the
# original fixture go unformatted, and this check exists to cover the
# gap, not repeat it.

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
unformatted=$(cd "$ROOT" && gofmt -l .)

if [[ -n "$unformatted" ]]; then
	echo "gofmt found unformatted file(s):" >&2
	echo "$unformatted" >&2
	exit 1
fi
