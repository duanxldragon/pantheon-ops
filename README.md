# Pantheon Ops

English version: [README.en.md](./README.en.md)

Pantheon Ops 是基于 Pantheon Platform 底座拆出的运维管理平台，用 CMDB 和 Deploy 作为第一批业务能力，独立承载运维资源台账、分组、标签规范和部署任务管理。

该仓库保留平台底座能力作为业务运行基础，但演进重点放在 `business/cmdb`、`business/deploy` 及后续运维业务模块；通用后台能力的持续演进应回到 `pantheon-platform` 仓库。

默认协作模型已经调整为：`pantheon-ops` 消费 `pantheon-base` 的 foundation release，而不是直接跟随 `pantheon-base/main`。

## 项目定位

- **平台层**：继承 Pantheon Platform 的应用壳层、路由装配、中间件、平台工作台、跨域聚合视图。
- **系统域**：继承认证安全、用户角色权限、菜单、组织、配置、字典、审计等底座能力。
- **业务域**：当前内置 `business/cmdb` 和 `business/deploy`，作为运维管理平台的核心业务。

## 核心能力

- **认证与会话**：access/refresh token、注销失效、在线会话、登录日志。
- **IAM 与权限**：用户、角色、菜单、页面权限、操作权限、Casbin 接口策略。
- **组织管理**：部门、岗位、用户组织归属，以及组织架构视图。
- **配置治理**：系统设置、字典管理、缓存刷新、敏感配置保护。
- **审计能力**：登录日志、操作日志、关键写操作审计。
- **动态菜单**：菜单 seed、前端 manifest、组件注册表和构建期契约检查。
- **CMDB**：主机资源、分组、标签规范和一次性采集。
- **Deploy**：部署包、部署任务、目标选择和任务快照。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 后端 | Go、Gin、GORM、Casbin、JWT、MySQL、Redis |
| 前端 | React、TypeScript、Vite、Arco Design、Zustand、i18next |
| 工程 | Docker Compose、Playwright、GitHub Actions、gstack QA 流程 |

## 目录结构

```text
backend/
  cmd/server/              # 后端启动入口
  modules/auth/            # system/auth：认证、会话、安全中心
  modules/dashboard/       # platform：工作台聚合数据
  modules/system/          # system/*：IAM、组织、配置、审计等底座能力
  modules/business/        # business/*：业务域模块样例
  pkg/                     # 公共契约、数据库、响应、JWT 等
frontend/
  src/core/                # 应用壳层、路由、主题、菜单装配
  src/modules/auth/        # 认证与安全中心页面
  src/modules/dashboard/   # 平台工作台
  src/modules/system/      # 系统域管理页面
  src/modules/business/    # 业务域页面
docs/                      # 架构、权限、前端规范、验收与运维文档
database/system_init.sql   # 初始化 schema、seed、i18n
```

## 快速启动

### 1. 启动基础设施

```bash
docker compose up -d
```

默认会启动：

- MySQL: `127.0.0.1:3306`
- Redis: `127.0.0.1:6379`
- 默认数据库：`pantheon_ops`

### 2. 启动后端

PowerShell 示例：

```powershell
$env:PANTHEON_DSN='root:DHCCroot@2025@tcp(127.0.0.1:3306)/pantheon_ops?charset=utf8mb4&parseTime=True&loc=Local'
$env:PANTHEON_REDIS_ADDR='127.0.0.1:6379'
$env:PANTHEON_REDIS_PASSWORD='DHCCdhcc2025'
go run ./backend/cmd/server
```

后端默认监听 `http://127.0.0.1:8080`。
`pantheon-ops` 作为独立业务仓库，应独占 `pantheon_ops` 数据库；即使与 `pantheon-base` 共用同一 MySQL 实例，也不能共用同一个库。

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端默认监听 `http://127.0.0.1:5173`。

### 4. 默认登录

非生产环境未设置 `PANTHEON_INITIAL_ADMIN_PASSWORD` 时，后端迁移会创建开发默认账号：

```text
用户名：admin
密码：123456
```

生产环境必须在启动前设置 `PANTHEON_INITIAL_ADMIN_PASSWORD`，长度不少于 12 位。

## 常用命令

```bash
# 后端测试
go test ./backend/modules/auth ./backend/modules/system/...

# 前端构建与菜单契约检查
cd frontend
npm run build

# 系统页 smoke
npm run test:smoke:system

# 角色授权专项 smoke
npm run test:smoke:role-auth

# 导入导出 API smoke
npm run test:smoke:impexp

# 后台 UI smoke
npm run test:smoke:backoffice-ui

# 规划消费某个 foundation release
npm run upgrade:foundation:plan -- --manifest <bundle-root>\\manifest.json --bundle <bundle-root>

# 应用共享 backend/frontend、保留 ops overlay，并更新继承锚点
npm run upgrade:foundation:apply -- --manifest <bundle-root>\\manifest.json --bundle <bundle-root>
```

