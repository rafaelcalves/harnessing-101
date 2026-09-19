// Command harnessing is the CLI adapter. It is a thin wrapper over
// internal/host.Capabilities: every command constructs Capabilities via
// host.Open and calls only its exported methods. CLI flags, exit codes,
// and rendering stay here, outside the core, per
// docs/architecture/boundaries.md.
package main

import (
	"os"
)

// version is set at build time via -ldflags; "dev" is the unreleased default.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
