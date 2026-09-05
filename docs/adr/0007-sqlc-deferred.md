# ADR-0007 — Data access via hand-written pgx (sqlc deferred to M2)

**Status:** accepted for M1; revisit at M2

Spec section 3 mandates `pgx/v5 + sqlc`. For the first working slice (M1) the data
layer in `internal/store` uses hand-written pgx queries instead of generated sqlc
code. Rationale: while the schema and query surface are still moving, hand-written
SQL keeps velocity and gives full control; the queries are explicit and N+1-free
(the stated goal of choosing sqlc). UUIDs cross the Go boundary as strings and are
cast at the SQL edge (`::uuid`) to avoid type-mapping ceremony.

**Plan:** adopt sqlc in **M2** (the challenge-library / data milestone), once the
schema stabilizes — generate type-safe code from `db/queries`, and swap the store
internals without changing the package's public API. This is a documented deviation
per spec section 0.5.
