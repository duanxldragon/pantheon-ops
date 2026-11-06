# 下线计划：Legacy 认证双入口（2026-12-31）

English note: this is a **ready-to-execute teardown plan**, deliberately not executed.
`SYSTEM_AUTH_CONTRACT.md` §7.7 promises four `/system/*` self-service endpoints stay
available until 2026-12-31; removing them early is a breaking change for any external
integrator, so the removal is gated on a maintainer confirmation that no such
integrator exists. This document is the verified inventory and the exact edit list so
that, once that confirmation exists, the removal is mechanical rather than an
investigation.

**状态：未执行。** 本文只是"当天照着做"的执行单，不含任何已落地的删除。

---

## 1. 为什么要等，以及等的是什么

`SYSTEM_AUTH_CONTRACT.md` §7.7 把四个端点的下线日期写成 **2026-12-31**，并在约束里写明
"下线动作属对外承诺变更，执行前需维护者确认无外部集成方"。

所以门禁不是技术性的，是**事实性的**：是否有人在 2026-12-31 之前仍调用这四个路径。
本文无法回答这个问题，也不能代替回答。可执行的准备工作只有两件：

1. 把"要删什么"清点到一个不依赖记忆的清单里（§2）；
2. 把"删完怎么证明删干净了"写成可判定的检查（§5）。

---

## 2. 待下线面清点（实测，含文件与行号）

### 2.1 路由（4 条，两个 group）

| 路由 | 注册位置 | 备注 |
|------|----------|------|
| `POST /api/v1/system/login` | `backend/modules/auth/module.go:73` | 在 `sys := r.Group("/system")`（`:71`）下 |
| `POST /api/v1/system/refresh` | `backend/modules/auth/module.go:74` | 同上 |
| `POST /api/v1/system/logout` | `backend/modules/auth/module.go:97` | 在 `systemProtected`（`:94`）下 |
| `GET /api/v1/system/user/info` | `backend/modules/auth/module.go:99` | 同上 |

两个 group 都需要处理：

- `sys`（`:71-75`）只承载这 2 条 legacy 路由，删掉路由后**整个 group 变空**，应一并删除。
- `systemProtected`（`:94-115`）还承载 15 条**当前**接口（`/profile/password` 1 条、
  `/login-log/*` 4 条、`/security-event/*` 5 条、`/session/*` 5 条），**只删 `:97` 与 `:99` 两行**，group 保留。

### 2.2 弃用响应头机制（整体删除）

| 符号 | 位置 | 处置 |
|------|------|------|
| `legacySelfServicePaths` | `backend/internal/middleware/csrf_middleware.go:13-18` | 删（仅被下面两者使用） |
| `legacySunsetHTTPDate` | `csrf_middleware.go:20` | 删 |
| `LegacyDeprecationMiddleware()` | `csrf_middleware.go:24-32` | 删 |
| 挂载点 1 | `modules/auth/module.go:71` | 随 `sys` group 一起删 |
| 挂载点 2 | `modules/auth/module.go:94` | 从 `.Use(...)` 链上摘掉这一环，其余中间件保留 |

删完后 `grep -rn "LegacyDeprecation\|legacySunsetHTTPDate\|legacySelfServicePaths" backend/`
必须为空——这是 §5 的第 1 项检查。

### 2.3 CSRF 精确豁免（2 条）

`csrf_middleware.go:46-48`：

```go
// legacy 双入口（与 /auth/* 同 handler），保留至旧路由下线。
"/api/v1/system/login":   true,
"/api/v1/system/refresh": true,
```

**只删这 2 行与注释**。同 map 里 `/api/v1/auth/login`、`/auth/refresh`、`/auth/mfa/verify`
是当前接口的豁免，保留。

合同 §7.7 的原话是"不得留下无对应路由的豁免"——这正是本条的存在理由，也是 §5 第 2 项检查。

### 2.4 Casbin 自助路由 case（2 条）

`backend/internal/middleware/casbin_middleware.go` 的 `isSelfServiceRouteBySignature`：

