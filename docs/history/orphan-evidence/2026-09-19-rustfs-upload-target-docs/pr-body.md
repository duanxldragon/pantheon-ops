## 变更摘要

- 改动层级：docs + i18n 文案（backend builtin JSON + frontend resources）
- 改动模块：system/config（upload 设置组 remark 文案）、system/i18n（内置资源）、docs
- 目标问题：S3 兼容上传目标的示例文案与设计/部署文档只列 MinIO/AWS S3/OSS，未包含已实测验证的 RustFS，且本地开发 endpoint 的 `http://` 前缀语义未沉淀
- 预期影响：管理界面与文档在五语言下将 RustFS 列为已验证 S3 兼容实现；本地 RustFS/MinIO 开发配置语义更明确；无代码行为变更

## Harness 链路

- Task ID：2026-09-19-rustfs-upload-target-docs
- Task Manifest：none
- Evidence：.harness/evidence/2026-09-19-rustfs-upload-target-docs/commands.json
- Verification evidence：.harness/evidence/2026-09-19-rustfs-upload-target-docs/summary.md
- Review Artifact：none
- OpenSpec change：none
- Trivial change：yes
- Quality Profile：i18n
- Ratchet Decision：no-repeat-observed
- GitHub Signal：not-applicable

## Harness adoption markers

> 保留本区块的英文 marker，供 `scripts/harness/check-adoption.mjs` 做机械检查。

- task id: 2026-09-19-rustfs-upload-target-docs
- task manifest: none
- evidence: .harness/evidence/2026-09-19-rustfs-upload-target-docs/pr-body.md
- boundaries: docs + i18n copy only; no runtime behavior change
- backend response contract: none
- backend DTO contract: none
- permission contract: none
- audit coverage: none
- visual evidence: none
- inheritance contract: none
- base drift: none
- Base/ops inheritance: ops consumes this via next foundation release; no manual copy into pantheon-ops

## 边界说明

- [x] 本次改动仅涉及单一层级
- [ ] 本次改动涉及跨层，已说明边界与依赖

> 改动全部落在 pantheon-base 的 system/config 上传文案、system/i18n 内置资源与 docs 层，未触碰权限/菜单/审计契约；上传驱动代码（pkg/upload）零改动。

## 验证记录

- [x] 后端测试：`go test ./modules/system/i18n/... ./modules/system/config/setting/...`（本 PR 触及范围的最小充分集合）
- [x] 前端构建：`tsc -b && vite build`
- [x] 轻量 smoke：i18n 门禁（check-i18n-hardcode / check-i18n-generated-scope / audit-i18n-locales 5 语言 2811 key 无缺失）
- [ ] 如涉及系统域深链路，已补充专项 smoke：不适用（无运行时行为变更）
- [x] 其他专项验证已补充：docs frontmatter check（266 docs）、task-packet template check、eslint
- [ ] CodeQL 结果已检查并解释：由 GitHub required checks 在 PR 上执行
- [ ] 如有 open CodeQL alert，已说明：不适用
- [x] Full Smoke 仅在必要时手动或预发布执行，未错误纳入 PR 必过门禁
- [ ] GitHub required checks 通过：推送后由分支保护执行
- [x] Copilot review 已请求，或已说明当前仓库/账号不可用：创建 PR 后按仓库策略请求
- [x] 已启用或确认将启用 squash auto-merge

补充说明：另有真实端到端证据——RustFS v1.0.0 本地实例与 minio-go v7.2.1 的建桶/上传/Stat/下载/列举/删除全链路验证通过，且 pantheon 上传 API 已实际对接 RustFS 完成字节级读回比对（详见 verification.md）。

## 审核留痕

- Copilot review：requested
- CodeQL 结果：由 PR required checks 执行并留痕于 GitHub
- GitHub checks 结果：推送后由分支保护判定
- Auto-merge：enabled
- Duplication Gate 结果：not-applicable（纯文案与文档）
- 是否高风险改动：no（docs + i18n copy，非高风险范围清单所列）
- Residual risk / follow-up：ops 侧通过下一次 foundation release 消费，无需手抄

## 检查清单

- [x] 已明确本次改动归属 `system/config`、`system/i18n` 与 docs
- [x] 未把认证、IAM、组织、配置等系统域职责混写
- [x] 前端新增展示文案已使用 i18n（既有 key 的值更新，key 集不变）
- [x] 菜单、页面授权、操作授权、接口授权边界保持清晰（未触碰）
- [x] 涉及数据库/权限/菜单/接口变更时，文档已同步（无 schema/权限变更）
- [x] 已确认不会泄露敏感配置、账号密码或 Token
- [x] 已确认本次 PR 由 GitHub required checks、CodeQL 和分支保护负责最终合并门禁
