//go:build linux

package process

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// otherMembersAbsent is Linux's half of H101-222/H101-224's membership
// predicate: does any process, other than excludePID, belong to the
// process group pgid? Scans /proc for every numeric entry, reads each
// candidate's /proc/<pid>/stat, and compares its pgrp field (stat's
// own field 5) to pgid -- adapter-local, stdlib only, no dependency
// needed on this target (docs/architecture/h101-226-native-membership-dependency.md).
//
// A candidate that disappears between the directory listing and the
// stat read (ENOENT) already exited -- correctly not a member. Any
// OTHER read/parse failure is uncertainty, never treated as "empty":
// H101-222 point 3, "a partial enumeration or unreadable process is
// uncertainty, not an empty group."
func otherMembersAbsent(pgid int, excludePID int) (absent bool, err error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, convErr := strconv.Atoi(e.Name())
		if convErr != nil {
			continue // not a pid directory (e.g. "self", "net")
		}
		if pid == excludePID {
			continue
		}
		data, readErr := os.ReadFile("/proc/" + e.Name() + "/stat")
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue // exited between listing and read -- not a member
			}
			return false, readErr
		}
		pgrp, ok := parseStatPgrp(data)
		if !ok {
			return false, errors.New("unparseable /proc/" + e.Name() + "/stat")
		}
		if pgrp == pgid {
			return false, nil
		}
	}
	return true, nil
}

// parseStatPgrp extracts field 5 (pgrp) from /proc/<pid>/stat's own
// format: "pid (comm) state ppid pgrp session ...". comm can itself
// contain spaces or parentheses, so this locates the LAST ')' rather
// than splitting the whole line on spaces from the start.
func parseStatPgrp(data []byte) (pgrp int, ok bool) {
	s := string(data)
	closeParen := strings.LastIndexByte(s, ')')
	if closeParen < 0 || closeParen+2 > len(s) {
		return 0, false
	}
	fields := strings.Fields(s[closeParen+2:])
	// After "pid (comm) ": state(0) ppid(1) pgrp(2) ...
	if len(fields) < 3 {
		return 0, false
	}
	pgrp, convErr := strconv.Atoi(fields[2])
	if convErr != nil {
		return 0, false
	}
	return pgrp, true
}
