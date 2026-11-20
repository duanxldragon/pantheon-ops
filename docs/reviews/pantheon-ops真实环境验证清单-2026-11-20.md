# pantheon-ops 真实环境验证清单

2026-11-20 · 配套《pantheon-ops 独立评审与外部报告核实》

**用途：** 静态审查无法回答的 6 个问题，需要在真实环境里拿到证据。每项给出步骤、预期结果、以及结果出现偏差时对应哪条结论要修改。

**未验证声明：** 本清单由 Claude 根据代码和项目文档编写，**没有在真实环境里执行过**。命令里的接口路径、字段名来自代码阅读，可能与实际有出入，执行时以你们环境为准，发现不一致请记录下来，这本身就是一条有价值的结果。

## 0. 安全前提（先读）

- **只用一次性的测试数据和测试账号。** 不要使用生产库、生产凭据或生产 kubeconfig。
- **不要直接使用仓库根目录的 `docker-compose.yml` 暴露到公网主机。** 它把 MySQL root 密码和 Redis 密码写死在文件里，并把 3306、6379 映射到所有网卡。下面的验证用 compose 改为从环境变量读取密码，并只绑定 127.0.0.1。
- 在远程主机上做验证前，确认它不承载其他业务；SSH 部署目标用专用的低权限用户。
- 不要把密码、私钥、kubeconfig 发给任何人，包括 Claude。只发命令输出，发之前把令牌、密码替换成 `***`。

## 1. 起环境

### 1.1 依赖（MySQL 与 Redis）

保存为 `docker-compose.verify.yml`，不要提交进业务仓库：

```yaml
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: ${VERIFY_MYSQL_ROOT_PASSWORD:?set it}
      MYSQL_DATABASE: pantheon_ops_verify
    ports: ["127.0.0.1:3307:3306"]
    command: ["--character-set-server=utf8mb4", "--collation-server=utf8mb4_general_ci"]
  redis:
    image: redis:7-alpine
    command: ["redis-server", "--requirepass", "${VERIFY_REDIS_PASSWORD:?set it}"]
    ports: ["127.0.0.1:6380:6379"]
```

```bash
export VERIFY_MYSQL_ROOT_PASSWORD='<自己生成>' VERIFY_REDIS_PASSWORD='<自己生成>'
docker compose -f docker-compose.verify.yml up -d
```

如果你们已有本机 MySQL、Redis，可以跳过，只新建一个独立的库 `pantheon_ops_verify`。

### 1.2 后端

环境变量名来自 `docs/DEPLOYMENT_GUIDE.md`。非生产环境可以用开发模式自动建表：

```bash
cd backend
export PANTHEON_ENV=development PANTHEON_AUTO_MIGRATE=true
export PANTHEON_DSN="root:${VERIFY_MYSQL_ROOT_PASSWORD}@tcp(127.0.0.1:3307)/pantheon_ops_verify?charset=utf8mb4&parseTime=True&loc=Local"
export PANTHEON_REDIS_ADDR=127.0.0.1:6380 PANTHEON_REDIS_PASSWORD="${VERIFY_REDIS_PASSWORD}"
export PANTHEON_INITIAL_ADMIN_PASSWORD='<至少12位>'
export PANTHEON_ACCESS_TOKEN_SECRET=$(openssl rand -hex 32) PANTHEON_REFRESH_TOKEN_SECRET=$(openssl rand -hex 32)
export PANTHEON_OP_TOKEN_SECRET=$(openssl rand -hex 32) PANTHEON_SETTING_SECRET=$(openssl rand -hex 32) PANTHEON_MFA_SECRET=$(openssl rand -hex 32)
go run ./cmd/server
```

注意：Go 工具链需要 1.26.11（见最近的 CI 修复提交）。如果起不来，先记录报错原文。

### 1.3 登录并保存会话（后续所有 curl 复用）

```bash
BASE=http://127.0.0.1:8080/api/v1
curl -s -c admin.jar -X POST $BASE/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"'"$PANTHEON_INITIAL_ADMIN_PASSWORD"'"}'
# 写操作需要 CSRF：请求头 X-CSRF-Token 必须等于 cookie pantheon_csrf_token
CSRF=$(awk '$6=="pantheon_csrf_token"{print $7}' admin.jar)
```

## 2. 验证项

**记录方式：** 每项记录“实际 HTTP 状态码、响应体前 200 字符（去敏）、与预期是否一致”。

### V1　非管理员角色对 deploy 的实际权限（验证 SUP-08 降级与权限绑定缺口）

- **准备：** 在管理控制台新建角色 `verify_deploy_viewer`，只勾选 deploy 的任务查看相关权限点（菜单和按钮里有哪些就勾哪些）；新建用户 `verify_u1` 绑定该角色，登录得到 `u1.jar`。
- **步骤：** 用 `u1.jar` 依次请求：`GET /business/deploy/tasks`、`POST /business/deploy/tasks`（空体即可）、`POST /business/deploy/tasks/1/start`、`GET /business/deploy/credentials`。
- **预期（按我的推断）：** 全部 403。
- **若出现 200：** 说明存在我没找到的授权路径，“默认拒绝、功能缺口”的结论要推翻，并且这是安全问题，应立即升级为 P1。
- **对应结论：** 权限绑定逐条核对一节；SUP-08。

