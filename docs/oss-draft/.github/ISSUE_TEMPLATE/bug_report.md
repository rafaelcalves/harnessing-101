---
name: Bug report
about: Report reproducible behavior that differs from the documented contract
title: "[Bug]: "
labels: bug
assignees: ""
---

## Summary

Describe the failed action and its effect in one or two sentences.

## Environment

- Harnessing 101 version or commit:
- Operating system and architecture:
- Go version, if running from source:
- Interface used:

## Steps to reproduce

1.
2.
3.

Use a minimal local workspace. Remove credentials, personal data, agent transcripts, and confidential file contents.

## Expected behavior

What should have happened, based on which command or document?

## Actual behavior

What happened? State whether data was retained, changed, duplicated, or lost.

## Evidence

Provide the smallest relevant log excerpt, error text, or test case. Redact secrets and private workspace data.

## Frequency and recovery

- Does it happen every time?
- Can the user recover without editing stored files?
- Does restarting change the result?

## Safety checks

- [ ] This report contains no vulnerability that should be sent through `SECURITY.md`.
- [ ] This report contains no credentials or private workspace content.
- [ ] I checked for an existing issue describing the same behavior.
