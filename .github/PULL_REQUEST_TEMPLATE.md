## 变更摘要

- 改动层级：
- 改动模块：
- 目标问题：
- 预期影响：

## 边界说明

- [ ] 本次改动仅涉及单一层级
- [ ] 本次改动涉及跨层，已说明边界与依赖

> 如果跨层，请补充说明：本次为什么跨层、各层分别承担什么职责、是否影响菜单/权限/i18n/审计。

## 验证记录

- [ ] 后端测试：`go test ./...`
- [ ] 前端构建：`cd frontend && npm run build`
- [ ] 系统页 smoke：`cd frontend && npm run test:smoke:system`
- [ ] 角色授权 smoke：`cd frontend && npm run test:smoke:role-auth`
- [ ] 其他专项验证已补充
- [ ] CodeQL 结果已检查并解释
- [ ] GitHub required checks 通过
- [ ] Copilot review 已请求，或已说明当前仓库/账号不可用
- [ ] 已启用或确认将启用 squash auto-merge

补充说明：

## 审核留痕

- Copilot review：requested / automatic-policy / unavailable
- CodeQL 结果：
- GitHub checks 结果：
- Auto-merge：enabled / not-enabled / not-applicable
- 是否高风险改动：
- Residual risk / follow-up：

## 检查清单

- [ ] 已明确本次改动归属 `platform`、`system/auth`、`system/iam`、`system/org`、`system/config` 或 `business/*`
- [ ] 未把认证、IAM、组织、配置等系统域职责混写
- [ ] 前端新增展示文案已使用 i18n
- [ ] 菜单、页面授权、操作授权、接口授权边界保持清晰
- [ ] 涉及数据库/权限/菜单/接口变更时，文档已同步
- [ ] 已确认不会泄露敏感配置、账号密码或 Token
- [ ] 已确认本次 PR 由 GitHub required checks、CodeQL 和分支保护负责最终合并门禁