### V2　软件包的导出与详情（验证 keyMatch2 的有意缺口）

- **准备：** 角色只勾选软件包的“查看”。
- **步骤：** `GET /business/deploy/packages`、`GET /business/deploy/packages/export`、`GET /business/deploy/packages/1`。
- **预期：** 列表 200；导出 403；详情 403（注释说明详情有意不绑定）。
- **若导出返回 200：** 源码注释所说的“策略被意外放宽”已发生，属于真实越权。

### V3　其余未绑定模块（k8s、cicd、filedist、bizscope）

- **准备：** 角色勾选这些模块页面和按钮的全部权限点。
- **步骤：** 各取 1 个查询接口和 1 个写接口请求，例如 k8s 的集群列表与集群创建。
- **预期：** 403，说明“勾选了但不生效”。
- **记录：** 页面上按钮是否仍然显示，点击后前端提示是什么。这决定了问题是“后端缺绑定”还是“前后端体验不一致”。

### V4　多租户隔离（验证 ARC-001）

- **准备：** 在管理控制台把 `platform.tenant_mode` 设为 `multi`；用平台账号创建租户 A、B，各带一个管理员（`POST /system/tenant`，字段是 snake_case，见 `docs/TENANT_INITIALIZATION_GUIDE.md`）。
- **步骤：**
  1. 租户 A 管理员创建一台主机、一个软件包、一个 deploy 任务。
  2. 租户 B 管理员请求 `GET /business/cmdb/hosts`、`GET /business/deploy/packages`、`GET /business/deploy/tasks`。
  3. 租户 B 管理员尝试 `GET /business/deploy/tasks/<A 的任务ID>`。
- **预期（按项目合同）：** 业务域 V1 不参与租户隔离，所以 B 可能看到 A 的数据。
- **关键：** 这里要得到的是“事实”，不是“对错”。如果 B 看不到，说明已有我没发现的隔离，要更新结论；如果看得到，就是 ARC-001 的直接证据，附上截图和响应。

### V5　凭据泄露面（验证 SEC-004 与凭据加密）

- **步骤：**
  1. 用管理员创建一个测试 SSH 凭据（用一次性私钥）。
  2. `GET /business/deploy/credentials` 与单条详情，检查响应里是否出现私钥、密码或密文字段。
  3. 在数据库里查看 `secret_encrypted` 一类字段，确认不是明文。
  4. 用另一个有 deploy 权限但不是创建者的账号请求同一凭据。
- **预期：** 响应不含明文，数据库为密文；第 4 步看是否按归属隔离（SEC-004 的核心）。

### V6　长耗时操作与 10 秒前端超时

- **步骤：** 在浏览器开发者工具的 Network 面板里，对这些操作记录耗时和前端是否提示超时：同步集群、导入大文件（软件包、主机）、创建带预览的 deploy 任务、拉取日志。
- **预期：** 超过 10 秒的接口，前端会先报超时，而后端仍在执行。
- **记录：** 超时后再次点击是否会产生重复数据，这是“用户重复提交”风险的证据。

### V7　SSH 部署与注入（验证 SEC-001、租约、重启恢复）

在 115.190.251.143 上操作，**使用专用低权限用户，不要用 root**：

- **注入探针（无害）：** 对 SEC-001 报告里列出的 5 个固定模板之一，在结构参数里填 `$(touch /tmp/verify_marker_001)`。任务运行后检查目标机是否存在该文件。预期不存在；存在则 SEC-001 的利用路径被证实，立刻停止并上报。
- **主机指纹：** 首次连接一个指纹不匹配的主机，预期被拒绝。
- **重启恢复：** 任务运行中 `kill` 后端进程再启动，观察任务状态是否被恢复为合理状态，是否出现重复执行。
- **取消：** 在命令执行中点击取消，观察目标机上进程是否被终止。

### V8　K8s 凭据轮换（验证 SUP-01）

- **准备：** 用 kind 或 k3s 起一个一次性集群，用一个 ServiceAccount 令牌生成 kubeconfig A，在平台导入集群。
- **步骤：** 持续请求该集群的工作负载列表（每 5 秒一次）；在集群里撤销 A 的令牌；同时在平台里更新为新的 kubeconfig B。
- **预期（按我的推断）：** 旧连接在池中继续使用，直到空闲超过 30 秒才失效。持续请求会让它一直不过期。
- **记录：** 令牌撤销后多久请求开始失败。

## 3. 结果回收模板

| 编号 | 实际状态码或现象 | 与预期一致？ | 证据位置 | 对结论的影响 |
| --- | --- | --- | --- | --- |
| V1 | | | | |
| V2 | | | | |
| V3 | | | | |
| V4 | | | | |
| V5 | | | | |
| V6 | | | | |
| V7 | | | | |
| V8 | | | | |

把填好的表和去敏后的原始输出发回来，我会据此更新评审报告，并把结论分为“已验证”“被推翻”“仍未验证”三类。

## 4. 清理

```bash
docker compose -f docker-compose.verify.yml down -v
rm -f admin.jar u1.jar
# 目标主机：删除验证用的目录、测试用户的授权公钥，以及 /tmp/verify_marker_001（如存在）
```
