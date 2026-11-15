---
title: Independent Velero v7 architecture review
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-11
---

Native independent architect `/root/post_config_review_arch` returned APPROVE / CLEAR. Root captured the actual verdict because the architect role prohibits file writes. This is tooling approval only; Ready/BSL/S3/BackupRestore/persistence/cleanup, product runtime, publication and P0 remain open. Original v6 approval and superseding BLOCK are preserved in `.local/ops-acceptance/post-config-reviews/`.

The complete ReplicaSet key/type gate is invoked by both annotation admission and pod-state. Actual sealed v6 admits minReadySeconds=120 and foreignSpecField with original owner/template; v7 rejects both, explicit default-like extras, missing keys and noninteger replicas. Revision1/exact Deployment UID/generation/native owner/selector/full template remain mandatory. Normalized successor diff preserves journal, freshness, predecessor and failure-inspection logic. Strict refusal of additional native defaults is intentional.

Independent checks:20 focused in-memory checks passed, writes/processes prohibited;347 historical assets totaling146130300 bytes streamed and matched; five v6 sources retained, remote baseline37 unchanged; actual read-only check-inputs passed for five helpers/22objects; request/all five sources/report-format matched before/after. Full331-execution/278-unique/52-forgery evidence inspected, not independently rerun. Report-format SHA2562d9594e4a1a06f343b26d467ec22f790285f328ffbcf32230f7d68402f6a5692.

```json
{
  "subject": "Velero-v7",
  "lane": "ARCH",
  "decision": "APPROVE",
  "architecturalStatus": "CLEAR",
  "independent": true,
  "requestSHA256": "6bba684f283d6eca328ffe3e66e46391db67ae8606b634941183ea5a99e83ecc",
  "scriptHashes": {
    ".local/ops-acceptance/continue-velero-alias-v7-contract.py": "c33a31f6d6af824a9fdff7560433a6f4a6a2e617ae4f93cc33fa12c7b9e5a2f5",
    ".local/ops-acceptance/continue-velero-alias-v7-remote.py": "637368b547c41c0e11f234873998defe297435513aa2cccef17fddf1d82dfb71",
    ".local/ops-acceptance/continue-velero-alias-v7-root.py": "a9ec79669b0ff5733842319a618415741bfa6b2ef30848d6762eac1936e347d3",
    ".local/ops-acceptance/continue-velero-alias-v7-generate.py": "541c6ab85957a7b5b973a6e6ba30993cec46cf7f55cf8d38b6db6782b73aa870",
    ".local/ops-acceptance/continue-velero-alias-v7-test.py": "da3cd46b5693b8dce507a064c693b5833a24513b7838d05f1b5699eb4fc4122a"
  }
}
```

Gate Outcomes: exact complete-spec regressions close the concrete v6 architecture BLOCK; no remote operation or gate waiver by reviewer.
