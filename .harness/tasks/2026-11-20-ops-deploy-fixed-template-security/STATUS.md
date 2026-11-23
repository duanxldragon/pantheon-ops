# 固定部署模板参数与卸载路径安全

- Status: completed
- Updated: 2026-11-23
- Baseline: a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc

## 第九轮（2026-11-23，维护者裁定齐）：三项残留全部关闭，T02 completed

维护者授权按建议执行，三项裁定收口：

- **① 执行超时语义 → 选 A（per-attempt 预算，仅补文档，不改行为）**：在 `deploy_service.go` `deployExecutionTimeout` 增注释，明确 `ExecutionTimeoutSeconds=T` 是单次 attempt 预算、`executeAsyncHostAttempt` 按 attempt 独立 re-arm、有效墙钟上限 ≈ T×deployAttemptMax；不引入行为变更，避免动摇已验证 deploy lane。第七轮 triage 回执引用保留。
- **② nginx_systemd 完整源码构建装机 → 环境性延后定稿**：归 T16 `2026-11-20-ops-review-live-runtime-acceptance` fresh-install 专用装机，共享 VM 不复跑（nginx/redis/minio/prometheus 用户全部活跃，避免破坏共享验收宿主）。
- **③ 完整 `-race ./...` 提交门禁 → 以 codex 独立 `-race` 资绿记录接受**（本 shell TSan 分配失败属 Windows 环境限制；codex-fix-live 记录已覆盖 real VM race）。

三项合毕，验收全部闭合，T02 进入 completed。改动全部 uncommitted，待统一提审时提交。

## 第七轮（2026-11-23，超时 triage 关闭）：执行超时语义已核实，非缺陷，三选项待维护者裁定

round5"安装 task 14 timeoutSeconds=1500 运行 41 分钟未自动终止"经候选引擎源码 + 产品库 attempt 时间线双向核实，**判定为语义差异而非引擎缺陷**：

- **taskContext 无任务级 deadline**（`deploy_service.go:1597` 仅 `WithCancel`），执行超时在 `executeAsyncHostAttempt`（1708）**按 attempt 独立 re-arm**，重试上限 `deployAttemptMax=3`（52 行）。
- **DB 铁证**（task 14，`execution_timeout_seconds=1500` 正确落库）：3 个 attempt 连续重试（衔接仅约 50ms）——11:17→11:32、11:32→11:47、11:47→cancel。**每个 attempt 都约 15 分钟脚本级失败（`Process exited with status 28`，即模板脚本内 3×curl `--max-time 300`），没有任何 attempt 真正跑满 25 分钟预算**；41 分钟墙钟 = 3×15 分钟重试，而不是一次 25 分钟预算被忽略。
- **超时传播路径本身已证可用**：round-2 的 20s 超时案例复现过 `context deadline exceeded`；若脚本真挂死，首次 attempt 会在 budget 内触发超时。
- **语义缺口**：`timeoutSeconds` 对外呈现为任务执行超时，实际是 per-attempt 预算 × 最多 3 次，有效墙钟上限 ~75 分钟。维护者裁定选项：A) 接受 per-attempt 语义只补文档；B) 任务级总预算 `WithTimeout`；C) 总时间=timeout×attempts 封顶。**建议 A 或 C；不算缺陷。** 这是 T02 质量门禁裁定项。

[triage 回执](../../evidence/2026-11-20-ops-deploy-fixed-template-security/round6-timeout-triage-20261123.json)。

## 第八轮（2026-11-23，自主收口）：filedist Windows 路径 bug 已修复；验收三项全部有实据

维护者移交自主收口（Codex 第三轮独立复验不再运行；本钱独立复核为 documented 自审）。遗留的候选产品代码缺陷——**filedist 分发 Windows 反斜杠 bug**——已在本会话修复并带回归测试：

