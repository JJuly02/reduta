# ADR-0006 — License

**Status:** ACCEPTED — Apache-2.0 (2026-09-05)

Options considered:
- **Apache-2.0** — permissive with an explicit patent grant; easiest for third
  parties, companies, universities, and CTF organizers to adopt, embed, and build
  plugins against. No obligation to share modifications.
- **AGPL-3.0** — strong copyleft; a modified version offered as a network service
  must publish its source. Protects a commercial hosted offering, at the cost of
  adoption (many organizations disallow AGPL outright).
- **MIT** — simplest permissive, but without a patent grant.

Decision: **Apache-2.0**. The goal is wide adoption and a healthy plugin ecosystem
for a project meant to be downloaded and self-hosted, not a commercial SaaS whose
hosted forks need protecting. The AGPL network-copyleft rationale does not apply
here, and AGPL would deter exactly the adopters this project wants. Apache-2.0's
patent grant also makes it a safer choice than MIT. Plugins run out-of-process
(ADR-0003), so the license does not reach third-party plugin code regardless.

Note: Reduta is a clean-room implementation and shares no code with any existing
platform, so this choice is free and not inherited from another project's license.
