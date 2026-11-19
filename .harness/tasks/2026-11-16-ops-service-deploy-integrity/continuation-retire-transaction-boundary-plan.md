# C01 transaction implementation boundary repair

Parent: 2026-11-16-ops-service-deploy-integrity / G003-c01-c04-e01-e03. L2; business-only. Evidence remains in `.harness/evidence/2026-11-17-ops-history-audit/`.

The complete backend race passed against the 1221-source Retire isolation candidate (exit 0, 1011.134 seconds, 734 unchanged backend sources). The subsequent full business boundary checker rejected seven GORM exposures in `business/capability/transaction.go`. That contract directory must remain GORM-free. Preserve both successful verification and the later real boundary failure; the previous source binding does not certify a relocated candidate.

Minimum complexity: reuse the existing `business/shared` implementation-utility ownership and its recursive overlay asset declaration. Physically move the transaction helper and its tests to `backend/modules/business/shared/transaction`, package `transaction`. Change only the package declaration and necessary consumer import paths/call qualifiers (`biztx`). Do not add capability aliases, change/allowlist the checker, modify public DTO contracts, add an overlay path, or alter runtime transaction behavior.

Executor ownership: the two moved files and eight helper consumers in Service/Deploy. Preserve other agents' edits. Root owns evidence, source binding, fresh official rebuild, exact-source complete Windows MinGW/CGO race, preview readiness, task status and governance. Code-reviewer and architect independently compare helper/test bodies and consumers against the prior verified candidate and bind final hashes. No backend source edits overlap root's final verification.

Success: the real boundary checker passes; body equivalence apart from package/import qualifiers is established; fresh source-equivalent official rebuild and complete CGO race pass; narrow independent review is recorded. Existing private pointer proof, actual READ COMMITTED begin, same-pool context preservation, nesting/rollback and Deploy commit/error ownership hooks remain intact. Preserve the unchanged earlier deployment admission test hash in the combined narrow binding.

Whole C01 remains WATCH. All 10 C/E execution packages, real role/provider/Retire UI acceptance, historical governance red findings, upstream/production gates and final maintainer acceptance remain open. No new archive group, rule waiver, commit, publication or production action is part of this narrow repair.
