# ADR-0003 — Plugins out-of-process

A plugin is a separate HTTP service that talks to the core via signed webhooks
and a narrow token-scoped API. Go's in-process `plugin` is fragile and
Linux-only, a plugin must not be able to crash the core, and plugins may be
written in any language — important because the King of the Hill project
already exists and will be **wrapped**, not rewritten. See `INSTRUKCJE.md`
ADR-003 and section 7.
