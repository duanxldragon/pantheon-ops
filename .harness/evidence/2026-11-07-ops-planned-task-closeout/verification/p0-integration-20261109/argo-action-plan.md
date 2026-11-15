# Argo action repair scope — 2026-11-09

Business-only L2 repair under O3/D1: preserve request cancellation, return a sanitized provider rejection, represent HTTP acceptance as Running without a completion timestamp, and roll back by provider-owned historical deployment ID. Existing public revision input and webhook wrapper remain compatible. No SSRF/TLS relaxation, dependency, shared source override, schema change or release-lock change.

Before/after verification uses isolated real MySQL and HTTP provider fixtures. Add protocol tests for history identity, cancellation and body redaction; run scoped race and full stable CGO race after backend edits. Real Argo installation, resource diff, terminal operation/history matching, namespace/DataScope policy and independent review remain open criteria.
