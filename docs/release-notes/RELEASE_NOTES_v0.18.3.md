# Pantheon Base v0.18.3 — Release Candidate Notes

Candidate patch for `release/0.18`, prepared on 2026-11-05. **Not published.** The last published foundation remains `pantheon-base-v0.18.2`; this candidate requires independent review, successful hosted checks, merge and the existing release gate before publication and consumer lock updates.

## Changes

- Bind outbound HTTP dialing to the DNS addresses verified by the SSRF helper, retaining request Host/TLS SNI, deadlines, address-family handling and redirect validation.
- Consume a refresh token atomically once. Session rotation updates only lifecycle fields and checks the previous refresh JTI, tenant, active status and expiry, so stale refresh/switch attempts cannot revive revoked sessions or overwrite concurrent tenant changes.
- Preserve failed dependency audit/installation exit codes in the security workflow. Existing strict versus report-only policy remains in force.
- Close system route-group, data-scope schema, multi-page candidate loading and generator formatting/build defects already implemented in the Nov 3/4 remediation; retain regression checks and update canonical documentation.
- Preserve all current tenant V1 capabilities: compat/multi, initialization, membership, authorized login/switch, system-domain isolation, tenant operations, plan configuration and quota measurement/display. Business-domain isolation and enforced quotas are outside the existing V1 contract.

## Upgrade notes

- The migration chain includes `000031_drop_permission_role_data_scope_policy`. Read [the production preflight/runbook](../runbooks/MIGRATION_000031_PRODUCTION_DROP.md), inspect existing data and take the required backup before production execution. No production migration was performed by this task.
- Refresh retries after consumption, concurrent rotation/revocation or storage failures may require signing in again. Failed refresh tokens are not restored.
- SSRF rejects private or mixed DNS answers under the existing protected policy. Verify Ops datasource/webhook connectivity and the documented allowlist/port policy; do not disable the protection to pass tests.
- A failed vulnerability scan now correctly fails strict workflows; fix the actual failure instead of changing thresholds.

## Consumer impact

`pantheon-ops` must consume the published checksummed artifact using its official upgrade/rebuild overlay path. Do not copy shared files from Base main or advance the live lock to this unpublished candidate.

The isolated Ops repair branch renumbered its unpublished host-state backfill to `000032`, preserving foundation-owned `000031`, and declared its migration/test assets in the business overlay. Validate both in the officially reconstructed consumer before applying changes. The isolated repair branch's earlier passing tests do not establish that the upgraded migration chain is correct.

## Validation and publication state

Current-candidate commands, exact exit codes, runtime results and gaps are maintained in [the execution evidence](https://github.com/duanxldragon/pantheon-base/blob/135730fd6f63ad73cf40f6e4b321fdd9f69ea932/.harness/evidence/2026-11-05-base-freeze-release/summary.md) and [functionality matrix](https://github.com/duanxldragon/pantheon-base/blob/135730fd6f63ad73cf40f6e4b321fdd9f69ea932/.harness/evidence/2026-11-05-base-freeze-release/functionality-matrix.md).

PR [#489](https://github.com/duanxldragon/pantheon-base/pull/489) is a draft. The repaired code candidate `57b42ace` has 27 successful hosted checks and two conditional skips, including successful structure, security, CodeQL, module smoke and Quality Gates. Sonar reanalysis reports zero unresolved issues (previously nine) and quality gate OK, without new exclusions. Exact candidate/check references are in [hosted validation](https://github.com/duanxldragon/pantheon-base/blob/135730fd6f63ad73cf40f6e4b321fdd9f69ea932/.harness/evidence/2026-11-05-base-freeze-release/hosted-validation.json); a subsequent evidence-only commit needs its own hosted checks. User-authorized independent gpt-6-astra round2 reviews returned APPROVE and CLEAR after five fixes. The final two-function SSRF extraction also passed both independent delta reviews (APPROVE/CLEAR). Current-source full validation, hosted checks, merge, release and official consumer upgrade remain incomplete.

## 独立评审修复

- 角色、岗位分页追加主键排序，保证同排序值静态数据集跨页完整。
- 生成必填布尔创建 DTO 区分缺失与 false，并同步模型映射。
- SSRF 已验证地址按剩余时间分配拨号期限，保留 Host/SNI 和整体超时。
- 运行态证据声明冲突保留任一 true 的要求；token-storage 扫描恢复正常报告。
