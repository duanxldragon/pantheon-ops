# Current official v0.18.5 consumption — 2026-11-07

Official release at immutable `1da0834b` is acquired and verified from four GitHub assets; standard reconstruction declares 847 files. Drift: 826 compared /816 identical /10 allowed. Complete readonly CGO MySQL/Redis race: **2170 pass /0 fail /1 real S3 skip /64 packages**, without a source overlay. Modules and test fixture were unchanged by the race. Frontend lint/type-check/build and218 unit tests;59 script tests; fresh migration32clean; generated, auto-recycle, master-detail and many-to-many standard smoke all pass.

[Validation](resume10-consumer-validation.json), [smoke](resume10-consumer-smoke.json), [independent source approvals](resume10-consumer-approval.json), [official provenance](resume9-official-provenance.json), [exact raw archives](resume10-raw-archives.json). All1031 protected source files match the tested candidate. Current hosted CI and formal merge remain required. Original scope is preserved; status stays in-progress until those gates complete.

Explicit runtime gaps: QA-002 deploy attempt table and extra host-import smoke table are absent in the isolated versioned-migration runtime. The earlier unprepared directory smoke failed3 tests/ passed2 and is preserved. Precise standard tests with original QA setup pass; no source fix or security relaxation was applied. S3/no waiver and unsigned freeze remain separate. No new UI source behavior; no visual polish claim.

## Historical evidence

# Current module graph correction2026-11-07

Officialv0.18.4 consumer reconstruction previously required scratch -mod=mod changes. Rebuilder now loads complete packages/tests before reporting, includes normalized module hashes, and aborts failed loads.59 regressions pass;official rebuild829, readonly package/test graph and all-backend build pass without module changes. Drift reports826 examined/816 identical/10 allowed.

[CODE](resume7-module-code-reviewer.md) and [ARCH](resume7-module-architect.md) approve the seven-path patch only; [fingerprints](resume7-module-approval.json). Full official consumer race remains required after the v0.18.5 fixture release; no test-overlay acceptance, provider/freeze approval or aggregate closure claimed. Ops owns this composition script; shared test correction remains Base-owned. No product/shared-source or root-lock override.

## Historical evidence

# Current consumption evidence

## 2026-11-06 当前状态（取代下文历史读数）

用户补充要求：纳入 GitHub 新合并 PR #485，修复 Dependabot #52（minimatch / GHSA-g3hv-75wg-m249，修复版本 11.0.1），发布 v0.18.4，再让 Ops 从 GitHub 最新稳定正式 release 消费。

Base 安全分支 `fix/v0184-minimatch-security` 已快进至 GitHub main `2a7061fa0e5bc943c6dc91166e16dc77d614ee70`。限定受影响版本范围的 minimatch override 已写入；当前网络连接 EACCES，在线及离线安装均失败，package-lock 未更新，未进行新版兼容性/audit/CI 验证，因此安全修复仍未关闭，VERSION 保持 0.18.3。

此前 v0.18.3 发布器 session53536 返回 exit0，输出确认四个正式资产上传完成。再次独立查询 GitHub 时网络连接被禁止；未重切或覆盖 v0.18.3 资产。v0.18.4 尚未切包、发布或消费。

Ops 隔离分支已实现 GitHub 最新稳定版获取/四资产校验/不可变 tag 与 manifest 及 git-archive commit 校验/锁定重建；CLI 拒绝本地 Base 来源参数。漂移检查直接读取验证过的快照字节，覆盖父 Ops HEAD 不同、共享文件删除、篡改/过期缓存及业务迁移冲突。当前消费与迁移测试 54 pass / 0 fail / 0 skip，overlay、frontmatter 和 task-template 校验通过。真实 v0.18.4 暂存重建、全量 CGO race、业务 smoke、GitHub CI 与合并均未完成；Ops 主工作树及正式锁文件仍为 v0.18.2。

两路独立复审未产生新批准：原适配器 MCP 配置解析失败；临时禁用无关 MCP 后代码/架构服务均无法连接。原批准和指纹只作历史，不覆盖本轮变更。当前 source manifest 另存，不将旧 21/21 当作新批准。

阻断：本会话 sandbox 禁止 npm registry、api.github.com 和审查服务网络连接，审批策略 never。未申请提权、未放宽门禁。恢复允许联网的执行环境后按依赖安装验证 → Base review/CI/merge/release gate → v0.18.4 官方发布 → Ops GitHub 下载消费/最终验证/review/CI/merge 顺序继续。

## Historical preparation evidence

# Consumption evidence

In progress. Existing business fix commit dbce4ae retained. Unpublished host backfill renumbered to 000032 and declared with its regression in business-overlay.json. Base 000031 remains untouched. Official v0.18.3 release, rebuilt consumer verification and independent review pending.

