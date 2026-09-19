## Outcome

What user or contributor outcome does this change produce?

Closes:

## Change summary

-

## Evidence

Commands run and relevant results:

```text
go build ./...
go test ./...
go vet ./...
golangci-lint run ./...
test -z "$(gofmt -l .)"
```

Add focused test cases, screenshots, or local artifact paths when they materially help review. Do not include secrets, personal workspace data, or agent transcripts.

## Architecture and compatibility

- ADR required or linked:
- Public contract or stored-data impact:
- Recovery or migration impact:
- Supported-platform impact:

## Local-only and security review

- Product-initiated network access added or changed:
- User-configured agent egress added or changed:
- New dependency, process, listener, telemetry, or update behavior:
- Security-sensitive input or output affected:

Use `None` only after checking the change. Explain every non-none answer.

## Checklist

- [ ] The issue and acceptance criteria are linked.
- [ ] The change is focused and avoids unrelated cleanup.
- [ ] Tests cover the new behavior or regression.
- [ ] Build, tests, vet, lint, and formatting checks pass.
- [ ] Contributor and user documentation is updated where needed.
- [ ] Accepted architecture decisions remain intact, or a superseding ADR is linked.
- [ ] Local-only behavior remains intact, or the visible opt-in exception is documented and reviewed.
- [ ] Repository-local author identity is configured and the commit hook passes.