## 手动 Sonar

Sonar 仅作为辅助审查工具，不参与 GitHub required checks。CodeQL 负责安全主信号，Codacy 如果出现也只看作参考仪表盘。

```powershell
Set-Content pantheon-sonarcloud.env "SONAR_HOST_URL=https://sonarcloud.io`nSONAR_TOKEN=..."
./scripts/run-sonar.ps1
```

扫描结果上传后，直接在 SonarCloud 仪表盘查看热点、重复率和新代码问题。更完整的门禁策略见 `pantheon-base/docs/designs/QUALITY_AND_SECURITY_STRATEGY.md`。

## 权限模型摘要

Pantheon Platform 将权限拆成四层：

1. **导航授权**：是否能在侧边栏看到菜单，存储在 `system_role_menu`。
2. **页面授权**：是否能进入页面路由，来源于菜单元数据 `pagePerm`。
3. **操作授权**：是否能使用页面按钮或动作，来源于按钮节点 `perms`。
4. **接口授权**：后端 API 访问控制，由 Casbin 策略维护。

角色管理页已将导航、页面、操作三类授权统一为树形面板，支持搜索、全展开/全收起和父级批量勾选。

## 文档入口

- [docs/README.md](./docs/README.md)：中文主索引。
- [docs/PROJECT_INHERITANCE.md](./docs/PROJECT_INHERITANCE.md)：先看继承关系、版本锁定与本地业务范围。
- `docs/PROJECT_INHERITANCE.md` 中的 `Base release line + Base version` 是当前 consumer 版本锚点。
- `upgrade:foundation:apply` 会保留 ops 本地 menu/generator/workspace overlay，重写共享 backend import 到 `pantheon-ops`，并补跑 frontend base-sync + menu-contract。
- [.agents/skills/README.zh.md](./.agents/skills/README.zh.md)：本仓库的 repo-local Codex skills 入口。
- [DESIGN.md](./DESIGN.md)：再看仓库级设计边界。
- [CONTRIBUTING.md](./CONTRIBUTING.md) / [SECURITY.md](./SECURITY.md)：协作与安全规则。
- `pantheon-base/docs/designs/QUALITY_AND_SECURITY_STRATEGY.md`：质量与安全治理策略主文档。
- 如需英文入口，使用 [README.en.md](./README.en.md) 与 [docs/README.en.md](./docs/README.en.md)。

## 推荐阅读顺序

建议按这个顺序进入，不要跳读：

1. [README.md](./README.md)
2. [docs/README.md](./docs/README.md)
3. [docs/PROJECT_INHERITANCE.md](./docs/PROJECT_INHERITANCE.md)
4. [DESIGN.md](./DESIGN.md)
5. [AGENTS.md](./AGENTS.md)
6. 再按 `docs/README.md` 中的业务文档入口继续深入

## 提交规范

本仓库使用 Conventional Commits，格式如下：

```text
type(scope): subject
```

示例：

```text
feat(system-iam): unify role authorization trees
fix(system-org): validate post department ownership
docs(platform): improve repository README
test(system-iam): add role authorization smoke coverage
```

允许的 `type` 见 `CONTRIBUTING.md`。仓库提供 `.gitmessage` 和 `.githooks/commit-msg`，本地可通过以下命令启用：

```bash
git config commit.template .gitmessage
git config core.hooksPath .githooks
```

## GitHub 展示建议

建议在 GitHub Repository Settings 中配置：

- **Description**：Enterprise admin foundation with modular monolith, IAM, dynamic menus, i18n and audit.
- **Website**：如暂无线上环境，可暂留空。
- **Topics**：`go`、`gin`、`gorm`、`react`、`typescript`、`vite`、`arco-design`、`casbin`、`iam`、`admin-dashboard`、`modular-monolith`、`enterprise-platform`
- **Features**：启用 Issues、Pull Requests、Actions；如暂不开放协作，可关闭 Wiki。
- **Community Files**：仓库已补齐 `README`、`CONTRIBUTING`、`SECURITY`、Issue Templates 和 PR Template。

## 安全提示

- 不要把 GitHub 密码、Token、生产数据库 DSN、生产 Redis 密码写入代码、README 或 Git remote。
- GitHub 已不支持账号密码推送，请使用 GitHub CLI、Credential Manager 或 Personal Access Token 认证。
- `.env`、本地数据库、日志、构建产物和可执行文件已在 `.gitignore` 中排除。