- **根因在本机 Windows 复测证实**：`filepath.Dir("/data/dist/pkg.tgz")` → `\data\dist`（反斜杠），而 `path.Dir` → `/data/dist`。`pkg/sftp` 的 `MkdirAll` 只按 `/` 切分，整个反斜杠串当成一个组件在 SFTP CWD 下建**字面反斜杠垃圾目录**，真实目录从未建立 → `Create` 报 `SSH_FX_NO_SUCH_FILE` → 引擎掩盖为 `business.filedist.task.transferFailed`（即 round-6 实机现象）。
- **修复**：`sftp_client.go` `remoteDir := path.Dir(remotePath)`（新增 `"path"` import）；`\`-加载反常输入退化为 `.`，`MkdirAll(".")` 无害；本地下载行 `localDir := filepath.Dir(localPath)` 保留 host 分隔符语义。`client.Create` 仍用用户提供的 POSIX `remotePath` 原样。
- **回归测试**：新增 `sftp_client_test.go` `TestUploadContextRemoteDirUsesPathPins` 钉住两处调用行（远端必须 `path.Dir`、本地必须 `filepath.Dir`）——本包无 in-process SFTP server，故按调用点断言，跨 OS 确定性防回退。
- **验证**：`go test -run TestUploadContextRemoteDirUsesPathPins` PASS · `go build ./modules/business/filedist/` OK · gofmt 净 · 本包声明验证 `go test -count=1 ./modules/business/deploy/...` ok（243 项测试，仅 DB/live-VM 门控的 ownership 测试 SKIP）。

**范围说明**：`sftp_client.go`/`sftp_client_test.go` 不在本包 expectedFiles.modify（deploy-only），属 T10 filedist 范围；因 round-6 实机暴露且已定位，经授权自主收口时修复并记录于此，避免带已知候选缺陷归档。改动未提交（worktree 保持 uncommitted）。

**验收三项闭合**：
1. 七模板合法参数成功 + 命令替换/引号/换行/危险目录远程执行前拒绝 —— 14 项安全用例 + codex round2/round3 反例 GREEN。
2. 旧任务/重试/卸载路径无法绕过校验，密码特殊字符按既有引用契约可用 —— round2/round3 反例 GREEN。
3. 隔离 SSH 注入探针无远程标记副作用，主机指纹不符被拒绝 —— **同候选 bad479fc 实据**：d04 wrong-fingerprint → `sshHostKeyMismatch` PASS、d05 fixed-template 注入 → HTTP 400 `business.deploy.task.templateParamsInvalid`、远端无 injected marker（round4 `round4-v7-closed-bad479fc-20261123.json`）。

[第八轮回执](../../evidence/2026-11-20-ops-deploy-fixed-template-security/round8-filedist-path-fix-20261123.json)。

## 第六轮（2026-11-23，已完成）：jdk_systemd 安装 success 实机关闭（file:// 本地源）

**安装 success 在同一候选 `bad479fc` 上实机关闭**。维护者改派：不下载外网包，直接复用 `pantheon-ops/files/amd64` 预置的 `jdk-17.0.15_linux-x64_bin.tar.gz`（183,197,735 字节）——经产品自己的 **上传 + filedist 分发**（filedist task 5 completed、远端 sha `2dfef301…e0e13` byte-exact），再把 tmplate 包的 `sourceUrl` 指向 `file:///tmp/jdk-installer/.dist/jdk-17.0.15_linux-x64_bin.tar.gz`。install task 19（同款 package 18 / jdk_systemd / installRoot=`/tmp/omx_accept_20261121_codex_t02h/jdk`）**11.4 秒 SUCCESS**，产品执行器 stdout 即实据：`JDK installed at /tmp/omx_accept_20261121_codex_t02h/jdk` + `java version "17.0.15" 2025-04-15 LTS` + HotSpot VM 行。对比 round-5 的 TUNA 外网下载 41 分钟仍失败——**全离线 file:// 路径彻底排除网络变量**。

**分发链路两个环境性/产品性发现（已 bypass，未改产品代码）**：
1. **Windows 路径 bug（产品代码，已定位未修复）**：`sftp_client.go:196` `remoteDir := filepath.Dir(remotePath)` 在 Windows 把 `/tmp/jdk-installer/.dist/...` 变成 `\tmp\jdk-installer\.dist`；pkg/sftp `MkdirAll` 只按 `/` 切分，于是整个反斜杠串被当成一个元素，`Mkdir` 在 SFTP CWD（/root）下建了**字面反斜杠垃圾目录** `\tmp\jdk-installer\.dist`，真正的 `.dist` 从未建立 → `Create` 报 `SSH_FX_NO_SUCH_FILE` → 引擎统一掩盖为 `business.filedist.task.transferFailed`（processTask:190-191）。工作区绕过：paramiko 预建真实 `/tmp/jdk-installer/.dist` + maxretry。
2. **task 4 失败的间隔是环境性**：auth.log 显示 13:49:45–13:50:48 是 VM `MaxStartups` 节流窗口（几百条 `Invalid user ubuntu` 爆破连接被丢弃），引擎连接在到达 sshd 前即被丢，非引擎失败。节流清除后再分发即 byte-exact。

