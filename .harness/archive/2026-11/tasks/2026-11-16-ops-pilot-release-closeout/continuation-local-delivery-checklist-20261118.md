# C06 Ops 交付与独立演练清单（准备版）

所有勾选未执行。此清单关联 `2026-11-16-ops-pilot-release-closeout/manifest.json` 的四项验收；不得把历史 V1 勾选、Base 正式发布或本地 CI 接线测试作为当期业务验收。

## 1. 前置条件与候选冻结

- [ ] C01–C05 的 summary/review/commands/runtime evidence 指向同一最终候选，P0=0；有解释的跳过不能替代关键验收。
- [ ] 记录独立执行者、候选 full SHA、业务版本、目标 OS/arch、构建工具版本、UTC 时间、工作目录、命令、真实 exit/output。
- [ ] `git rev-parse HEAD` 与已通过 C05 的候选相同，`git status --porcelain` 为空；源码归档不来自脏工作区。
- [ ] 用当前 lock 从正式 Foundation 重建暂存树，保存 `.business-overlay-report.json`、owned/generated 文件 SHA-256，验证所有业务迁移配对与装配字节。
- [ ] 列出 FileDist/ArgoCD/Helm/Observability/Velero 等逐项开放状态。开放项 E04 通过；裁剪项实际菜单、路由/权限、可见能力、说明验证通过，不能只声明 Out。
- [ ] P1 延期包含维护者决定、owner、截止日期、补偿控制；既有质量门禁全额保留。

## 2. 命令来源（此处未运行）

以下从 Ops 根目录执行，路径取自现有脚本；暂存目录只能在任务拥有的 `.tmp/` 内，不能覆盖运行工作区：

```powershell
git rev-parse HEAD
git status --porcelain
npm run check:business-overlay
npm run check:shared-drift
npm run rebuild:from-base -- --target .tmp/pilot-release-candidate
node scripts/business-overlay/check-business-overlay.mjs --root .tmp/pilot-release-candidate
npm run check:shared-drift -- --root .tmp/pilot-release-candidate
```

重建入口会读取 `foundation-release.lock.json`，校验正式 release archive/repo snapshot 并生成报告；不读取 sibling Base HEAD、不升级 lock、不复制共享文件。暂存树依赖检查用 `-mod=readonly`，不能在验收中临时改 module 依赖。

构建入口来源：`Dockerfile` 已有前端 `npm ci --ignore-scripts`、Arco 补丁、`npm run build` 以及 Linux amd64 后端 `go build`，通过 `PANTHEON_VERSION` 注入业务版本。它是构建来源，不是已验证的安装/离线 runner。交付执行时再固定所有 base image digest、依赖来源、镜像 ID、目标架构，验证版本与完整业务 overlay。

独立构建若采用源码方式，后端入口是 `backend/cmd/server`；前端入口是 `frontend/package.json` 的 build。Windows 完整 race 必须前置 `D:\msys64\mingw64\bin` 并启用 `CGO_ENABLED=1`；镜像里的 `CGO_ENABLED=0` 构建不代替 race。通过 C05 的源码 hash 必须与包内 provenance 一致。

## 3. 包与校验和契约

- [ ] 包名使用 Ops 业务版本，例如 `pantheon-ops-<business-version>-<os>-<arch>.tgz`；版本在冻结后填写，不能沿用 Base tag 或未验证的旧 GA。
- [ ] 包含当前候选的后端二进制/运行镜像、完整前端 dist、版本化迁移来源及装配结果、非秘密配置模板、确切安装/升级/回滚文档、release notes 和 verification summary。
- [ ] 包含 `foundation-release.lock.json`、`business-overlay.json`、当前 `.business-overlay-report.json` 与源码归档（或可验证 immutable checkout 指针）；来源、文件 inventory 和 hash 逐项一致。
- [ ] 离线交付包含 app/前端静态服务器及 MySQL/Redis 等必要镜像的 archive、digest、许可证/依赖信息；目标断网环境不能隐式下载 npm/Go/镜像。
- [ ] 包不包含开发数据、uploads、真实秘密、浏览器 session、私钥、kubeconfig、`.env`、node_modules 缓存、旧 fixture 或任务临时日志；运行数据单独备份，不混入代码包。
- [ ] 写 `delivery-manifest.json`：schemaVersion、businessVersion、opsCandidateSha、sourceArchiveSha256、foundation release/baseCommit/archive/repoSnapshotSha256、lock/overlay/reportSha256、build OS/arch/tool versions、bundle name/hash、逐文件 hashes、migration source/destination/version/direction/hash、extension exposure、C01–C05/E04 evidence 引用。
- [ ] `SHA256SUMS` 记录包及所有交付资产；manifest 的逐文件表排除自身与校验表，避免自引用。外层包 SHA-256 保存在外置校验表并绑定 handoff evidence。
- [ ] 独立执行者在解包前核验外层 hash，解包后核验每个文件 hash；篡改一个字节必须拒绝安装，不能仅核验文件名。

冻结后的 PowerShell 校验形式（`$deliveryArchivePath` 是执行者选定的任务内绝对路径，此处不赋值也不运行）：

```powershell
Get-FileHash -LiteralPath $deliveryArchivePath -Algorithm SHA256
```

## 4. 安装、重复启动和备份

