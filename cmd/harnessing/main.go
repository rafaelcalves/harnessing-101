// Command harnessing is the CLI adapter entry point. Phase 0 provides a
// version stub only; no other command exists yet. See
// docs/architecture/boundaries.md — CLI flags, exit codes, and rendering
// stay outside the core.
package main

import (
	"fmt"
	"os"
)

// version is set at build time via -ldflags; "dev" is the unreleased default.
var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version") {
		fmt.Println("harnessing " + version)
		return
	}
	fmt.Fprintln(os.Stderr, "harnessing: no commands are implemented yet (Phase 0 skeleton)")
	os.Exit(1)
}