**回执**：[round6-filedist-bootstrap-20261123.json](../../evidence/2026-11-20-ops-deploy-fixed-template-security/round6-filedist-bootstrap-20261123.json)（secret-key 缺口：restart-live.ps1 未配 `PANTHEON_BUSINESS_SECRET_KEY`，restart-live-secret.ps1 补齐）· [round6-install-success-20261123.json](../../evidence/2026-11-20-ops-deploy-fixed-template-security/round6-install-success-20261123.json)（install/uninstall task 19/20 executor stdout + 全部复原）。

**复原（字节级）**：合法 uninstall task 20 SUCCESS 后 `installRoot` ABSENT、`PKG_DIR` 模板自清洁 ABSENT、systemd unit 0；厂商 `/etc/profile.d/jdk.sh` 恢复为快照 **sha `72307e44…1dc9cc`、60B、644 root:root**（restoreNeeded 关闭）；package 18 `sourceUrl` restore 回 TUNA；垃圾字面 `.dist` 目录已删。**保留项（有意）**：filedist_file id=1 + backend/uploads 183MB 上传副本 + files/amd64 预置包——按维护者新工作流（上传+filedist）作为 T16 fresh-install 复用资产，不删。`/root/\tmp`、`/root/\tmp\offline` 为 2026-08-29 既有 VM 状态，未触碰。

- Owner: 部署安全负责人
- Priority: P1
- Activation: core
- Dependencies: none

## 第五轮核验（2026-11-23）：jdk_systemd 合法卸载实机关闭，主机字节级复原；nginx 环境性延后定稿

**jdk_systemd 合法卸装在候选 `bad479fc` 上实机 SUCCESS**（uninstall task 17，`uninstallSuccessCount=1`，run `omx_accept_20261121_codex_t02g`）。流程：harness d00–d06 7/7 PASS 后，背景 harness 被会话内存告警回收（非 harness 失败，未重启）；服务端执行器继续，安装任务 14 因 TUNA 镜像吞吐过低、三次 curl `--max-time 300` 中两次 `exit 28`、第三次重试时用产品 cancel API 带内终止（`context canceled`）。随后创建并运行**合法 uninstall 任务**（同 installRoot=`/tmp/<runId>/jdk`、`serviceName=pantheon-t02-jdk`，action=uninstall）：停止/disable 空 unit、rm unit、daemon-reload、`rm -rf` 安装根、`rm -f /etc/profile.d/jdk.sh` 全部执行，成功数 1。

**主机复原（字节精确）**：卸载删除预置 `/etc/profile.d/jdk.sh`（60B 厂商文件）后，用快照恢复为 **sha256 `7230…1dc9cc`、644 root root、60B**（恢复回执 `restoredSha` 一致）；确认 `/tmp/<runId>/jdk` ABSENT、`/tmp/jdk-installer` 部分下载已清理、`pantheon-t02-jdk` 服务 unit 0/active inactive、jdk.sh 为厂商字节。共享 VM 回到运行前状态。

**指纹关键发现**：产品 executor 握手**呈现 ECDSA 主机键**（collectFingerprint `SHA256:wGNI…Gb4`），而 ssh-probe 钉住的 baseline ed25519 指纹 `SHA256:FhLFR…` 仅用于探针 known_hosts。uninstall task 15/16 误传 ed25519 指纹 → 均 `sshHostKeyMismatch`（**顺带在 uninstall 执行路径复现 d04 wrong-fingerprint 拒绝语义**）；改传 collect 指纹后 task 17 SUCCESS。

**执行超时观察（候选侧，待 triage）**：安装任务 14 请求 timeoutSeconds=1500（25 分钟），但 11:17–11:58 共 41 分钟未观察到超时终止，需显式 cancel（对比 t02f 20s 超时曾报 `context deadline exceeded`）。记录为候选端观察项，建议维护者/Codex 裁定执行超时是否应覆盖阻塞 SSH read 全程。

**共享主机 hazard audit（只读）**：nginx_systemd→`userdel nginx` 且 nginx 用户活跃跑 mariadbd/redis-server/argocd-repo-server；redis_systemd→redis 用户活跃（`/data/redis` 真数据）；minio_systemd→minio 活跃（`/data/minio`）；prometheus_systemd→prometheus 活跃（`/data/prometheus`）；jdk_systemd 无 userdel 是唯一环境安全模板。**nginx_systemd 合法安装/卸载定稿为环境性延后**（丢弃共享 VM，归 T16 fresh-install 专用装机）。

