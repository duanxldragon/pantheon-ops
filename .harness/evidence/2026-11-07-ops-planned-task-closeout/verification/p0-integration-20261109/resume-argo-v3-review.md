Parent task: `2026-11-07-ops-planned-task-closeout`; L2; Ops business domain only. Self-review; root owns independent CODE/ARCH review.

Findings: no additional blocker found in the scoped source diff. Real provider acceptance remains unverified in this lane; no final P0 approval.

Evidence: v2 trace contains exactly application GET and manifests GET, both 200; manifests revision missing. The next source guard rejects the empty decoded revision before managed-resources/diff. Missing namespace is normal and already defaulted by the existing explicit Group/Kind allowlist to the authorized matching application destination. The regression reproduces this shape before repair.

Change: an omitted response revision may use only the full immutable Git hash already sent to the renderer. A mutable ref without an explicit resolved hash, malformed returned revision, or mismatched immutable/rollback revision fails closed. No local branch resolution, extra provider request, dependency or public debug field was added.

Retained constraints: current/proposed authorization, server/repository availability and fingerprints, same actor/encrypted review, private manifest drift hash, provider UID/RV/spec conditional PATCH, unsupported Group/Kind denial, authority/credential/session revocation and sanitized errors/projections. Existing source tests remain unchanged. New real-shape tests exercise source default, SHA-256, rollback, mutable-ref rejection, namespace/group/kind denial, hidden values, actor/authority/repository/namespace/manifest drift, and unchanged reviewed submission.

Driver: new ignored versioned runner and remote script reuse the existing finite project credential, verified TLS/pinned SSH relay, four-operation protocol, UID/label/path cleanup, bounded cache and retained 960 MiB provider. Public records are allowlisted; fixture config/token/provider bodies stay in private memory/RPC. The runner checks its own, remote script and module hashes before remote access. Original v2 artifacts are retained.

Residuals: mutable-reference resolution and full Base session/IAM/audit/public-origin wiring are not implemented; real v3 provider phase and original O3/D1/P0 acceptance remain open. Full-tree/overlay/rebuild/UI/migration/independent review remain coordinating-root work. No source outside Argo, SQL/shared/UI/lock/ledger, commit/push/release or remote operation performed here.
