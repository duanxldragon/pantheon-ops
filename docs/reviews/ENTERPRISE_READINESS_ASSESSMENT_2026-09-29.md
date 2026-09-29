# Pantheon Ops 企业级就绪评估

**评估日期**: 2026-09-29
**评估版本**: VERSION 1.4.1（无任何 v* 发布 tag，版本机制失效）
**继承基线**: foundation-release 锁定 pantheon-base v0.13.0（base 当前 v0.14.1，滞后 18 commits / 2 个 minor）
**评估方式**: 三路并行源码审查（后端与继承纪律 / 前端 / 工程化 DevOps），全部结论附 file:line 证据
**评估目标**: 判断 ops 作为 base 的第一个业务仓库是否达到企业级标准

## 执行摘要

**总体结论：综合成熟度 3.5/5，低于底座（base 4/5）。**

继承纪律本身优秀——327 个共享 Go 文件抽查 99.1% 一致、前端底座零手抄 drift、overlay hook 使用规范，"base-first"协作模式真实运转。业务契约（菜单 / 权限点 / i18n / 审计 / 数据范围）落地到位。但存在三个结构性缺口：**业务层完全未跟进 base v0.14 的多租户、base 基线滞后 2 个 minor、业务测试覆盖稀疏且无版本化 SQL 迁移**。当前适合作为企业内部运维平台使用；对外交付前必须先完成租户接入与基线升级。

| 维度 | 评分 | 一句话结论 |
|---|---|---|
| 后端业务与继承纪律 | 3/5 | 契约合规、99.1% 零漂移，但多租户未跟进、测试稀疏、无 SQL 迁移 |
| 前端业务 | 4/5 | 底座零 drift、复用良好，业务零单测、baseline-swap 挂起 |
| 工程化与 DevOps | 3.5/5 | quality 门禁硬 fail，但缺 release-gate、drift 检测不阻断 PR、版本机制失效 |

## 一、后端业务与继承纪律（3/5）

### 业务模块盘点

11 个业务域 + shared/opsinit，合计约 36K LOC / 165 个 Go 文件，契约层统一为 handler/service/model 三件套（无独立 repository 层，GORM 直写在 service 内）：

| 模块 | LOC | 测试文件 | 备注 |
|---|---|---|---|
| k8s | 10011 | 6 (13%) | 最大，cluster/helm/rbac/ingress/backup 等 8 子域 |
| deploy | 9035 | 5 | 发布流水线 |
| cmdb | 5197 | 9 | host/group/label，CRUD+import/export+SSH collect |
| observability | 2890 | 3 | |
| cicd | 2383 | 2 | 单文件聚合 |
| filedist / service / bizscope | 1800-2300 | 1-3 | |
| capability/secrets/metrics/shared/opsinit | <200 | 0 | 薄壳/工具 |

### 继承纪律（亮点）

- 327 个共享 .go 文件哈希抽查：**324 全等 / 3 漂移**（99.1% 一致）
- 唯一违规漂移：`pkg/security/ssrf/validator.go:102` 签名扩展 `CreateHTTPClient(timeout)` —— 属应回 base 的 generic drift
- 另一处漂移 `menu/component_registry.go:28` 是合规 overlay hook（base 预留扩展点）
- 业务层零 import `modules/system/*` / `modules/auth/*`（grep 空结果），只依赖 `pkg/*` 与 contracts
- 权限点统一命名 `business.cmdb.{host,group,label}.permission.*`，经 `SeedMenusFunc/SeedI18nFunc` 注册（`cmdb/module.go:13-22`）；审计走 `common.SetAuditMetadata`（业务域 94 处）；数据范围 `DataScopedGroup` 412 处使用

### 关键缺口

1. **多租户完全未跟进（最严重）**：base v0.14 已发布多租户，ops `business/*` 全目录 `TenantID/tenant_id` 零匹配，业务模型无 tenant 列、查询无 tenant filter
2. **基线滞后**：lock 在 v0.13.0，未消费 v0.13.1 SSRF 修复与 v0.14 全部租户工作
3. **业务表无版本化 SQL 迁移**：19 个迁移全部是 system/tenant 主题，业务表全靠 GORM AutoMigrate，缺索引/约束版本化管理
4. **测试稀疏**：34 个 _test.go / 165 文件（20.6%），k8s 仅 13%，cicd 2/13，service 1/9，无覆盖率门槛
5. **漂移检测只看重建**：`inheritance-drift-detection.yml` 只做 base 兼容性重建 + 编译通过，不比对共享文件哈希——ssrf 签名漂移因此漏网，且仅周一 cron、不挂 PR
6. `.foundation/releases/` 残留 6+ 个历史 bundle（v0.8.8 起）

