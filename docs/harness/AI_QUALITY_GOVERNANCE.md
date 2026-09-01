# AI Quality Governance

This document applies the portable Harness Engineering method to `pantheon-base`.

`pantheon-base` is an admin-platform consumer of the method. The reusable method belongs in `../../pantheon-harness/`. This file only defines controls that depend on the current business architecture, repository layout, and CI stack.

## 0. 版本信息

| 版本 | 日期 | 变更 |
|---|---|---|
| v1.0 | 2026-06-15 | 初始版本 |
| v1.1 | 2026-06-26 | 增强质量门禁矩阵（按档位差异化）、新增与 pantheon-harness 同步说明 |

## 1. 方法论同步
### 1.1 Canonical Source

`pantheon-harness/` 是方法论的 Canonical Source。以下文件应与 pantheon-harness 同步：
- `TASK_PACKET_SPEC.md` - Task Packet 规范
- `VERIFICATION_EVIDENCE_SPEC.md` - 验证证据规范
- `REVIEW_LOOP_SPEC.md` - Review 循环规范
- `HANDOFF_PROTOCOL.md` - 交接协议（v1.1 新增）
- `ERROR_RECOVERY_STRATEGY.md` - 错误恢复策略（v1.1 新增）
### 1.2 本地定制

本目录下的文件如有本地定制差异，应在文件末尾添加 `LOCAL_ADDITIONS.md` 说明。
## 2. Boundary

Portable across projects:

- task packet lifecycle
- evidence and review artifact shape
- failure ratchet rules
- guide, sensor, state, gate, template, adapter vocabulary
- multi-agent handoff expectations

Specific to `pantheon-base`:

- system module contracts
- auth, IAM, config, org, i18n, generator, and dynamic module quality profiles
- GitHub Actions topology
- GitHub gate and review-evidence scope
- Playwright smoke route selection
- base-to-ops inheritance constraints

Do not promote `pantheon-base` module details into the portable method unless the same failure appears in another unrelated repository template.

## 3. Quality Profiles

Every non-trivial task should choose one primary profile. Multiple profiles are allowed when a task crosses boundaries.

| Profile | Applies To | Required Guides | Required Sensors | Evidence |
|---|---|---|---|---|
| `auth-security` | login, session, MFA, JWT, cookies, CSRF, lockout | `docs/contracts/SYSTEM_AUTH_CONTRACT.md`, `SECURITY.md` | backend tests, security review, CodeQL/security gate when applicable | command summary plus security boundary note |
| `permission-policy` | IAM policy, menu policy, role/user/dept/post permissions | `docs/contracts/SYSTEM_IAM_CONTRACT.md`, `docs/designs/PERMISSION_MODEL.md` | policy mapping tests, backend tests, focused smoke if UI permissions changed | before/after policy mapping or route authorization note |
| `i18n` | locale resources, hardcoded text, generated i18n scope | `docs/designs/I18N_MODULE_DESIGN.md`, frontend i18n scripts | i18n hardcode check, generated-scope check, relevant backend tests | locale key/resource summary |
| `ui-runtime` | user-facing pages, layout, forms, tables, dashboards | UI design docs and acceptance docs | frontend lint/build, focused Playwright smoke, rendered evidence when visual behavior changed | screenshot or explicit no-UI-change statement |
| `generator` | low-code generator, dynamic modules, scaffolded modules, generated feature ledger | generator and dynamic module design docs | generator contract tests, feature-ledger drift checker, generated-module cleanup check, smoke if runtime module path changed | generated diff, ledger snapshot status, cleanup result |
| `ci-workflow` | GitHub Actions, CodeQL, duplication, review evidence, smoke topology | `docs/designs/QUALITY_AND_SECURITY_STRATEGY.md`, `docs/GITHUB_GOVERNANCE_CHECKLIST.md` | workflow posture, YAML inspection, affected workflow run when available | workflow intent and gate classification |

## 4. Required Task Packet Addendum

For non-trivial work, add this block to the task packet or implementation note:

```text
Quality Profile:
Portable Failure Class:
Owner Layer:
Consumer-Specific Controls:
Required Sensors:
Required Evidence:
Ratchet Decision:
Delivery Governance:
GitHub Signal:
Deferred Code Issues:
```

`Ratchet Decision` must be one of:

- `no-repeat-observed`
- `guide-updated`
- `sensor-added`
- `gate-updated`
- `template-updated`
- `adapter-updated`
- `registry-only`

## 5. 质量门禁矩阵（v1.1+ 新增）
### 5.1 按档位差异化配置

| 门禁 | L0 | L1 | L2 | 超时 | 失败处理 |
|---|---|---|---|---|---|
| Go 测试 | - | 修改的包 | 全部包 | 5min | 阻塞 PR |
| TypeScript 检查 | - | 涉及文件 | 全局 | 3min | 阻塞 PR |
| Lint | - | 涉及文件 | 全局 | 2min | 警告可过 |
| Frontend Build | - | 涉及模块 | 全栈 | 10min | 阻塞 PR |
| SonarQube | - | 新增问题 | 全部 | 15min | 按规则处理 |
| Smoke 测试 | - | 关键路径 | 完整套件 | 20min | 阻塞合并 |
| UI 检查 | - | - | 必须 | 10min/screen | 阻塞 |
| 安全审计 | - | - | 必须 | 30min | 阻塞 |

### 5.2 门禁通过标准

```
通过 = 所有必需门禁已执行且结果满足标准
阻塞 = 任一必需门禁未通过或未执行
警告 = 非必需门禁未通过，记录但不阻塞
```

## 6. CI Signal Placement

Use fast and deterministic signals in PR required checks.

PR required:

- docs governance
- frontend contract, lint, build
- backend tests
- duplication gate
- smoke sanity
- security gates
- feature-ledger drift checker for generated-module / schema-generated changes

Auxiliary or scheduled:

- full smoke suite
- deep dependency audit
- broad ecosystem drift checks

Rules:

- Full smoke is valuable but should not block every ordinary PR.
- Duplication must be visible in PR and merge queue checks, but the current full-repository baseline is enforced on protected-branch push or manual quality review until a new-code duplication gate exists.
- CodeQL should have one primary execution path in the security workflow.
- Non-trivial PRs should carry Copilot review status plus residual-risk notes when automation cannot fully prove architecture, intent, or maintainability.
- A slow or flaky sensor must either be narrowed, moved later, or made advisory until it is reliable.

## 7. Ratchet Operating Rule

When a failure recurs:

1. Add or update a row in `docs/harness/failure-registry.md`.
2. Decide whether the owner layer is portable method, consumer template, consumer repository, or agent adapter.
3. Promote the smallest useful control.
4. Measure recurrence in the next two weeks or next three related tasks.

If the same pattern reaches human review three times and no control was promoted, the harness work is incomplete even if the code patch passed CI.

For generated-capability drift, the smallest useful control is usually the feature-ledger checker plus the `schema/generated/feature-ledger.json` snapshot, not another manual review note.

## 8. Metrics

Track weekly:

- AI first-pass local gate success rate
- repeated failure recurrence rate
- PR quality gate P50 and P95 duration
- manual remediation hours spent on CI/security/review issues
- new-code CodeQL alerts or review-gate exceptions
- new-code duplication
- new-code backend coverage for included modules
- sensor false positive and false negative counts
- percentage of repeated failures with a ratchet decision

