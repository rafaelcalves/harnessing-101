# Security policy

## Supported versions

Before the first release, security fixes target the current default branch. Release-specific support will be documented when versioned releases exist. Do not infer support from an old branch or tag.

## Report a vulnerability privately

Send vulnerability reports to **rafael.ca.dev@gmail.com**.

Do not open a public issue, discussion, or pull request for an undisclosed vulnerability. Do not include credentials, tokens, private agent messages, personal workspace contents, or unnecessary production data in a report.

Include what you can safely provide:

- the affected version, commit, operating system, and architecture;
- the security boundary or asset at risk;
- steps to reproduce with minimal test data;
- observed and expected behavior;
- likely impact and required user interaction;
- relevant logs with secrets and personal data removed; and
- any suggested mitigation or fix.

## What happens next

Maintainers will confirm receipt when the reporting route is available, assess scope and severity, and coordinate validation and remediation with the reporter when practical. No response or fix deadline is promised before the project has a published security-response capacity.

The project will avoid public detail that makes exploitation easier before a fix or mitigation is available. When disclosure is appropriate, the record should credit the reporter if requested and explain affected versions, impact, remediation, and remaining limits.

## Security-sensitive areas

Reports are especially useful for unexpected network egress, command or argument injection, unsafe child-process handling, path traversal, symlink or permission errors, workspace boundary escapes, malicious message or state files, secret exposure in logs, dependency compromise, and recovery behavior that can corrupt or misattribute state.

Harnessing 101 coordinates local files and user-provided agent processes. “Local-first” does not make untrusted files, process output, or configured agent connections safe by default. Product-initiated network access is a defect unless a reviewed feature explicitly requires visible, opt-in behavior. Egress by an agent the user configured is a separate trust boundary and must remain visible.
