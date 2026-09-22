//go:build darwin

package process

import "golang.org/x/sys/unix"

// otherMembersAbsent is Darwin's half of H101-222/H101-224's membership
// predicate: does any process, other than excludePID, belong to the
// process group pgid? unix.SysctlKinfoProcSlice("kern.proc.pgrp", ...)
// asks the kernel directly for exactly that group's member list --
// this is real, native observation (docs/architecture/h101-226-native-membership-dependency.md),
// not a boolean liveness probe: it can distinguish the retained,
// terminated leader (still a member, still zombie) from an actually
// live other member, which is precisely what kill(-pgid,0) cannot do.
//
// A nil slice/error return of (true, nil) means the kernel reported
// zero OTHER members at this instant; any read error is uncertainty
// (false, err), never treated as "empty."
func otherMembersAbsent(pgid int, excludePID int) (absent bool, err error) {
	procs, err := unix.SysctlKinfoProcSlice("kern.proc.pgrp", pgid)
	if err != nil {
		return false, err
	}
	for _, p := range procs {
		if int(p.Proc.P_pid) == excludePID {
			continue
		}
		return false, nil
	}
	return true, nil
}
