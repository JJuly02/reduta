# ADR-0006 — License

**Status:** ACCEPTED — AGPL-3.0 (2026-09-05)

Options considered:
- **AGPL-3.0** — strong copyleft; a modified version offered as a network service
  must publish its source. Keeps hosted/SaaS forks open.
- **Apache-2.0 / MIT** — permissive; easier third-party adoption and embedding,
  no obligation to share modifications.

Decision: **AGPL-3.0**. The platform is meant to stay open even when run as a
hosted service by third parties, so the network-use copyleft of AGPL is the right
fit. The `LICENSE` file carries the full text.

Note: Reduta is a clean-room implementation and shares no code with any existing
platform, so this choice is free and not inherited from another project's license.
