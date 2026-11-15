---
title: Velero S3 v4 scoped generator plan
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-10
---

Parent task: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`.
Tier L2; business/k8s acceptance tooling; physical product boundaries unchanged.
Generator owns only NEW ignored `.local/ops-acceptance/resume-velero-s3-v4-*`
and matching evidence here. No original mutation, delegation, remote execution,
credentials/configuration, backend/SQL/ledger/lock/commit/release changes.

Minimum complexity: reuse v3 provider, original fixture and actual driver,
original root validate function, existing pinned SSH helper, Python stdlib.

1. Derive remote v4 from v3: approve private 28333, require the three actual
   help flags, disable Iceberg/Lance and explicitly configure S3 gRPC.
2. Derive fixture solely by changing its provider import. Preserve finite SA,
   UID cleanup, actual RBAC and persisted S3/non-PV restore assertions.
3. Add root-only exact native inspection/cleanup and phased continuation.
   Bind prior receipts, originals, v4 inputs, request and actual live binary.
   Refuse foreign file/mount/identity/UID; never replay install before cleanup.
4. Test native/help/listener regression, AST preservation, exact metadata and
   mount refusal and cleanup with local fixtures. Parse/compile all new code.
5. Save fresh command/derivation/review evidence and exact phased root handoff.

Verification follows repo-verify (read `.agents/skills/repo-verify/SKILL.md`):
targeted Python tests and compilation for ignored acceptance scripts;
docs-frontmatter/task-packet-template checks for evidence linkage. No product
Go edits or commit/PR action; original driver binary integrity and prior actual
race-build evidence retained. Fresh real server acceptance is root-owned and
remains an explicit gap. Generator self-review is preparation, not P0 approval.
