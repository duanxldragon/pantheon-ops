# k8s38 provenance

- Candidate: current `main` `697c0771` backend source (worktree backend clean), binary `.local/acceptance-20261121/pantheon-ops-claude-t06-20261124-112531.exe`, sha256 `d6a76a5f002ccb6f33b1cc1657aba0337a0e45de9d6734f119d57b08ea0c8a17`.
- Backend: port 8081, DB `ops_acceptance_formal_20261121`, auto-migrate off, `PANTHEON_ALLOWED_ORIGINS` extended with `http://127.0.0.1:5175` for the harness Origin.
- Target: existing K3s on `115.190.251.143`; SSH/TLS pinned by the harness; credentials supplied from DPAPI files for the process only.
- Result: 17 PASS / 0 FAIL / 0 BLOCKED; k12 pod-log WebSocket, k13 credential rotation/restore, k14 namespace ownership/write protection, k15 cleanup all PASS.
- Caveat: the harness `summary-*.json` receipt hardcodes `sourceHead` to the planning baseline `a2d69d4a`; the actual candidate head is `697c0771`.
- The concurrent Codex backend on 8080 was untouched; `runtime-live/ownership.json` was set during the run and restored byte-identically afterwards.