## Business migration assembly

Canonical host backfill SQL now lives in backend/modules/business/cmdb/host/migrations, and its regression lives in the host package. businessMigrations lists the paired numbered SQL files; the Ops rebuild script derives fixed backend/pkg/database/migrations destinations and adds them to the generated report. Generic product paths remain forbidden in businessPaths/repositoryOverlayPaths. Missing pairs, duplicate versions/directions, unsafe sources and foundation version collisions block reconstruction.

The initial declaration under backend/pkg/database failed the real rebuild preflight despite the previous shallow check passing. That red evidence is preserved at workspace audit-artifacts/base-freeze-2026-11-05/ops-overlay-invalid-business-path.log. The current checker reuses manifest validation. Fixture reconstruction verifies Base000031 unchanged, exact copies of Ops000032, deterministic output and collision rejection. Current 14 overlay tests and isolated MySQL backfill race pass; the real official-v0183 rebuild is still pending publication.

## Drift ownership repair

Architecture blocker reproduced and repaired; 37 overlay tests pass, and real MySQL race covers missing table, missing column and actual down/reapply/CAS. Final source fingerprints cover 21 files. Final CODE/ARCH re-review is running. Main Ops remains unchanged; official v0183 consumption and full final validation are pending.

Final preparation review: independent CODE APPROVE / ARCH CLEAR;21 source hashes match. Official release and complete consumer verification remain pending.

## 最新续办：明确 v0.18.4 与授权

用户再次明确授权发行 v0.18.4、让 Ops 主线锁定消费并检查结果。已补 `npm run fetch:base-release -- --release pantheon-base-v0.18.4`，该入口只请求指定的 GitHub 正式 tag，不会退回 latest 或其他版本；无参数仍支持最新稳定版。新增回归后 57 tests pass / 0 fail / 0 skip。重新运行真实 v0.18.4 获取命令仍被系统拒绝网络 socket，锁文件哈希保持不变。GitHub查询与npm安装也返回相同阻断。当前审批策略 never，不能发起工具提权请求；聊天授权已记录但不会修改执行环境。v0.18.4 未发行，Ops主线仍为v0.18.2，主线消费结果尚不能验证。详见 v0184-explicit-consumption-status.json。

## 2026-11-06 官方 v0.18.4 获取与暂存验证（本会话）

- 网络恢复后完成 base v0.18.4 发布（PR494/495，tag 0a8229c6，见 base 仓证据）。
- Ops 侧：fetch:base-release 显式获取 pantheon-base-v0.18.4，四资产 SHA256 独立复核一致，锁文件钉定 0a8229c6。
- 暂存重建 827 文件；overlay 契约 53 路径 OK；shared-drift 826 文件与官方快照一致（10 allowlist、164 overlay 跳过、2 派生迁移校验）。
- CGO race 全量 exit 0；前端 lint/type-check/build/218 单测全绿；全新库版本化迁移重放至 version=32、dirty=0（Base 000031 + 业务 000032 装配成功）。
- 生成模块业务 smoke：governance/auto-recycle/master-detail/many-to-many 4 项通过；host-real 失败（CMDB 分组折叠导致菜单标签 hidden），用 v0.18.2 基线全新库同跑得到完全相同失败，判定为既有环境态问题，非 v0.18.4 回归。
- tenant 矩阵在全新库上 fixture 供给 403 permission.denied，同属长命 QA 库状态假设；托管 CI 的确定性种子为权威门禁。
- test:scripts 存在 15 个与 clean HEAD 完全相同的既有治理失败（ci.yml 期望、遗留 manifest 词汇），单独排队。
- 下一步：Ops 基线替换与消费 PR（L2 门禁）。详见 ops-v0184-staging-validation.json 与四份 smoke 日志。

## 2026-11-06 基线替换与托管 CI 收口

- 工作区基线替换：sync 后共享范围与暂存树 byte-identical（copied=347，删除上游移除文件 16），提交 5ee377d chore(base): consume the pantheon-base-v0.18.4 shared snapshot。
- 消费 PR：duanxldragon/pantheon-ops#171（base main），PR body 经 scripts/check-pr-governance.mjs 机械校验通过。
- 托管 CI 全绿：Quality Gates pass、Security Gates pass（两项必跑门禁）、Smoke Sanity pass 12m31s（本地 host-real/tenant 全新库失败在托管确定性种子下通过，证实为本地 fixture 状态假设）、Backend Tests/Frontend Contract/Go Lint/CodeQL/Secret Scan/Duplication Gate/Docs Governance 全 pass。
- 非阻塞既有问题：Vercel 预览部署在 #168–#171 全部 fail（外部账号侧，非本改动）；Copilot reviewer 非 collaborator 不可指派，已在 PR body 显式说明。
