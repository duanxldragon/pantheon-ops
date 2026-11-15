---
title: V7 Generator scope and evidence review
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-10
---

This is a Generator scope check, not CODE/ARCH approval. No independent reviewer
was delegated because this lane expressly prohibits delegation. Root must inspect
the final sources/request and all actual remote receipts.

The final source comparison was inspected against pinned v6 root/remote sources.
The only absence compatibility allowance is the reviewed BSL identity with its
manifest CRD association, live CRD NotFound, successful authoritative group-absent
discovery and exact unregistered GET error. Ordinary errors fail closed. Complete
performs the fresh check before migration writes and before remaining CREATEs.
Original tail AST and CRD Established wait are covered by executable tests.

V6 native and fixture behavior is imported from exact predecessor bytes. The
bounded loader redirects explicit predecessor imports and protected file reads
to capture/check/consume the same bytes; runtime/native/private reads retain their
existing semantics. Original validation/cleanup bodies and driver source/binary
seals are retained. New remote helper commands independently capture, hash and
compile the exact requested helper bytes. No private K8s fallback, auth widening,
installer replay, provider cleanup phase or namespace adoption was added.

Fresh verification in `resume-velero-s3-v7-commands.json`: 45 v7 tests plus
46/32/25 unchanged predecessor tests, all 148 passing without skips; seven helper
syntax compilations; local root input check; 180 protected hashes; 17 original
product/driver source seals. Tests cover missing/wrong proofs, all key errors,
existing CR/CRD, changed discovery/schema, exact five/mirror identities, failed or
wrong-hash predecessors, unknown remote files, changed remote helper hashes,
extra progress bytes, captured helper bytes, Established failure, and last-check
drift refusing CREATEs.

The first capture test run refused on Windows because lstat/fstat expose different
ctime semantics. V7 now compares ctime on POSIX and inode/size/mtime/mode/link
identity on both platforms, together with SHA256 of the consumed buffer. The
initial local v7 request and preparation remain immutable; final request is a
separate path, hash `05a39bef4df96a8c08ad78b17373d4968b6116be8ee8a2682e7708147517ad6e`.
No old source/request/failure bytes were rewritten or reapproved.

All Kubernetes/native/S3/SQL activity in tests is mocked. No remote phase or
credential operation ran. Real controller/BSL Available/Backup/Restore/S3 retention
and fixture cleanup are unverified here. No Go/frontend product changes or UI
work occurred; product build/race/visual evidence is outside this local preparation.
CodeGraph was not exposed; the task supplied exact source paths. Root remains the
review and execution path; no P0 closure or acceptance waiver is asserted.
