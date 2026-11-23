# T02 第二轮独立验收（2026-11-22）

结论：**changes-required / REQUEST CHANGES**。两个 HIGH 缺陷均由非执行渲染反例确认；T02 保持 in-progress。原有 14 项安全测试和 4 项兼容回归通过，新增 2 个顶层反例失败（4 个子场景失败），整体 exit 1、0 skip。实现者原证据记为 `Freebuff coding agent`，本审查不改写其归属。

## 阻塞发现

### R2-01 · HIGH · Redis 配置占位参数错位

[deploy_fixed_templates.go](../../../backend/modules/business/deploy/deploy_fixed_templates.go):478 的 `fmt.Sprintf` 尾部参数将 `passwordWrite` 放在 `port` 前；模板 448–455 行的顺序为 port、dir、logfile 根、EOF 后的密码写入片段。当前参数错位使无密码配置实际生成：

```text
port
dir 6379
logfile /data/redis/data/logs/redis.log
...
EOF
/data/redis
```

有密码时，`port` 后接 `if [ -n "$REDIS_PASSWORD" ]; then` 和 `printf`，整段 Shell 密码写入逻辑进入 redis.conf heredoc；EOF 后仍出现 `/data/redis` 裸执行命令。合法默认参数也无法产生正确配置。`TestCodexReviewRedisGeneratedConfig` 的有密码/无密码两个子场景均失败。确认的是生成内容，未运行安装脚本或 Redis。

交由业务实现者修复：按占位符顺序重排参数，确保密码写入在 EOF 后；将完整 `port 6379`、`dir /data/redis/data`、`logfile /data/redis/logs/redis.log` 和 heredoc 边界断言纳入业务回归。只检查片段存在不能证明配置结构有效。

### R2-02 · HIGH · 重复斜杠绕过危险卸载目录校验

同文件 [validateFixedUninstallRoot](../../../backend/modules/business/deploy/deploy_fixed_templates.go):156–166 只去掉末尾斜杠/空格并逐字比较禁止目录，未按 POSIX 路径语义归一化。渲染 `installRoot="//etc"`、`installRoot="/usr//local"` 均被接受，生成含 `rm -rf` 的卸载脚本；Linux 下分别指向被禁止的 `/etc` 和 `/usr/local`。

`TestCodexReviewRepeatedSlashUninstallRoot` 两个子场景均失败。**没有执行卸载脚本，也没有发生本轮远端删除操作。** 风险判断来自实际生成结果与 Linux 路径语义。交由业务实现者采用 POSIX 归一化或统一拒绝重复分隔符，并覆盖创建、快照、旧任务渲染与卸载入口；Windows `filepath.Clean` 不能直接替代 Linux 目标路径语义。

## 验证证据与边界

实际测试在 `backend/` 执行，overlay 参数使用 apparatus 文件的绝对路径；其余参数如下：

```text
go test -race -count=1 -timeout=120s -v -overlay codex-round2-apparatus/overlay.json -run 'TestCodexReview|TestFixedTemplate|TestDeployFixedTemplateScriptsRenderForBuiltInComponents|TestDeployRedisTemplateUsesSignalStop|TestDeployJDKTemplateServiceSemantics|TestDeployPrometheusTemplateUsesCustomConfigPath' ./modules/business/deploy
```

- [真实测试回执](codex-round2-tests-20261122-155917.json) · [日志](codex-round2-tests-20261122-155917.log)：18 顶层 PASS、2 顶层 FAIL、4 子场景 FAIL、exit 1、0 skip。
- [Reviewer overlay](codex-round2-apparatus/overlay.json) 与 [补充测试](codex-round2-apparatus/deploy_fixed_safety_overlay_test.go) 只在 Go 编译中替换测试文件；业务源码及原业务测试未修改。
- [准备时四文件哈希](codex-round2-apparatus/source-before.json) 与本轮报告生成时逐项一致，也与实现者 [commands.json](commands.json) 四个哈希一致。报告 JSON 保存完整绑定。
- 首次 [setup 日志](codex-round2-tests-20261122-152845.log) 因 Go 缓存 Access is denied 未进入业务测试；授权提权重跑后才得到上述真实产品反例失败。首试不计为产品缺陷。
- MinGW/CGO=1、race、MySQL required 环境已配置；本次所选测试只渲染模板，**不证明数据库行为、完整仓库 race 或实机部署成功**。
- 代码审查 recommendation：**REQUEST CHANGES**；架构状态：**BLOCK**。两路独立证据见 [lane 回执](codex-round2-lanes.json)。

## 进度与旧证据复核

新树 18 张卡为 **3 in-progress（父任务/T16/T02）、15 planned、0 completed**。T02 新增包版本、有效 task.Action 和密码传输校验是真实进展，但不能关闭安全验收。

旧 K8s run12 的 22 场景 PASS、部署 d00–d06 的 7 PASS、CMDB retry 的 8 PASS 保留为历史范围证据。本轮没有新构建或新实机结果。现有 k8s33 二进制 SHA256 为 `5cd98af198657a667a5deb0016fe3884265f3d96222da212b5c46ab40ad3e368`；K8s 74 文件回执仍匹配，旧部署回执与本轮四个修改文件漂移且缺 `service_link_http_test.go`，不能认证新模板候选。

上一轮 CAP 连接500/查询400误判 PASS、V1/V2/V3 的403现状核实、V5 creator 过滤缺口、V4/V6–V8、恢复/调度/Ingress/安装、WebSocket/UI、完整仓库 race 和清理复核均没有新增关闭证据；旧十包 residual 继续开放。[第一轮报告](../2026-11-20-ops-review-live-runtime-acceptance/codex-progress-review-20261122.md) 保留原件。

实现者“无授权 SSH 目标”的 gap 描述需纠正：维护者在会话中已授权 `115.190.251.143` 作部署/K8s 验证目标；真正缺口是**修复后的新候选尚未实机验收**。本轮不连接服务器、不输出凭据、不执行生成脚本。

Codex 本轮只准备验收反例和审查证据，业务代码继续由 Claude 修复。修复后应先证明上述反例转绿，再补合法配置/危险路径边界及同候选实机证据；当前不批准关闭 T02 或整体验收。

[机器报告](codex-round2-review.json)。