## 二、前端业务（4/5）

### 已达企业级

- **8 个业务域**（bizscope/cmdb/deploy/k8s/service/cicd/filedist/observability），约 30 个懒加载路由
- **复用良好**：SearchToolbar 用于 10+ 列表页，PageLoading/Error/Empty/AppTable 广泛引用，usePermission 65 处，各域自带动 locales
- **底座零 drift**：src/components、src/modules/system、tests 与 base 完全一致；src/core 仅 router 注册表有合规差异；App.tsx md5 相同
- prebuild 12 项契约门禁与 base 一致；业务 smoke 12 个 spec（base 仅 5 个），含 API 层覆盖

### 缺口

1. **业务零单测**：vitest 单测全部与 base 逐字节一致，无一业务用例，质量仅靠 smoke 兜底
2. **v0.13.1 baseline-swap 挂起**：lock 声明 pendingWork 未执行；package.json 身份仍是 `@duanxldragon/pantheon-base-ui@0.11.2`，版本/身份混乱
3. PermissionAction 组件在 business 中零使用，按钮权限表达不统一
4. tsconfig 未开 strict（与 base 一致，需 base 先行）

## 三、工程化与 DevOps（3.5/5）

### 已达企业级

- quality.yml 核心门禁硬 fail：docs-governance（frontmatter/encoding/structure/overlay）、backend-tests(race)、smoke-sanity（MySQL/Redis 服务 + 起后端跑 smoke）、quality-gates 汇总
- security.yml 有 OSV/secret/zizmor enforce gate
- business-overlay 机制完整：rebuild-from-base.mjs + check-business-overlay.mjs + 单测
- 部署与 base 同构：三阶段非 root Dockerfile、k8s（3 副本滚动 + HPA + Prometheus 注解）、Grafana
- 文档治理完整：PROJECT_INHERITANCE.md、acceptances/ 20+ 双语、runbooks 齐全

### 缺口

1. **缺 release-gate**：无发布五硬门禁（base 有），且**无任何 v* 发布 tag**，VERSION=1.4.1 与 RELEASE_NOTES_v1.2.0 脱节，版本机制失效
2. drift 检测不阻断 PR（仅周一 cron），为事后检测
3. go-lint / duplication 在 push 下仅报告
4. 根目录 28 个 .md，大量 COMPLETION/RELEASE 报告未归档（与 base 同样问题）
5. Dockerfile 注释仍是 "Pantheon Base 生产镜像"，复制未本地化

## 四、关键缺口清单（按优先级）

| # | 缺口 | 类别 | 说明 |
|---|---|---|---|
| 1 | 业务模型/查询接入 TenantID + 租户 filter | 后端 | 跟进 base v0.14，对外交付的硬前提 |
| 2 | 升级 foundation-release 至 v0.14.1 并回归（含 baseline-swap） | 继承 | 消除 18 commits 滞后 |
| 3 | ssrf validator 签名扩展回流 base | 继承 | 消除唯一违规 drift |
| 4 | k8s/cicd/service 补单测至 ≥50% 文件覆盖 + CI 覆盖率门槛 | 测试 | 覆盖与 LOC 占比严重不匹配 |
| 5 | 业务表引入版本化 SQL 迁移 | 数据层 | 替代纯 AutoMigrate |
| 6 | 漂移检测加共享文件哈希比对并挂 PR | 工程化 | 堵住签名级 drift 漏网 |
| 7 | 建立发布 tag 机制，对齐 VERSION/tag/release notes | 工程化 | 当前无法按 tag 发布 |
| 8 | 业务前端补单测、统一 PermissionAction | 前端 | |
| 9 | 清理根目录报告与 .foundation 历史 bundle | 卫生 | |

## 五、适用性判断

- **企业内部运维平台**：可用。继承机制健康、业务契约完整、smoke 覆盖主要链路
- **对外交付 / SaaS 化**：缺口 1（多租户）、2（基线）、5（迁移版本化）是硬前提
- **协作模式验证结论**：base-first + foundation release + overlay 的多仓继承模式**被证明可运转**（99.1% 零漂移是强证据），但漂移检测需从事后 cron 升级为 PR 阻断 + 哈希比对，才能支撑更多业务仓库接入
