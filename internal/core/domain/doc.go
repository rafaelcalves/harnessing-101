// Package domain holds the core's own vocabulary: opaque identifiers and the
// record shapes that cross a port boundary. It has no dependency on any UI
// framework, command parser, operating-system process object, filesystem
// watcher, or network client. See docs/architecture/boundaries.md.
//
// Phase 0: this package defines shapes only. No transition rules, execution
// policy, or persistence behaviour lives here yet.
package domain
