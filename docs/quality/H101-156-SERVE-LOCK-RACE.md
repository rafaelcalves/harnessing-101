# H101-156 — serve lock race diagnosis

The race was in the product readiness signal, not in the workspace lock or
the test. `harnessing serve` printed its `listening` line before
`assembly.Serve` opened the workspace and acquired the exclusive lock. The
native test correctly treated that line as its synchronization point, so a
second `host.Open` could win the interval before the serve process acquired
the lock.

The fix keeps the test's synchronization contract and moves the announcement
behind transport host readiness: the host writes its marker only after the
workspace has been opened, then invokes an assembly callback that prints the
line. The lock remains unchanged and the test is not loosened.