- [ ] 使用独立任务 namespace、随机/专用端口、唯一 MySQL schema/Redis namespace、SSH sandbox/K8s namespace 与任务拥有的卷；保存身份与清理边界。
- [ ] 安装文档给出真实可运行的 app、前端静态资产服务与 API 代理/SPA fallback 入口，健康 URL、配置文件、production 必需配置、进程/日志/超时/终止方式；共享入口缺失先 Base-first，不在 Ops 复制通用 runner。
- [ ] 空库通过真实版本化迁移及 seed 启动；保存 schema_migrations 版本/dirty、业务表/索引/seed/registry/hash。不得挂载旧 system_init.sql 或用 AutoMigrate/手动 stamp 替代版本化路径。
- [ ] production 缺少安全配置应 fail closed；只记录秘密是否配置及非敏感指纹，不输出凭据。
- [ ] 登录、业务菜单/API、两部门/业务范围最小角色正反向、引用删除保护、部署 preview intent/version、任务执行/日志、K8s 真 provider 核验通过。
- [ ] 重复安装/启动后 schema/seed/唯一键无重复，不重跑已完成任务、不产生额外外部写入。
- [ ] 保存业务数据、schema、uploads/外部 artifact 引用、加密材料的独立受控备份，明确备份恢复所需 key 保管；校验备份 hash 并在另一个隔离实例实际恢复。

## 5. 升级、故障恢复和回滚

- [ ] 选择明确的上一 Ops 业务候选与包/hash/迁移版本，不能把 Base release 当上一业务包。旧实例先通过最小正向验收并填入可比较哨兵数据。
- [ ] 保存升级前 schema version/dirty、表与审计哨兵、运行 artifact、进程身份、备份 SHA-256；停止本任务旧进程后才替换新包，确认不双写。
- [ ] 用真实新包执行升级，核对增量迁移、seed 不覆盖维护者数据、数据/外部引用/角色权限一致，重跑同候选关键 smoke 与桌面/窄屏交互证据。
- [ ] 在隔离环境注入至少一个迁移失败和一个启动/依赖失败；记录真实非零 exit、dirty/health 拒绝、日志与状态，不能手动 stamp 版本消除失败。
- [ ] 验证修复/重试使用实际恢复入口；任意破坏性 down migration 明确数据边界。未有无损 down 证明时从已验证备份恢复数据与旧 artifact，不把仅换旧二进制称为回滚。
- [ ] 停止新实例，仅恢复本任务旧包、兼容配置与备份；核验旧包/数据/加密材料 hash、schema version/dirty、角色/业务哨兵、外部资源引用及日志。
- [ ] 恢复过程中不自动重放部署/备份/K8s 等历史外部写任务；再次升级成功，证明重试与回滚可重复。
- [ ] 所有命令有起止 UTC、cwd、候选 hash、环境 identity、exit、stdout/stderr 安全日志与预期/实际对照；清理仅处理本任务拥有资源。

## 6. 四项验收映射与收口

| 原 manifest 验收                                             | 本清单落点 | 当前证据状态 |
| ------------------------------------------------------------ | ---------- | ------------ |
| 独立执行者从 clean checkout/包安装、升级、回滚且候选校验一致 | §1、§3–5   | not-run      |
| C01–C05 完成、交付/回滚通过、P0=0、P1 有维护者决定与补偿     | §1、§4–5   | not-run      |
| Ops 业务包且不以 Base release/merge/旧 GA 替代验收           | §2–4       | not-run      |
| 开放扩展 E04 或实际裁剪菜单/路由/权限/能力/说明              | §1、§4     | not-run      |

- [ ] release notes 仅列实际交付、实际裁剪和已批准风险，附候选/迁移/hash/安装/回滚与 evidence。
- [ ] 保存包与独立执行者 evidence，协调者独立 review；最终功能/视觉验收由维护者完成后才更新 C06/父任务状态。生产迁移/维护冻结仍属于独立轨道。

## 7. 本次观察到的准备缺口

1. 原 `docker-compose.yml` 只启动 MySQL/Redis，带固定 container_name、固定端口、明文开发凭据与 legacy `system_init.sql` 挂载，不能原样作为本清单的隔离业务安装入口。本轮不改该文件。
2. 原 `Dockerfile` 虽复制了前端 dist，但不能仅从 COPY 推定 SPA/static/代理已能服务。需要当前候选独立 HTTP 验证与真实 runner/服务入口记录；本次未发现独立 install/upgrade/rollback CLI，后续应核对锁定 release 契约，禁止虚构命令。
3. 当前 `.business-overlay-report.json` 的 baseCommit 为 `3203624e55438866edb951bd74e671eab38d2f16`，与 lock 的 `6316f9b7189c86542424145fa49344ad737e952b` 不同；最终候选必须重新生成报告。该旧报告不能证明当前继承状态。
4. 当前 lock 固定 `pantheon-base-v0.18.9`，archive SHA-256 `d8672932ff8f8eca683bee7b4c510558bdfb7bb6540b902ece2ded388458142a`，repo snapshot SHA-256 `6782ffb9b8858ff536c5d601dd32ca136951b8f8c70b5e0fc37cdd78ad64794e`。这只是读取的元数据，不代表本次重新校验资产成功。
