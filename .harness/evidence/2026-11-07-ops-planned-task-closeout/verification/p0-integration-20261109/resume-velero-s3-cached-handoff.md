# Offline Velero S3 cached completion

Parent task: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`. Tier L2; business/K8s local acceptance tooling. Generator scope is three new ignored `resume-velero-s3-cached-*.py` validation scripts, new evidence here, and the explicitly authorized cache loader record update. No backend, platform, system-domain, shared source, lock, ledger, original script, OCI archive, or remote resource was changed. This is local administrative integrity reconstruction, not new release trust or provider acceptance.

## Completed result

[`resume-velero-s3-cached-completion-20261110T003228969327.json`](resume-velero-s3-cached-completion-20261110T003228969327.json) independently certifies the actual cached official SeaweedFS 4.53 release and both exact official Velero 1.17.0 / AWS plugin 1.13.0 OCI archives. The reviewed root loader record now contains those same receipt bytes, `exitCode: 0`, `transportVerified: true`. Receipt SHA256: `e54e0c8b9596349bcd36c95cba0f022ea65a9d5b8b9e69b3c0974acd893bc22b`. Staged transfer payload: **229,386,389 bytes**.

The earlier failed loader record is preserved verbatim in [`resume-velero-s3-cached-prior-loader-20261110T003228969327.json`](resume-velero-s3-cached-prior-loader-20261110T003228969327.json). Both failed aggregate transport receipts remain byte-for-byte unchanged. Successful component evidence was reused with its own pinned receipt SHA256; failure statuses were not copied into successful checks.

| Retained source                                         | SHA256                                                             | Reused facts independently rechecked                                                                                 |
| ------------------------------------------------------- | ------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------- |
| `resume-velero-s3-transport-20261109T101156612488.json` | `03ad6bf68c05e84c1237834ad9a0b51946784b8fb1fc0691e01b08ce94dd20c6` | Seaweed metadata, official asset origin/ID/size/digest, release archive and ELF; official Velero CLI archive and ELF |
| `resume-velero-s3-transport-20261109T101459860875.json` | `ee189ce4a3bd4e00f02447436a0ec01f07b86bc6b6bf29e2f97f886be43016b7` | Latest exact OCI tar hashes, official indices/manifests, configs and compressed/rootfs layer chains                  |
| `resume-velero-s3-root-request.json`                    | `6d079b8d2b9dc99ffd278c3d74825e4d72f4ff3f5c5d847d3ddd6781701c9ef0` | Original reviewed scripts and deployment authority remain mandatory                                                  |

The cached Seaweed metadata matches the retained `7908e97ed2a798b18fe07ccf665143a19d5655d98d30ee1b54d93d8c145f7344` SHA256. The unchanged original `verify_metadata` checks version, canonical origin, unique asset, digest and size, and its result is compared to the original receipt including asset ID. Release archive bytes and streamed amd64 ELF contents match the original component receipts; existing extracted binaries are also rehashed without extraction or execution.

For each OCI image, the script rehashes the latest exact tar, the original official index and selected manifest, all tar blobs and corresponding loose cache blobs. It verifies one linux/amd64 manifest selection, generated transport index identity/annotations, config platform and rootfs layer count, compressed blob size/digests, and bounded decompression against every rootfs diff ID. It refuses missing files, modified bytes, filesystem symlinks/reparse points/hardlinks, unsafe or duplicate outer archive paths, archive links/special files, duplicate JSON fields, invalid digests, external descriptor URLs and oversized data. Official rootfs layers are only decompressed and hashed, not extracted. Required input hashes are checked again before publication to detect shared-workspace changes.

## New ignored scripts

| Script                                                          | SHA256                                                             |
| --------------------------------------------------------------- | ------------------------------------------------------------------ |
| `.local/ops-acceptance/resume-velero-s3-cached-completion.py`   | `44433c1a81406988c758ddf0ae2a46d5e2af60b173e8d005d4a365ca84deffc6` |
| `.local/ops-acceptance/resume-velero-s3-cached-check.py`        | `d475fef3d762d4155751ea60f69637a49d00c19189e2a5eafebdac3540a7ddec` |
| `.local/ops-acceptance/resume-velero-s3-cached-loader-check.py` | `9876402c2f81edbe2ef6daba1c4ee88189a74c1c23e591854a04975f45315c84` |

## Fresh evidence

- [`resume-velero-s3-cached-checks-20261110T003216480740.json`](resume-velero-s3-cached-checks-20261110T003216480740.json): **34 passed checks**, including actual complete cache reconstruction, tamper/missing/path/type/duplicate/size refusal, bounded decompression, unchanged source receipts and fail-closed publication. Disposable fixtures were removed. Network connections were forbidden during inspection. No OCI image archive or provider fixture was generated.
- [`resume-velero-s3-cached-loader-check-20261110T003340924706.json`](resume-velero-s3-cached-loader-check-20261110T003340924706.json): **3 passed checks**. Exact AST-extracted original `reviewed()` accepted all **8 original script hashes and 17 product source hashes**, including pinned SSH RejectPolicy. Exact original `load_artifacts()` accepted actual local payloads; the loader bytes equal the new completion receipt. Only the prep-script path was mapped to its actual local location because root uploads it beside payloads remotely. No connect/provision/diagnostic/install function ran.
- Python syntax compilation passed for all three new scripts. Scoped whitespace checks emitted no findings; `git diff --no-index --check` returns 1 because these new files differ from NUL. `git check-ignore` confirmed all three scripts remain ignored. No fixture directory or publication temporary file remained.
- `npm run check:docs-frontmatter`: exit 0, 462 docs checked. `npm run check:task-packet-template`: exit 0.
- [`resume-velero-s3-cached-final-check-20261110T003603802805.json`](resume-velero-s3-cached-final-check-20261110T003603802805.json): final independent rehash of **26 required cache files**, stable retained source/original script hashes, receipt/loader byte equality, valid new evidence links and no temporary leftovers. Both documentation checks passed again after evidence creation.

The initial local loader inspection passed the original root review but failed to locate the prep script inside the cache directory. The corrected inspection maps that single source path read-only and passes; no deployment script was changed or copied. This was an inspection-layout error, not insufficient cached provider bytes. Both successful integrity-check runs remain retained; the intentional missing-fixture publication failures were confined to disposable evidence directories.

## Root handoff and remaining work

Offline completion command (already executed successfully):

```powershell
python -B .local/ops-acceptance/resume-velero-s3-cached-completion.py
```

Root's next already-authorized command, using the original reviewed loader and request:

```powershell
python .local/ops-acceptance/resume-velero-s3-root.py diagnostic
```

Original root wrapper SHA256 remains `65d8155901e32eb76318658972f995b24ec241a38df71d2adb92e0ae694675d5`; original preparer remains `0b5219ab2fe498a81b5d224c35dc8eaa7827f68467ab4dbee413a45a6445cc02`. Do not rerun the online preparer for unchanged metadata to replace this offline receipt. Root must retain the original root-request/script/product checks, run actual binary diagnosis, review its output, then follow the original install/validate path. This lane executed no remote operation or provider call.

Actual provider readiness, OCI import, real backup/restore, D1, full IAM/audit/HTTP, formal P0 and production gates remain open. No full backend/race/frontend build was run because the assigned change consists only of local Python integrity tooling and evidence. UI was untouched. Self-review only, with the leaf prohibition on delegation recorded in [`resume-velero-s3-cached-review.json`](resume-velero-s3-cached-review.json).

Gate Outcomes: integrity checks caught tamper/missing/path/link/duplicate/size violations; unchanged-script/root-request checks passed; local loader inspection caught a source-path layout mismatch and passed after a read-only mapping.