[第五轮回执](../../evidence/2026-11-20-ops-deploy-fixed-template-security/round5-jdk-legal-roundtrip-20261123.json)。

## 第四轮核验（2026-11-23）：V7 注入/指纹在同候选 bad479fc 上已有实据

**V7 注入与指纹已在当前候选 `bad479fc`（`pantheon-ops-codex-t02b-20261122-172553.exe`）上由 deploy lane 实机关闭**（run `omx_accept_20261121_codex_t02d`，receipt `binarySha256=bad479fc…`、exit 0、sourceStable/runtimeStable 均 true）：

- d04 wrong-fingerprint：`SHA256:AAAA…`（错误指纹）start → task `failed`，远端 `marker-neg` 不存在 → **PASS**
- d05 fixed-template injection negative：`serviceName=nginx$(touch …/injected)` → create HTTP 400 `business.deploy.task.templateParamsInvalid`，未 preview、未远端 start，无 injected marker → **PASS**（此即 R2 修复后同候选的同款注入反例）

补充：d00–d06 deploy lane 7/7 PASS（含 d03 真实 SSH 安装远端 marker + 日志回读 + 无 secret），全部在 bad479fc 上。Go 层合法安装/回滚在同候选由 `codex-fix-live-20261122-182309`（`TestDeployLiveVMExecution` success/failure + `TestDeployLiveVMCancellation`，real VM，`-race`，exit 0）覆盖。

**仍缺（未在此轮关闭）**：nginx_systemd 合法 install 在共享 VM 上是完整源码构建（`./configure + make`、`apt-get/yum install` 系统包、建 `nginx` 系统用户、写系统 unit 并 enable+start）——属改变共享验收宿主系统状态的**厚重操作**，不属任务命名空间内可逆动作；T02 STATUS「Known Gaps」与 T16 收尾一致将其归为 fresh full install 项，**需维护者决策**（是否允许在共享 VM 上跑真构建，还是用专用装机）。完整 `-race ./...` 提交门禁在本 shell 不可用（TSan 分配失败，Windows 环境限制），codex 已留独立 `-race` 资绿记录。第三轮独立 Codex 复审未运行，T02 保持 in-progress。

## 第三轮修复核验（2026-11-23）：两处 HIGH 反例转绿

第二轮两处 HIGH 已修复且本轮反例 GREEN：`TestCodexReviewRedisGeneratedConfig`（无密码/有密码两子场景）与 `TestCodexReviewRepeatedSlashUninstallRoot`（`//etc`、`/usr//local`）均通过；原 14 项安全测试及 jdk/redis/prometheus 渲染回归一并通过。修复用 `path.Clean`+`ToSlash` 归一化重复斜杠，Redis 密码写入移至 heredoc EOF 之后。codex 自测 `-race` 记录（172253/package/live 18:23）与本次复核一致；本 shell 内 TSan 分配失败未复跑 race，不放大范围。

[核验回执](../../evidence/2026-11-20-ops-deploy-fixed-template-security/round3-fix-verified-20261123.json)。

## 第二轮独立验收（2026-11-22）：changes-required

Codex反例确认两处HIGH：Redis配置fmt参数错位（合法默认参数生成空port/错误dir/密码写入进入配置）；重复斜杠目录`//etc`、`/usr//local`绕过危险卸载根拒绝。原18项PASS，补充2顶层/4子场景FAIL，exit1、0skip。独立代码审查REQUEST CHANGES、架构BLOCK；本包安全验收未通过。只渲染脚本，未执行安装/卸载，未改业务代码。

[独立审查](../../evidence/2026-11-20-ops-deploy-fixed-template-security/codex-round2-review.md) · [机器回执](../../evidence/2026-11-20-ops-deploy-fixed-template-security/codex-round2-review.json)。

## 实现者自审检查点（2026-11-22，保留历史）

实现并验证共享参数安全门，覆盖七个固定模板的创建/快照/渲染/旧快照卸载路径；补齐包版本、有效卸载动作和 Redis 密码传输三处绕过。聚焦 `deploy` 包测试 14 项安全用例与既有兼容回归全部通过。

## Next Atomic Action

**T02 已收口（status=completed，第九轮三项裁定齐）**，无剩余执行项。filedist Windows 反斜杠 bug 修复（`path.Dir` + 回归测试，第八轮）、超时语义文档化（A，第九轮）、nginx 环境性延后（→T16 fresh-install）、`-race` 门禁以 codex 资绿记录接受。改动全部 uncommitted，随 batch 统一提审时提交。

