# O2 v12 Generator review candidate

L2 parent: docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md.
Ownership: only NEW continue-o2-live-v12 helpers/binaries/tests and o2-current-live-v12 evidence.
Minimum change: reuse selected MinGW tools and installed objdump, recursively inspect PE imports,
and observe native Windows loader file-handle paths in the actual command environment/cwds.
Pin non-OS MinGW DLL bytes/file identity, selected origins and ordered loader context before/after.
OS loader/System DLLs/API sets remain an explicit trusted platform boundary; OS/kernel inputs are not pinned.
Missing, ambiguous loaded imports, foreign/replaced origins and preceding search shadows reject.
Unused lower-priority PATH candidates are recorded; the observed ordered first match determines selection.
All compiler/build temporary files, caches and logs stay in .local.
Original 12 cases/45 assertions and original 15 Go regressions remain equivalent modulo v12 filenames.
Original default SSH, single verifier, no replay, physical lease/cold restart, cleanup and exit codes remain unchanged.

Verification plan: actual v11 sealed request DLL-copy acceptance with zero DLL hash calls,
v12 actual discovery/request rejection for primary and transitive DLL drift; native metadata/trust-boundary negatives;
all retained Python tests; native gofmt/gopls/vet/15 Go race tests/Windows CGO1 race build/Linux CGO0 build;
stable selected source/tool/DLL/helper/owner/history hashes and rehashed archived input members.
Final verification and one exclusive seal follow this handoff. See handoff-final.md for exact completed results.

No remote/credential/production/owner mutations, delegation, model overrides or self-approval.
Independent NEW CODE/ARCH reviews, Dispatcher authorization and actual runtime/public Base/P0 acceptance remain open.

Actual archived members rehashed: 5549. Proof: .harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-current-live-v12/archive-verification-1794303434945656000.json; SHA256 caa7e436f10af3f9cc5c34cba08b51becc0b9ebb12428764fbca303655d81377.

FINAL verification: .harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-current-live-v12/local-verification-1794303434945656000/receipt.json; SHA256 c4833ccdbccfd9a657f79bb024b1afda2b78f31286acdcc71fe477ab71fc53dc. Seven commands exit0; Python15/Go15, zero skips; 9 actual AST parses. Source/tool/DLL/helper/owner/history stable. Windows race/CGO1 and Linux non-race/CGO0 build metadata verified. No runtime acceptance executed; independent NEW CODE/ARCH and Dispatcher remain required.
