# ADR-0009 - Per-team instances: mock provisioner by default

**Status:** accepted for M8

The instance lifecycle API (create, get, extend, destroy; one running instance
per team per challenge; global cap; TTL) and per-team dynamic flags
(`{{team_flag}}`, HMAC per event/team/challenge) are implemented and tested.

Actual container provisioning ships as a **mock** by default: it returns a
deterministic host and port without touching Docker. Rationale: real provisioning
requires the server to hold Docker/Kubernetes privileges and per-instance network
isolation for an audience that will actively try to escape (spec 5.8). Granting
that by default is the wrong posture. A real provisioner is the opt-in follow-up
(mount the host Docker socket or a K8s client, pull challenge images from a
registry, enforce per-instance network namespaces) behind the same interface;
the API contract and the anti-flag-sharing dynamic flags do not change.
