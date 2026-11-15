# O2 v11 source-stable tooling candidate

Parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`; L2,
business acceptance tooling only. Product/backend/frontend/Base sources, old
helpers, approvals and sealed requests remain unchanged. No credentials, remote
execution, delegation or self-approval occurred.

Ten finite helpers and three included Go files use the v11 filename prefix.
The original 12 cases, 45 assertion calls, Go tests and protected runner bodies
remain byte/AST equivalent modulo filenames. Default Deploy SSH, independent
host-key verification, one verifier connection, no replay/reconnect, real
physical lease/cold restart and exact cleanup remain intact. Runtime artifact
filenames and the durable request-attempt directory belong to v11.

Discovery retains Windows non-race named-file/package tests and Linux inputs,
and adds actual Windows `-race` named-file AND owner-package test discovery.
All selected Go/Cgo/C/assembly/Syso/embed/module inputs are hashed; MinGW header
and native linker inputs are discovered from selected Cgo packages and flags.
There is no fixed-count inventory fallback. Both executable build metadata
records are checked against the actual platform/race/cgo settings.

Final receipt: `local-verification-1794299401982924300/receipt.json`, SHA256
`39f049a42f95bb6d011e4769cb5dc2d1e283676474b49a52b5b0577861df1d5f`.
Seven commands passed: 12 Python regressions (original 11 retained), gofmt,
native gopls, vet, 15 actual Go race tests, Windows race build and Linux build.
Before/after 5,522 source and 390 tool inputs, ten helpers, 232 protected history
pins and 65 original owner pins are stable. Logs/caches reside solely in `.local`.
RED actual old discovery misses race Go/Syso mutations of local byte copies;
GREEN actual new discovery selects them and `validate_request` rejects both.

Receipt compatibility deliberately retains `o2-v10-review-v1`: original final
owner receipts bind the unchanged owner handoff; new independent tool reviews
must bind the actual v11 request. Authorization uses `o2-v11-authorization-v1`.
The v10 request SHA256 remains
`f5bdc55e94a56dbeec56429b20189eeedd4cfc5524a85cdf81ce0bd0f294c2a2`.

Independent v11 CODE/ARCH reviews, Dispatcher authorization and real VM
execution remain open. Runtime cases executed: zero. Public Base identity/IAM/
audit, public HTTP/browser/SSE and P0 acceptance remain open. This handoff is
generator evidence, not approval or authorization.
