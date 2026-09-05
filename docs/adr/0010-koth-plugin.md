# ADR-0010 - King of the Hill as a reference plugin

**Status:** accepted for M8

`cmd/koth-plugin` is a reference plugin that speaks Plugin API v1 (spec 7): it
serves a manifest, verifies signed webhooks (HMAC over `timestamp.body`),
deduplicates deliveries, and on `tick.minute` awards points to the current crown
holder through the core awards endpoint (idempotent by `ref_id`). It passes
`reduta-cli plugin verify`.

Wrapping the client's existing King of the Hill project (its agent reads the
holder from `/root/king.txt` on each target) means pointing that holder detection
at this plugin and setting `KOTH_CORE_URL`, `KOTH_PLUGIN_TOKEN` and `KOTH_EVENT_ID`.
The existing repository is wrapped, not rewritten; its URL is the open item from
INSTRUKCJE section 14.1.
