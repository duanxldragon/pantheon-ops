# E04 隔离本地接受准备计划（2026-11-18）

Parent task: `2026-11-16-ops-extensions-live-acceptance`；L2；业务扩展验收。Minimal Complexity Rung: reuse（原 manifest、既有 smoke、provider tests、任务拥有的进程/身份 receipt）。本轮只新增 readiness plan/matrix/commands handoff，不修改应用、既有 manifest/summary/review/commands 或主进度，不回退其他执行者的并发修改。

## In / Out

In：从原验收建立 enabled-provider 条件矩阵；核对隔离 preview 的任务 receipt、进程身份、listener 及两个公开 loopback HTTP 健康入口；记录当前旧 binary provenance、mock/local-fixture/live-provider 的证据等级和精确后续命令/前置/停止条件。

Out：不输出秘密/DSN/会话/进程环境，不请求凭据，不访问外部主机，不创建或操作 provider，不重启服务，不运行 Go/浏览器/heavy tests，不标 E04 完成。自动 exporter/file_sd、实时健康平台、商业 License/运行时 ModuleGate 继续 Out。

## 条件执行与证据规则

1. 先冻结试点真实开放清单。源码装配不等于选中，也不等于当前 binary 的 runtime exposure。
2. FileDist 开放时要求 E01 `completed`；ArgoCD 开放时要求 E02 `completed`；Helm 开放时要求 E03 `completed`。未选能力由 C06 在同候选验证菜单、路由/权限、可见能力与说明裁剪。仅开放 Observability/Velero 不等待未选 E01–E03。
3. 每个 selected provider 记录任务 ownership、当前候选/二进制/迁移/Foundation/source hashes、独立两域最小角色正反向、外部事实与故障恢复；fake/mock、端口监听、HTTP200 或旧源码测试都不能升级为 `LIVE_PASS`。
4. 现有 Observability Web suite 含端口1 datasource/失败 FileDist 目标和固定实验主机的 CMDB 可见性断言；K8s Web suite 含 `page.route(...).fulfill`。这些属于 UI/composed fixture 覆盖，不证明 SSH/SFTP、原生 K8s、Velero 或 Argo 外部事实。
5. `TestDatasourceLiveAcceptance` 强制指定历史外部实验主机，不能在本 scope 运行或为本地 provider 改测试；未真实执行记 `NOT_RUN`，缺少 host 导致 skip 不能作为接受完成。

## 验证与停止条件

- 本次只读 probe：`Get-CimInstance Win32_Process`/`Get-Process`（不取 CommandLine/环境）、`Get-NetTCPConnection -State Listen`，GET `http://127.0.0.1:8080/api/v1/health` 与 `http://127.0.0.1:5173/`（5秒、零重定向、只记 status）。
- MySQL 以 current identity receipt 的 PID/startTime/executable 与 listener 交叉核对。旧 ownership 的 startup PID 和实际监听 child PID不同不单独判 ownership 错误；原观察与补核判定一并保留。
- 验证新增 JSON 可解析、关联文件存在、原三项 acceptance / 三项 conditionalDependencies 完整映射，不运行后续 suites。
- 本地准备停止条件：真实只读结果落盘、matrix/commands/prerequisites 可追溯、静态验证通过，交协调者独立 review 后 freeze。E04 仍 `NOT_ACCEPTED`。
- 依赖执行停止条件：选中能力未完成条件前置、current-candidate binding缺失、task-owned provider/session/fixture provenance缺失或身份不匹配、危险/外部生产/凭据操作不在授权范围。继续独立本地准备，不绕过安全/质量门禁。

未来桌面/窄屏 UI 真实接受执行先用 `impeccable`，保留渲染/交互证据；本轮未运行浏览器，不声称视觉验收。

## Linkage

- Task manifest: `.harness/tasks/2026-11-16-ops-extensions-live-acceptance/manifest.json`
- Task doc: `docs/harness/tasks/2026-11-16-ops-extensions-live-acceptance.task.md`
- Matrix: `.harness/tasks/2026-11-16-ops-extensions-live-acceptance/local-provider-acceptance-matrix-20261118.json`
- Evidence: `.harness/evidence/2026-11-16-ops-extensions-live-acceptance/local-readiness-commands-handoff-20261118.md`
- Machine evidence: `.harness/evidence/2026-11-16-ops-extensions-live-acceptance/local-readiness-commands-handoff-20261118.json`
