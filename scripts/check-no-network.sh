#!/usr/bin/env bash
set -euo pipefail

# H101-44 checks the production dependency graph. `go list -deps ./...`
# excludes test-only imports, so a test helper may use a network package
# without becoming a product dependency; production packages must not. A
# future policy that bans network imports in tests too should use a separate
# `go list -deps -test` check rather than silently changing this scope.
#
# This is an import-graph gate, not proof that production code cannot reach the
# network. It rejects the literal import paths net and net/http; it cannot see
# raw socket syscalls or an os/exec child such as curl when those paths are not
# imported. Creed's H101-227 security review records this limit. Any admission
# or version bump of a syscall-capable dependency requires a manual review of
# its syscall behavior; a pin must not become a semver-trusted auto-update.

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
