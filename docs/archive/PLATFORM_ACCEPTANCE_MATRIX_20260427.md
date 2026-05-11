# 平台层验收矩阵（2026-04-27）

更新时间：2026-04-27

类型：Acceptance
归属层：platform
状态：Superseded

本文用于沉淀 Pantheon `platform` 层与各系统域的统一验收基线，避免后续只记住“测过”，却没有一份可持续维护的标准矩阵。

适用范围：

- `platform`
- `system/auth`
- `system/iam`
- `system/org`
- `system/config`
- 低代码辅助开发链路

---

## 1. 当前基线结论

截至 2026-04-27，Pantheon 已完成一轮从 `P0` 安全加固到 `P1` 主链路回归，再到 `P2` 文档基线沉淀的闭环。

当前结论：

- 适合继续作为企业级后台底座推进
- 适合作为业务模块研发加速器继续接入
- 不应误判为成熟的运行时低代码平台
- 低代码仅作为平台补充功能存在，不作为核心后台能力目标

---

## 2. 验收矩阵

| 维度 | 归属层 | 验收项 | 当前状态 | 证据 |
| :--- | :--- | :--- | :--- | :--- |
| 功能 | `platform` | 平台壳层、工作台、标签页、主题、锁屏主链路可用 | 通过 | `test:smoke:system` |
| 功能 | `system/auth` | 登录、refresh、logout、会话安全、超时提示 | 通过 | `test:smoke:system` |
| 功能 | `system/auth` | 安全中心展示运行时认证策略快照（密码长度、失败阈值、锁定时长、空闲超时） | 通过 | `go test ./backend/modules/auth` + `npm run build` |
| 功能 | `system/iam` | 用户、角色、菜单、权限主链路 | 通过 | `test:smoke:system` |
| 功能 | `system/iam` | 角色授权树搜索、父级批量授权、保存回写 | 通过 | `test:smoke:role-auth` |
| 功能 | `system/iam` | 权限工作台识别未知权限、页面链路缺口、API 策略缺口 | 通过 | `go test ./backend/modules/system/permission` + `npm run build` |
| 功能 | `system/iam` | 权限工作台治理报表导出 | 通过 | `go test ./backend/modules/system/permission` |
| 功能 | `system/org` | 部门、岗位页面主链路 | 通过 | `test:smoke:system` |
| 功能 | `system/config` | 字典、设置、i18n、上传、操作审计主链路 | 通过 | `test:smoke:system` |
| 功能 | `system/config` | 导入导出接口可用 | 通过 | `test:smoke:impexp` |
| 功能 | `system/org` | 组织治理任务清单与治理任务导出 | 通过 | `go test ./backend/modules/system/dept` |
| 多语言 | `platform` | 壳层语言切换、默认语言回退、退出后回退 | 通过 | `test:smoke:system` |
| 多语言 | `system/config` | i18n 管理、导入导出、详情/编辑/新增链路 | 通过 | `test:smoke:system` |
| 安全 | `system/auth` | Token 分离、会话吊销、空闲超时、锁屏 | 通过 | `test:smoke:system` |
| 安全 | `system/config` | 设置写操作二次验证 | 通过 | `test:smoke:system` |
| 安全 | `platform` | 动态模块生产默认关闭、开发环境显式开关 | 通过 | 代码与后端测试 |
| 安全 | `platform` | 生产环境密钥校验 | 通过 | 代码与后端测试 |
| 稳定性 | 全局 | 后端测试 | 通过 | `go test ./...` |
| 稳定性 | 全局 | 前端 lint | 通过 | `npm run lint` |
| 稳定性 | 全局 | 前端构建 | 通过 | `npm run build` |
| 稳定性 | 全局 | 菜单契约 | 通过 | `npm run check:menu-contract` |
| 性能 | `system/audit` | 审计模块 benchmark 可执行 | 通过 | `go test -run Test -bench . ./backend/modules/system/audit` |
| 低代码 | `system/config` | 动态模块为研发辅助链路，不作为运行时能力 | 已明确 | 文档结论 |
| 低代码 | `system/config` | 生成器 i18n-first（字段/枚举/占位/审计 key-first） | 通过 | `frontend/src/modules/generator/*` + `npm run lint` + `npm run build` |
| UI | `platform` | 统一后台视觉壳层 smoke | 通过 | `test:smoke:backoffice-ui` |

---

## 3. 已通过验证命令

当前已通过：

- `go test ./...`
- `frontend npm run lint`
- `frontend npm run build`
- `frontend npm run check:menu-contract`
- `frontend npm run test:smoke:system`
- `frontend npm run test:smoke:impexp`
- `frontend npm run test:smoke:role-auth`
- `frontend npm run test:smoke:backoffice-ui`
- `go test -run Test -bench . ./backend/modules/system/audit`
- `go test ./backend/modules/auth`
- `go test ./backend/modules/system/permission`

说明：

- `system smoke` 覆盖 `platform`、`system/auth`、`system/iam`、`system/config` 主链路
- `impexp smoke` 覆盖导入导出治理链路
- `role-auth smoke` 覆盖 `system/iam` 角色授权树专项链路
- `backend/modules/auth` 覆盖安全中心运行时策略快照与密码策略消费
- `backend/modules/system/permission` 覆盖权限工作台治理统计、缺口识别与导出链路

---

## 4. 当前剩余风险

### 4.1 低代码治理风险

- 生成器已完成 key-first 国际化收口，并支持独立英文录入
- 生成器仍更接近研发脚手架，而不是运行时低代码平台

### 4.2 视觉一致性风险

- 登录页、壳层、工作台与系统页已形成本轮视觉 smoke 证据
- 后续仍需持续回归，避免壳层与系统页风格再次漂移

### 4.3 性能风险

- 前端包体仍偏大
- 尚未建立统一平台页性能基线

### 4.4 授权治理风险

- `system/iam` 当前已完成“发现问题 + 导出报表”
- 风险角色整改仍需回到角色授权页与接口策略页协同处理

---

## 5. 后续使用方式

建议以后每次阶段收口都按这份矩阵更新：

1. 先标明属于 `platform`、`system/auth`、`system/iam`、`system/org`、`system/config` 的哪一层
2. 再补每个维度的通过状态与证据
3. 最后只保留“未通过项”和“剩余风险”

这样后续开发不会再回到“功能有了，但权限、多语言、安全、导入导出、视觉一致性没有统一基线”的状态。
