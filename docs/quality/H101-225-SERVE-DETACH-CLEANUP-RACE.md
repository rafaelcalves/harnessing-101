# H101-225 — serve detach cleanup diagnosis

The reported `directory not empty` failure is a distinct defect from
H101-156. H101-156 was a pre-open readiness/lock-acquisition race: the serve
process announced `listening` before it owned the workspace lock. This failure
occurs later, while an attached client is detaching and cleaning up its session
directory; it has a different test, phase, and error signature.

The cleanup path is `transport.Client.Detach`: it waits for the host's detach
response, then calls `os.RemoveAll` on the session directory. The host's
single poll loop answers detach and drops its bound session, but the observed
failure means cleanup ordering or another session-directory writer needs a
separate investigation. No production fix is included in this diagnosis.

Reproduction: the requested `go test ./cmd/harnessing/ -count=5` reproduced a
related detach-cleanup failure once in five package runs, in
`TestRun_ServeAttachRoundTripThenDetach` (the same `directory not empty`
signature). The earlier three isolated item-5 runs were all green. The
requested `go test ./... -count=2` did not complete as a clean reproduction
pass: it stopped on Kevin's in-progress
`TestSupervisor_StopOrderingEscalationAndDecoy` failure in
`internal/adapters/process/lifecycle_test.go`, not on this cleanup symptom.
