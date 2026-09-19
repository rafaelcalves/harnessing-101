#!/usr/bin/env bash
set -euo pipefail

# H101-19's feasible Phase 1 rule. This is package import containment, not
# resolved-symbol analysis. The latter requires a future go/parser + go/ast
# checker; do not silently call this script a Commit call-site checker.

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
CONFIG="${IMPORT_ALLOWLIST_CONFIG:-$ROOT/docs/architecture/import-allowlist.txt}"
MODULE=$(sed -n 's/^module[[:space:]]*//p' "$ROOT/go.mod" | head -n 1)

if [[ -z "$MODULE" ]]; then
	echo "import allowlist: module path missing from go.mod" >&2
	exit 2
fi

allowed_store() {
	awk -v wanted="$1" '
		/^# Production packages allowed to import the concrete state store:/ { section=1; next }
		/^# Production packages allowed to import the ports package/ { section=0 }
		section && $1 == wanted { found=1 }
		END { exit(found ? 0 : 1) }
	' "$CONFIG"
}

allowed_ports() {
	awk -v wanted="$1" '
		/^# Production packages allowed to import the ports package/ { section=1; next }
		section && $1 == wanted { found=1 }
		END { exit(found ? 0 : 1) }
	' "$CONFIG"
}

check_allowlist_paths() {
	bad=0
	while IFS= read -r package_path; do
		[[ -z "$package_path" ]] && continue
		case "$package_path" in
			"$MODULE"/*) relative=${package_path#"$MODULE"/} ;;
			*)
				echo "import allowlist: invalid package path $package_path" >&2
				bad=1
				continue
				;;
		esac
		if [[ ! -d "$ROOT/$relative" ]]; then
			echo "import allowlist: package does not exist: $package_path" >&2
			bad=1
		fi
	done < <(awk '/^github.com\// { print $1 }' "$CONFIG")
	return "$bad"
}

state_store="$MODULE/internal/adapters/statestore"
ports="$MODULE/internal/core/ports"
if ! check_allowlist_paths; then
	exit 2
fi
targets=("./...")
if (($# > 0)); then
	targets=("$@")
fi

violations=0
while IFS= read -r row; do
	[[ -z "$row" ]] && continue
	pkg=${row%% *}
	imports=${row#* }
	for imported in $imports; do
		imported=${imported#[}
		imported=${imported%,}
		imported=${imported%]}
		if [[ "$imported" == "$state_store" ]] && ! allowed_store "$pkg"; then
			echo "forbidden state-store import: $pkg -> $imported" >&2
			violations=1
		fi
		if [[ "$imported" == "$ports" ]] && ! allowed_ports "$pkg"; then
			echo "forbidden outbound-ports import: $pkg -> $imported" >&2
			violations=1
		fi
	done
done < <(cd "$ROOT" && go list -f '{{.ImportPath}} {{.Imports}}' "${targets[@]}")

if ((violations != 0)); then
	exit 1
fi
