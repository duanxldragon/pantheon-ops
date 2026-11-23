# T02 固定部署模板参数与卸载路径安全 — 实施检查点

- Task: `2026-11-20-ops-deploy-fixed-template-security`
- Status: `completed`（round9 三项裁定齐：超时文档化 A / nginx 延后 T16 / -race 以 codex 记录接受；验收全闭合）
- Checkpoint: 2026-11-23（自主收口 round8 + 裁定 round9）
- Baseline: planning `a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc`, foundation `pantheon-base-v0.18.9` / `6316f9b7`

## 更新（2026-11-23 round8，自主收口）：filedist Windows 路径 bug 已修复；验收三项全部有实据

维护者移交自主收口（Codex 第三轮独立复验不再运行）。遗留的候选产品缺陷——filedist 分发 Windows 反斜杠 bug——本会话修复：`sftp_client.go` `remoteDir := path.Dir(remotePath)`（Windows 上 `filepath.Dir` 把 POSIX 远端路径变成 `\data\dist` 反斜杠串，pkg/sftp `MkdirAll` 建字面垃圾目录致 transferFailed 掩盖）+ 新增 `sftp_client_test.go` 回归测试，本机复测证实。验证：`go build` OK、gofmt 净、回归测试 PASS、本包声明验证 `go test -count=1 ./modules/business/deploy/...` ok。详见 [STATUS 第八轮](../../tasks/2026-11-20-ops-deploy-fixed-template-security/STATUS.md) 与 [round8 回执](round8-filedist-path-fix-20261123.json)。

**验收三项闭合**：① 七模板合法参数成功+远程执行前拒绝注入/危险目录（14 安全用例+codex 反例 GREEN）；② 旧任务/重试/卸载路径无法绕过校验、密码特殊字符按既有引用契约可用（round2/3 反例 GREEN）；③ 隔离 SSH 注入探针无远程标记副作用、主机指纹不符被拒绝（round4 d04/d05 同候选 bad479fc 实据）。

## 更新（2026-11-23 round4）：V7 注入/指纹在同候选 bad479fc 实机关闭

deploy lane run `omx_accept_20261121_codex_t02d`（receipt `binarySha256=bad479fc…`、exit 0、source/runtime stable）：

- d04 wrong-fingerprint：`SHA256:AAAA…` start → task `failed`，远端 marker-neg 不存在 → PASS。
- d05 注入 negative：`serviceName=nginx$(touch /tmp/<runId>/injected)` → create HTTP 400 `business.deploy.task.templateParamsInvalid`，未 preview、未远端 start → PASS。
- d03 合法 SSH：真实远端 marker + 日志回读 + 无 secret → PASS。
- Go 合法安装/回滚：`codex-fix-live-20261122-182309` `TestDeployLiveVM*` real VM、`-race`、exit 0。

**仍开放**：合法 nginx_systemd 装机/卸载实机（共享 VM 完整源码构建 = 厚重系统变更，需维护者放行装机方式，归 T16 fresh-install）；完整 `-race ./...`（本 shell TSan 不可用）；Codex 第三轮独立复审。

## 本轮完成

1. 在 `deploy_fixed_templates.go` 建立单一参数安全门 `validateFixedTemplateValues`，同时接入创建/快照校验与渲染入口，覆盖七个固定模板：
   - `installRoot` / `dataRoot`：绝对 POSIX 路径、长度上限、拒绝 `.`/`..` 段、控制字符、空白和 shell 元字符。
   - 卸载动作：拒绝 `/`、`/etc`、`/usr`、`/var` 等危险根目录，包含尾部斜杠归一化。
   - `serviceName` / `rootUser`：保守字符集 + 长度上限。
   - 端口：十进制且 `1..65535`。
   - `hostname`、`version`、`storageRetentionPolicy`、`scrapeConfigFile`、JDK 参数：按字段语义使用保守 token/路径/类名校验。
2. 渲染期防御性复验：
   - `renderFixedTemplateScript` 使用 `task.Action`（而非参数中可能缺失的 `action`）作为有效动作，旧任务卸载也走危险根目录检查。
   - 包版本 `pkg.Version` 虽为包元数据，但 MySQL 与 Redis fallback 会拼入 shell 字面量，现统一在渲染门校验；允许 semver build metadata `+`。
   - 创建/模板步骤快照校验同样注入归一化动作，避免卸载路径只在执行期才被拒绝。
3. Redis `requirePassword` 从裸 heredoc 插值改为 `REDIS_PASSWORD=<shell-quoted>` + `printf` 追加，`$(id)`、反引号、分号等无空格载荷不再执行；空白/换行仍按 redis.conf token 传输约束拒绝。
4. 新增 `deploy_fixed_safety_test.go`，覆盖注入、引号/换行、路径穿越、危险卸载根、非法端口、密码不透明性、七模板合法/非法元数据、包版本和旧快照渲染。

## 验证

- `go test -count=1 ./modules/business/deploy/ -run 'TestFixedTemplate' -v` → 14 PASS。
- 既有固定模板兼容回归（MySQL/Redis/MinIO/Harbor/JDK/Prometheus）→ PASS。
- `go test -count=1 ./modules/business/deploy/...` → `ok`，exit 0。

## 来源判定

- SEC-001 的静态代码路径已闭合：创建、快照、渲染、旧快照卸载均有共享校验；参数或包版本进入 shell 前被拒绝。
- 不代表真实环境验收完成。V7 的隔离 SSH 注入副作用检查与主机指纹不符拒绝未执行。

## Round9 收口（2026-11-23，维护者裁定齐）

- ① 执行超时：**选 A**——`deploy_service.go` `deployExecutionTimeout` 补注释明确 per-attempt 预算 × `deployAttemptMax` 重试语义，不改行为。
- ② nginx_systemd：**环境性延后定稿**，归 T16 fresh-install 专用装机。
- ③ `-race` 全量门禁：**以 codex 独立资绿记录接受**（本 shell TSan 环境限制）。
- T02 status → `completed`；改动保持 uncommitted。

## 保持开放的 gap（全部已按维护者裁定收口）

- ~~V7 隔离 SSH 注入探针与指纹校验~~：已在同候选 bad479fc 实机关闭（t02d d04/d05，见更新）。
- ~~Windows filepath.Dir 分发 bug~~：第八轮修复 + 回归测试（范围超 expectedFiles，按授权自主收口修复，T10 归属已记录）。
- ~~③ 完整 `go test -race ./...` 提交门禁~~：第九轮以 codex 独立 `-race` 资绿记录接受（本 shell TSan 分配失败）。
- ~~① 执行超时语义裁定~~：第九轮选 A（per-attempt 预算文档化，不改行为），见第七轮 triage + `deployExecutionTimeout` 注释。
- ~~② nginx_systemd 合法装机/卸载实机~~：第九轮定稿环境性延后，归 T16 fresh-install 专用装机。
- ~~Codex 第三轮独立复审~~：维护者指令移交自主收口，以第八轮 documented 自审替代；后续可另约独立复审。

## 证据

- `commands.json`（真实命令、cwd、exitCode、源码 SHA-256）
- `review.md`（findings-first 自审与机器可读 linkage）
