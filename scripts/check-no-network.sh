#!/usr/bin/env bash
set -euo pipefail

# H101-44 checks the production dependency graph. `go list -deps ./...`
# excludes test-only imports, so a test helper may use a network package
# without becoming a product dependency; production packages must not. A
# future policy that bans network imports in tests too should use a separate
# `go list -deps -test` check rather than silently changing this scope.

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
forbidden=0
while IFS= read -r dependency; do
	case "$dependency" in
		net|net/http)
			echo "forbidden production network dependency: $dependency" >&2
			forbidden=1
			;;
	esac
done < <(cd "$ROOT" && go list -deps -f '{{.ImportPath}}' ./...)

if ((forbidden != 0)); then
	exit 1
fi
