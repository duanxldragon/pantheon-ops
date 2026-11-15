# O1/O3 independent static review triage

- F1: schema-dependent premise rejected. Existing Group has no ownership columns; the CMDB catalog is global. Host resolution continues to enforce both BizScope and DataScope. Authorization of the new public route still requires released IAM mappings.
- F2: conditional privilege issue rejected after reading updateTask: it calls loadVisibleTask inside the transaction before any mutation. Add a cross-DataScope regression before final acceptance.
- F3: repaired by reading only referenced groups and ancestor chains, with budget/depth limits and cycle/orphan rejection. Unrelated group edits no longer change the selection revision.
- F4: repaired. Derive a purpose-specific signing key from PANTHEON_DEPLOY_CREDENTIAL_KEY. Shared replica configuration is mandatory; key rotation invalidates tickets. Missing/invalid secrets fail closed.
- F5: repaired using a single transaction service helper preserving ServiceState and transaction-bound BizScope reads.
- F6: repaired. Request context propagates to provider/DB/transactions; canceled-create/update/preview/start regression and full race verify no unintended mutation.
- F7: normalized both stored and manual IPs; ambiguous canonical IPs reject. Identical raw manual selections resolve deterministically; editing input intentionally requires re-resolution.
- F8: repaired. Invalid snapshots return conflict even on trusted internal starts.
- F9: exclusions intentionally remove targets before eligibility; excluded assets never execute. Scope revision still binds changed host facts.
- F10: explicit host/manual numeric ID optimization is implemented and verified against 10,001 real MySQL rows. Dynamic scope/group and canonical-IP scans retain the explicit 10k cap; large dynamic scopes still require a separate design.
- F11: empty installed-components inventory remains a reported health/verification risk; claim snapshots include Status. Secrets stay ciphertext/hash only. Final independent re-review is still required.
