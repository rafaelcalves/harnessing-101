# Licence recommendation

Status: **Proposal for the project owner. No licence is granted by this document.**

## Recommendation

Adopt the **Apache License 2.0** when the owner is ready to accept outside contributions and make the repository public.

Apache-2.0 is permissive: companies may use, modify, distribute, and include the tool in proprietary systems. It also includes an explicit patent grant from contributors and a patent-termination clause. That combination fits a corporate-friendly project with outside contributors better than a shorter permissive licence whose patent terms are implicit or absent.

The copyright holder should be identified personally as **Rafael Correa Alves**. Accept contributions under the same licence without requiring copyright assignment unless the owner later obtains legal advice that justifies a contributor licence agreement.

## What it costs us

- Downstream users may create and distribute closed-source modifications. They do not have to publish improvements.
- Distributions must retain the licence and required notices. Modified files must carry prominent change notices, and any project `NOTICE` file must be handled according to the licence.
- Contributors grant recipients the patent rights necessary to use their contributions. The licence's patent rights terminate for a party that brings specified patent litigation.
- The licence provides no warranty and grants no trademark rights. Branding and name use need separate rules if that becomes necessary.
- The project must keep copyright and provenance records accurate as outside contributions arrive.

The main cost is strategic, not financial: the project gives up reciprocal source-sharing in exchange for easier adoption. The repository can use the standard licence text; no paid licence or per-user fee is required. Legal review may still be appropriate before public release, especially if an employer, sponsor, or substantial third-party code later creates ownership questions.

## Alternatives considered

### MIT License — permissive alternative

MIT is shorter and places minimal conditions on reuse. It would also support corporate adoption, but it does not provide Apache-2.0's express patent licence and patent-termination terms. With contributions expected from unrelated people, that missing clarity outweighs the benefit of a shorter text.

### GNU General Public License v3 — copyleft alternative

GPL-3.0 would require distributors of covered derivative works to provide corresponding source under the same licence. Private internal use does not by itself trigger source distribution. Its reciprocity protects access to distributed modifications, but it also creates more review and integration work for companies considering the tool or embedding it in a larger distribution.

GPL-3.0 loses here because corporate-friendly adoption is a stated goal and the product is a local tool, not a hosted service whose server modifications need a network-use copyleft clause. The project has not stated that forcing distributed derivatives to remain open is more important than adoption.

## Owner decision required

Accept or reject Apache-2.0 before a `LICENSE` file is added or outside contributions are invited. If accepted, create the repository-root `LICENSE` from the official Apache-2.0 text and review whether a `NOTICE` file is needed. This recommendation is project guidance, not legal advice.