| case | 行号 | 处置 |
|------|------|------|
| `"/api/v1/system/logout"` → `method == "POST"` | `:97-98` | 删 |
| `"/api/v1/system/user/info"` → `method == "GET"` | `:103-104` | 删 |

同 switch 里 `/api/v1/auth/logout`、`/auth/me` 等是当前接口，保留。
`/api/v1/system/profile` 与 `/system/profile/password` 属**当前**接口（`module.go:100`），
**不是** legacy 面，保留。

### 2.5 受影响的测试（1 个文件，3 处）

`backend/internal/middleware/casbin_middleware_test.go`：

| 行 | 内容 | 处置 |
|----|------|------|
| `:26` | `{name: "system logout", fullPath: "/api/v1/system/logout", ...}` | 删 |
| `:29` | `{name: "system user info", fullPath: "/api/v1/system/user/info", ...}` | 删 |
| `:42` | `{name: "wrong method", fullPath: "/api/v1/system/logout", method: GET, want: false}` | 删；该用例测的是"方法不匹配则不豁免"，删除后同一意图由 `/auth/logout` 的等价用例承担（若无则补一条） |

### 2.6 需要同步的文档（6 个文件）

| 文件 | 行 | 需要做什么 |
|------|----|-----------|
| `docs/contracts/SYSTEM_AUTH_CONTRACT.md` | `197-215`（§7.7 全节） | 改为"已于 2026-12-31 下线"的终态记录，保留表格作为历史映射；删掉"保留至下线日"的三条约束 |
| `docs/contracts/SYSTEM_AUTH_CONTRACT.en.md` | `174-185` | 同上（英文镜像） |
| `docs/designs/BACKEND.md` | `61`、`79-83` | `61` 的"兼容旧路径"列表删掉 4 项；`79-83` 路由表的 4 行删除 |
| `docs/designs/BACKEND.en.md` | `64` | 同 `61` |
| `docs/TENANT_INITIALIZATION_GUIDE.md` | `659-664` | 两处 `POST /api/v1/system/login` 示例改为 `/api/v1/auth/login` |

**`docs/TENANT_INITIALIZATION_GUIDE.md` 是容易漏的一处**：它不在合同体系内，用 legacy 路径写
登录示例。若只改合同不改它，下线后按这份指南操作的人会直接拿到 404。

### 2.7 明确不在本次范围内的东西

| 路径 | 为什么保留 |
|------|-----------|
| `/api/v1/system/login-log/*`（4 条） | 当前接口，不是 legacy 面 |
| `/api/v1/system/security-event/*`（5 条） | 同上 |
| `/api/v1/system/session/*`（5 条） | 同上 |
| `/api/v1/system/profile/password` | 当前接口 |
| `/api/v1/system/refresh/state` | 与 `/system/refresh` **无关**的当前接口；CSRF 豁免是精确匹配，不覆盖它 |

---

## 3. 前置条件（缺一不可）

- [ ] 日期 ≥ **2026-12-31**（合同承诺的对外有效期已满）
- [ ] 维护者已确认无外部集成方仍调用这四条路径，并留下书面确认
- [ ] 已完成一次调用量观测：网关/访问日志中四条路径近 30 天命中数为 0（或命中方已迁移）
      —— 这是把"确认"从声明变成证据的唯一手段
- [ ] base 与 ops 前端均只调用 `/auth/*`（实测见 §4）

## 4. 前端的现状（已核实，无需改动）

`grep -rn "system/login\|system/refresh\|system/logout\|system/user/info" frontend/src/` 的
命中全部是 `/system/login-log*`（当前接口）与 `/system/refresh/state`（另一条当前接口），
**没有一处调用这四条 legacy 路由**。因此本次下线不涉及任何前端改动。

注意 `refresh/state` 与 `refresh` 只差一段路径：CSRF 豁免用精确匹配
（`csrfExemptPaths[path]`，`csrf_middleware.go:52`）而非前缀匹配，所以删掉 `/system/refresh`
的豁免不会影响 `/system/refresh/state`。这一点已由 §2.3 的设计注释说明，本次不改动该设计。

---

## 5. 执行步骤（当天，按序）

### 5.1 代码删除

