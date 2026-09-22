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

Reproduction: 0/3 isolated runs of
`TestCLI_ServeOwnedGracefulCaptureCompletesBothChannels` on the corrected tree
(all three passed). The reported full-suite occurrence remains one known
failure from Kevin; a reliable local reproduction was not obtained.
