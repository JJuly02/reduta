# ADR-0002 — Modular monolith, not microservices

One `reduta-server` process + `reduta-worker`. Module boundaries enforced by
package structure and an import linter (`depguard`), not the network.
Microservices at this scale are cost without benefit. See `INSTRUKCJE.md` ADR-002.
