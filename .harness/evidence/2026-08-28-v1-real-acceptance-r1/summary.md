# V1 真机验收 R1（2026-08-28 下午）

## 范围与结论

在 `115.190.251.143`（CMDB host 111）上对最新 main（`69c932a` + 本轮迁移修复）执行 V1 冻结验证的**第一批真机链路**。

**结论：部署引擎真实链路（认证 → 凭据解密 → SSH 连接 → 指纹校验 → 脚本渲染 → 远程执行 → 日志回传 → 步骤状态追踪）全部打通并实测通过。**
JDK 安装步骤因 180MB 包下载耗时挂起（14:18 进入 step.running，截至 14:32 仍在下载），其余步骤按计划分批执行。

## 环境

- 后端：`pantheon-ops-v1.0-test.exe`（main 69c932a + 迁移 23/24/25），端口 8090
- 数据库：本地 MySQL `pantheon_ops`（schema 版本 22 → 25）
- 凭据：`PANTHEON_DEPLOY_CREDENTIAL_KEY`（取自仓库 `.env`）
- 目标：115.190.251.143:22（root，凭据 ref 10，密码模式）

## 已验证链路（实测通过）

| # | 链路 | 结果 | 证据 |
|---|---|---|---|
| 1 | 新二进制构建 + 带凭据密钥启动 | ✅ | 8090 健康检查 200；迁移版本升至 25 |
| 2 | 登录认证（cookie + CSRF + Bearer） | ✅ | 业务 API 全部走通 |
| 3 | 创建业务域（v1-acceptance, id=161） | ✅ | POST /business/bizscope |
| 4 | 主机绑定业务域（host 111 → scope 161） | ✅ | POST /business/bizscope/161/hosts/bind → bound:true |
| 5 | 部署包创建（fixed 模板 + sourceUrl） | ✅ | jdk-17.0.20 (pkg 156)；**包不可变校验生效**（PUT 被拒，幂等设计符合预期） |
| 6 | 部署任务创建 + 启动 | ✅ | 任务 130→134；任务编排、状态机、租约正常 |
| 7 | SSH 主机指纹校验 | ✅ | ed25519/RSA 指纹被拒（host key mismatch），ECDSA `SHA256:wGNI4...` 通过——**发现 Go ssh 客户端协商 ECDSA host key**，指纹校验机制真实生效 |
| 8 | 凭据加密/解密链路 | ✅ | 未配 key 报 `key_missing`；配置 `PANTHEON_DEPLOY_CREDENTIAL_KEY` 后正常解密 |
| 9 | 部署引擎执行链路 | ✅ | trace: SSH connected → step started → script rendered → step.running（14:18:34）；日志流/步骤事件 API 可查 |
| 10 | 下载源故障处理 | ✅ | 清华镜像 17.0.10 404 → 换 17.0.20.1 真实可下载，任务重跑成功进入执行 |

## 本轮修复（Codebuff 遗留：模型-数据库漂移）

| 迁移 | 内容 | 说明 |
|---|---|---|
| 000023 | `biz_deploy_task` 增 5 列（rollback_enabled / rollback_task_id / source_task_id / rollback_strategy / rollback_snapshot） | RollbackTask 功能所需，模型有字段但迁移缺失 |
| 000024 | `biz_deploy_task_step` 增 `changed` 列 | 步骤变更追踪所需 |
| 000025 | `biz_deploy_task_secret` 表结构升级（param_key/encrypted_value → secret_key/secret_encrypted） | 保留旧数据（2 条测试遗留，v1 格式兼容读取） |

另外确认：CMDB host 模型 `ssh_port`/`cpu_cores`/`os_version` 等列命名正常，**无真实漂移**（此前脚本误报）。

## 挂起/未完成（后续见 R2）

- [x] JDK 安装完成（180MB 下载中，task 134 running）→ **R2 收尾：task 134 最终 `failed`，`Process exited with status 28`（curl 下载超时）。非平台缺陷，为真机外网带宽受限。改用小体积组件先行验证**
- [x] Nginx / Redis 安装 → **R2 完成，源码编译方式均 success（详见 R2 证据）**
- [x] 卸载 → 重装幂等 → **R2 完成**
- [x] 制造失败 → 断点续跑 → **R2 完成，checkpoint 只重跑失败 step**
- [ ] K8s 集群纳管（kubeconfig 加密入库、连通性探测）——6443 端口可达，但真机无 kubeconfig 文件，**需用户提供**
- [ ] Release 镜像发布与回滚（依赖 K8s 纳管）
- [ ] 平台自监控（/metrics + Prometheus 抓取）

## 风险提示

1. JDK 180MB 包下载是真机网络瓶颈（非平台问题），验收时建议预置包到内网源或用更小验证物（如 redis/nginx 二进制）先完成幂等/断点测试。
2. 本轮在**开发库**直接应用迁移 23/24/25——正式发布前需在隔离 schema 重演一遍（检查表 §2 尚未勾选）。
