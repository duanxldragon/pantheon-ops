# O4 CODE safety repair — Generator proposal (review required)

Parent: `2026-11-07-ops-planned-task-closeout`, L2, business-owned inheritance tooling. This supersedes no review or retained failed stage. No Base, backend, frontend/src, release, lock, waiver, remote or Dispatcher progress mutation.

Confirmed findings: independent CODE REQUEST CHANGES in `../o4-independent-code-review-20261110-01/review.md`. CodeGraph/LSP/AST diagnostic tools are unavailable; exact supplied files and native Node checks are the scoped fallback.

Minimal stdlib lifecycle correction:

1. `foundation-candidate.mjs`: separate observed directory pins from creation authority. Node/Windows mkdir returns no authenticated directory handle. Every created directory, including root and assembler target, is explicitly unproven for deletion. Cleanup preflights every directory and file, refuses foreign entries, then records deferred cleanup without deleting anything or issuing a cleanup-success receipt.
2. Bind regular file creation to O_EXCL open + fstat; open an existing file without truncation, compare the recorded identity, recheck the entire observed chain, then write via that descriptor. All writes, copies, exclusions and final receipt use these guards. Unknown files never gain authority from a scan/report. Checks before and after mutations detect observed replacements; separate syscalls cannot defeat an adversary inside the final check/syscall window.
3. `rebuild-from-base.mjs`: route remaining raw target writes through the supplied filesystem; exclude manifest-selected Base files before copying, preserving distinct file/directory semantics without needing unprovable directory deletion. Keep validation and ownership collision checks. Go commands use a retained alternate modfile workspace, never write target go.mod/go.sum by path; import successful results through guarded descriptor writes. Pre/post full target validation remains required.
4. `rebuild-from-candidate.mjs`: validate the target before verification and receipt, write the receipt through guarded I/O, and return only after final validation. Preserve primary error/cause plus explicit cleanup refusal and all refused resources.
5. Focused tests adapt only syscall injection hooks/cleanup expectations to this lifecycle. Add independent reproductions for the four findings, write/open/receipt/ancestor replacement, full preflight, unproven directories, Go boundaries and immutable fresh attempts. Capture RED before implementation and GREEN after; retain fixture artifacts under this new evidence directory.

Native syntax diagnostics; full candidate suite and business-overlay suites with zero skips; business-overlay checker; original six old/new hashes; repair-only diff; unchanged original/sealed artifacts and metadata hashes. Fixture assembly proves fixture behavior only. Independent CODE and ARCH must review the source proposal and limits before root runs a fresh full attempt. No self-approval or P0/release closure.
