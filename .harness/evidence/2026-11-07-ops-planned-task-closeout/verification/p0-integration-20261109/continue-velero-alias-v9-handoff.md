---
title: Sealed Velero alias v9 exact native ReplicaSet replica annotations
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-11
---

L2 parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`. This bounded successor changes business acceptance tooling.

Actual sealed v8 failed at its exact ReplicaSet annotation map before aliases/provider/S3 effects. Root's finite read-only control proves: original Deployment annotations empty; live Deployment only revision string `1`, pinned UID, integer generation1/replicas1 and exactly `strategy={'type':'Recreate'}`; one native ReplicaSet owned by that Deployment and referenced by the pinned original Pod; exact ReplicaSet three-key spec and integer replicas1. The ReplicaSet has exactly revision, desired-replicas and max-replicas annotations, all strings `1`. Its owner chain is intact. Observation SHA256 `b44c27eee6b945ff1a324dd1688ec43a0b5ae94fc3a9cade9e09f177745a873b` and reviewed diagnostic envelope/helpers are protected by the new request.

Kubernetes v1.33.0 official `deployment_util.go` lines230-290 initialize native replica annotations, lines412-426 format desired/max as decimal strings, and lines468-472 make MaxSurge zero for non-RollingUpdate. Thus this exact Recreate/replicas1 controller legitimately uses desired/max string `1`. Both cached controller source hashes are protected.

v9 admits exactly the original expected ReplicaSet annotations plus the three native string-one annotations. The allowance requires the existing pinned Deployment/generation1 and explicit integer replicas1/exact Recreate strategy. RollingUpdate, missing or extra strategy fields, native key collisions in the original policy, wrong/missing/unknown annotation keys or values/types, foreign owners/siblings, extra/missing ReplicaSet spec fields and selector/template drift refuse. The pinned original Pod owner-chain and strict pod-state gates remain mandatory. v8's two exact native serialization paths stay unchanged. No generic ignored annotations, body rewriting or new dependencies were introduced.

Request `continue-velero-alias-v9-request.json` SHA256 `5ee1048e44d101cf78b84b300dc773d2c7314333e55733f3d274c2a2d0fcd4ba`; operation `20261111t043731393694`; schema10.

| Source                                                       | SHA256                                                             |
| ------------------------------------------------------------ | ------------------------------------------------------------------ |
| `.local/ops-acceptance/continue-velero-alias-v9-contract.py` | `4d0cafc45a28a062c9522e1d037c8223751ab6d6f8be8275547a7375eae081cb` |
| `.local/ops-acceptance/continue-velero-alias-v9-remote.py`   | `9808a19641723be60ae77fe8f4613ded2d55c71e2cebf70e11befba4a2a939ae` |
| `.local/ops-acceptance/continue-velero-alias-v9-root.py`     | `58724bdd924161dc41a37eed98917c64b748e4213fcc88507d9ba27063a2623e` |
| `.local/ops-acceptance/continue-velero-alias-v9-generate.py` | `696288f7d6bf4bb865e5e116b2dcb19dc8627ba6493f6823aafedc404d0d841d` |
| `.local/ops-acceptance/continue-velero-alias-v9-test.py`     | `1b0410b652b27714332e689111e2c69f870eea19c685346fb99a1ed4349f2c6f` |

The request protects 384 historical assets, including all v8 sources/request/reviews/failure/COMPLETE inspection, every inherited seal, actual controller observation, envelope/helpers and official upstream bytes. Streaming checks before/after exclusive sealing match every protected hash. The exact41-file baseline adopts only the two completed v8 helper uploads from the consumed failure and inspection. Neither failed/inspection-only receipt grants continuation or replay. Two fresh v9 helper uploads remain exclusive.

Verification: 357 test executions/304 unique regressions, 13 new v9 checks, 52 forgery denials, zero failures/errors/skips. Actual sealed v8 rejects the observed native fixture; v9 accepts it without rewriting objects and retains the hostile gates above. Inherited preparation/readback assertions use only the new generator in memory. Five memory compiles pass. Actual Pyright1.1.414 LSP over all five frozen v9 sources returns zero errors/warnings. Actual Root check-inputs exits0 for this exact request, five helpers,22 owned objects and41-file baseline. All v8 bytes remain unchanged. Evidence is finite local tooling evidence; no v9 remote phase has run.

Fresh independent CODE APPROVE and ARCH APPROVE/CLEAR must bind this request and all five source hashes before connection. Use `continue-velero-alias-v9-report-format.json`: exactly one JSON fence per report, actual lane/decision/independent=true/requestSHA256/full scriptHashes, plus ARCH architecturalStatus=CLEAR. Old v8 approvals remain history and cannot authorize v9. Tooling approval does not close product/P0 acceptance.

Root checks inputs and, after both fresh exact approvals, starts only the new diagnostic:

```powershell
$v9Tool = '.local/ops-acceptance/continue-velero-alias-v9-root.py'
$v9SHA = '5ee1048e44d101cf78b84b300dc773d2c7314333e55733f3d274c2a2d0fcd4ba'
python -B $v9Tool check-inputs --request-sha256 $v9SHA
python -B $v9Tool diagnostic --request-sha256 $v9SHA
```

Continue diagnostic -> repair -> validate -> cleanup separately, checking each actual receipt's exit0/remoteExecuted=true/sourceHashesStable=true/exact request and artifact hash before using it as predecessor. A failure requires exact inspect-failed, preserves uncertainty and forbids replay. Never regenerate this request. Ready/BSL/native alias/S3/BackupRestore/persistence/cleanup remain owned by Root. The postseal full-suite/request-selection readback is recorded separately.

Gate Outcomes: Root's actual v8 gate isolated legitimate native replica annotations; fresh actual controls plus upstream semantics bounded the smallest successor. All failing receipts and consumed versions remain history. No gate waiver, source override, original installation replay or remote author operation occurred. An initial local wrapper reuse hit Windows GBK decoding of an inherited UTF-8 Markdown template; the new local wrapper explicitly reads UTF-8 and retained the already-created identical wrapper without overwriting it. Production five-source pins were untouched.