按 §2.1–§2.5 逐项删除。顺序建议：先删路由（§2.1），再删中间件（§2.2），
再删豁免与 case（§2.3/§2.4），最后改测试（§2.5）。先删路由会让"还有谁在用"立刻编译期/运行期暴露。

### 5.2 增加下线锁（新增，防止回潮）

在 `backend/modules/auth/` 下新增一条路由表断言，锁定"这四条不再注册"：

```go
// 断言 router.Routes() 中不存在 /api/v1/system/{login,refresh,logout,user/info}
```

理由：当前**没有任何测试**覆盖这四条路由或它们的 Deprecation 头（实测：
`grep -rn "Deprecation\|Sunset" --include=*_test.go backend/` 为空）。也就是说，
今天如果有人误删路由，测试不会红；下线后如果有人误加回来，测试同样不会红。
删掉之后补一条"不存在"的断言，成本极低，且正好补上这个空白。

> 不为 Deprecation 头补测试：该头随路由一起消失，为一个 60 天后删除的表面加测试不划算。
> 这是本计划唯一一处主动放弃的覆盖，理由是它保护的对象即将不存在。

### 5.3 文档同步

按 §2.6 改 6 个文件。合同 §7.7 从"待下线"改成"已下线"，并保留原表格作为历史映射
（删除映射关系会让读者无法从旧路径找到新路径）。

### 5.4 验证（全部必须通过）

```bash
cd backend
gofmt -l . && go vet ./... && go build ./...
go test ./modules/auth/... ./internal/middleware/... -count=1
```

静态残留检查（应为空）：

```bash
grep -rn "LegacyDeprecation\|legacySunsetHTTPDate\|legacySelfServicePaths" backend/
grep -rn '"/api/v1/system/login"\|"/api/v1/system/refresh"\|"/api/v1/system/logout"\|"/api/v1/system/user/info"' backend/
```

文档残留检查（只应剩下合同里的历史映射表）：

```bash
grep -rnE "api/v1/system/(login|refresh|logout|user/info)([^-a-z]|$)" docs/ | grep -v "login-log"
```

（`\b` 不行：它会在 `login` 与 `login-log` 的连字符之间也匹配，把当前接口一起算进来。
上面的写法用 `[^-a-z]` 排除 `login-log`。此模式仍会命中 `/system/refresh/state`——
该路径与 legacy 面无关，若出现在输出里是预期的，不是漏改。）

运行时检查（对测试环境起服务后）：

```bash
for p in login refresh logout user/info; do
  curl -s -o /dev/null -w "%{http_code} $p\n" -X POST "http://localhost:8080/api/v1/system/$p"
done
# 期望：四条全部 404，且响应头中不再出现 Deprecation / Sunset
```

**404 而不是 410**：gin 未注册的路由返回 404。若希望对外表达"曾经存在、已永久移除"，
可在 `NoRoute` 里为这四条返回 410 + 指向新路径的提示。这属于额外行为，本计划不默认做，
需要的话由维护者决定（`backend/pkg/database/notfound.go` 附近有既有的 notfound 处理可参考）。

### 5.5 证据归档

- 执行前后的 `curl` 输出
- 两条静态残留检查的空输出
- `go build` / `go test` 结果
- 维护者的书面确认（§3 第 2 项）与观测数据（§3 第 3 项）

---

## 6. 风险与回退

| 风险 | 表现 | 回退 |
|------|------|------|
| 仍有外部集成方 | 该方开始 404 | 重新加回 4 条路由（同 handler，一条 commit 即可恢复），但**不要**恢复 Deprecation 头以外的任何行为差异——两套入口指向同一 handler，行为本来就等价 |
| 漏删某处豁免 | 残留豁免指向不存在的路由 | §5.4 的 grep 会抓到；合同 §7.7 明确禁止这种残留 |
| 文档漏改 | 按 `TENANT_INITIALIZATION_GUIDE` 操作的人拿到 404 | §2.6 已逐条列出，含行号 |

回退成本极低（路由与 handler 都是一行注册），这是"可以放心执行"的前提；真正的不可逆部分
是**对外承诺**，不是代码。