## Evidence

- [commands.json](../../evidence/2026-11-20-ops-deploy-fixed-template-security/commands.json)
- [summary.md](../../evidence/2026-11-20-ops-deploy-fixed-template-security/summary.md)
- [review.md](../../evidence/2026-11-20-ops-deploy-fixed-template-security/review.md)
- [Codex第二轮独立审查](../../evidence/2026-11-20-ops-deploy-fixed-template-security/codex-round2-review.md)
- [Codex反例测试](../../evidence/2026-11-20-ops-deploy-fixed-template-security/codex-round2-tests-20261122-155917.json)
- [第三轮 fix 核验](../../evidence/2026-11-20-ops-deploy-fixed-template-security/round3-fix-verified-20261123.json)
- [第四轮 V7 同候选关闭回执](../../evidence/2026-11-20-ops-deploy-fixed-template-security/round4-v7-closed-bad479fc-20261123.json)
- [第五轮 jdk 合法往返回执](../../evidence/2026-11-20-ops-deploy-fixed-template-security/round5-jdk-legal-roundtrip-20261123.json) · [t02g deploy lane d00–d06](../../evidence/2026-11-20-ops-review-live-runtime-acceptance/runs/omx_accept_20261121_codex_t02g/deploy/) · [Go LiveVM receipt](../../evidence/2026-11-20-ops-deploy-fixed-template-security/codex-fix-live-20261122-182309.json)
- [第六轮 filedist 引导](../../evidence/2026-11-20-ops-deploy-fixed-template-security/round6-filedist-bootstrap-20261123.json) · [第六轮 安装 success 实据 + 复原](../../evidence/2026-11-20-ops-deploy-fixed-template-security/round6-install-success-20261123.json)
- [第七轮 执行超时 triage](../../evidence/2026-11-20-ops-deploy-fixed-template-security/round6-timeout-triage-20261123.json)
- [第八轮 filedist Windows 路径修复 + 验收闭合](../../evidence/2026-11-20-ops-deploy-fixed-template-security/round8-filedist-path-fix-20261123.json)

## Known Gaps

- **全部关闭（round1–round9）**：门禁/修复/V7 注入指纹/jdk 合法往返/超时 triage/filedist 路径 bug 均闭合；nginx 环境性延后与 `-race` 门禁按第九轮裁定接受。
- ~~两处HIGH尚未修复~~：已修复，round3 反例 GREEN（见上方核验）。
- ~~V7 注入/指纹~~：已在同候选 bad479fc 实机关闭（t02d d04/d05，见第四轮）。
- ~~jdk_systemd 合法卸载实机~~：第五轮在 bad479fc 关闭（uninstall task 17 SUCCESS，见上方核验）。
- ~~jdk_systemd 合法安装 success~~：**第六轮关闭**——同候选 bad479fc，install task 19 SUCCESS（file:// 本地源全离线 11.4s，executor stdout 实据 java 17.0.15 LTS），合法卸载+字节级复原完成（见第六轮）。
- ~~执行超时观察项~~：**第七轮 triage 关闭**——判定为 per-attempt 语义 × 3 次重试的语义差异，非引擎缺陷（超时传播路径已由 20s 案例证可用）；维护者从三选项（A 文档化/B 任务级预算/C 封顶）裁定，见 [round6-timeout-triage](../../evidence/2026-11-20-ops-deploy-fixed-template-security/round6-timeout-triage-20261123.json)。
- ~~Windows filepath.Dir 反斜杠 bug（filedist 分发 `sftp_client.go:196`）~~：**第八轮已修复**——`remoteDir := path.Dir(remotePath)` + 回归测试 `TestUploadContextRemoteDirUsesPathPins`，本机 Windows 复测证实反斜杠根因；改动未提交。
- 合法安装/卸载实机（nginx_systemd 完整源码构建）已定稿环境性延后：共享 VM 上 nginx/redis/minio/prometheus 用户全部活跃，归 T16 fresh-install 专用装机。
- 第三轮独立 Codex 复核未运行；完整 `go test -race ./...` 提交门禁未在本 shell 执行（TSan 分配失败，codex 有独立 `-race` 资绿记录）。
- 同候选实机重跑 k8s33 已由 T16 k8s35（22/22）关闭；T02 自身其余项归 T16 收尾。
