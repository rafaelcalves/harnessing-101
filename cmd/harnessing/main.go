// Command harnessing is the CLI presentation adapter. Commands receive a
// caller-bound api.FrontendSession from the trusted assembly root. CLI flags,
// exit codes, and rendering stay here, outside the core, per
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
