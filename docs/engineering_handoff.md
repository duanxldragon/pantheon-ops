# 工程师实现交接单 — pantheon-base v0.9.0 module 重命名 + 三项门禁

- **发起人**: Bob (Architect)
- **接手人**: 寇豆码 (Engineer)
- **协调人**: team-lead（齐活林）
- **审阅人**: 小龙（维护者，已签批）
- **日期**: 2026-07-15
- **仓库根**: `D:\workspace\go\pantheon-platform\pantheon-base`

> 本文档是实现的**唯一权威依据**。任何与本单冲突的理解，以本单为准。详细设计请同时参考 `docs/system_design.md`。

---

## 一、目标速览

| 类别 | 目标 |
|---|---|
| **主目标** | Go module 从 `pantheon-platform` 重命名为 `pantheon-base`，覆盖 142 个 .go 文件 339 处 import + 5 处生成器模板 + 4 处测试断言 + 3 处脚本 |
| **副目标 1** | business/* 边界门禁：check-boundaries.mjs 接 CI --strict + golangci-lint depguard |
| **副目标 2** | docs/DEPLOYMENT_GUIDE.md 新增"生产强制安全基线"章节（含 mfa_enabled=1） |
| **副目标 3** | ci.yml 新增 coverage-gate job，初始阈值 = 现状 90%，ratchet 至 80% |

---

## 二、维护者已拍板的 7 项决策（必须遵守）

| # | 决策 |
|---|---|
| Q1 | `backend/DEV_DB_INIT_GUIDE.md` 里 5 处 `pantheon-platform` **一并改** |
| Q2 | `docs/designs/**` 仅改**代码示例**中的 import；叙述性历史引用保留 |
| Q3 | depguard `files` glob **先 POC**，失败则改用"全 backend deny 反向规则" |
| Q4 | 覆盖率初始阈值 = **现状实测值的 90%**（先跑一次 `go test -coverprofile` 测出来） |
| Q5 | base CI 只扫自己，`check-boundaries.mjs` 加 `--repo pantheon-base` 参数 |
| Q6 | commit 按任务单 commit：T02/T03/T04/T05 各一个原子 commit |
| Q7 | 彻底切割，**不留 go replace 兜底** |

---

## 三、铁律约束（违反 = 返工）

### 铁律 1 — module 名与 import 路径
- 新 module 名统一 `pantheon-base`
- import **永不带 `backend/` 段**（go.mod 已在 backend/ 内）
- ✅ `"pantheon-base/pkg/common"` / ❌ `"pantheon-base/backend/pkg/common"`

### 铁律 2 — sed 替换模式
- 仅匹配**带前导引号的字符串字面量**：`"pantheon-platform/` → `"pantheon-base/`
- **不要**替换裸标识符、注释中的物理路径（`D:\workspace\go\pantheon-platform\...`）
- **不要**触碰 `pantheon-platform` 单独出现（不带 `/` 后缀）的叙述文本

### 铁律 3 — 豁免清单（不动）
- `.harness/evidence/**` 历史存证
- `pantheon-ops/.foundation/releases/base-v*/**` ops 已锁定 artifact
- `docs/**/*.md` 中 `D:\workspace\go\pantheon-platform\...` 物理路径引用
- `docs/designs/**` 叙述性历史引用（按 Q2 只改代码示例）

### 铁律 4 — 验证三件套（每任务完成必跑）
```bash
cd backend && go build ./... && go vet ./... && go test -race -short ./...
cd frontend && npx tsc --noEmit
bash scripts/maintenance/verify-module-rename.sh   # T01 产出后存在
```

### 铁律 5 — commit 策略
| Commit | 内容 |
|---|---|
| T01 | `chore(scripts): add module rename + verify scripts and design docs` |
| T02 | `chore(backend): rename go module pantheon-platform → pantheon-base` |
| T03 | `chore(frontend,scripts): sync generator templates, tests, cleanup/drift with module rename` |
| T04 | `ci: add boundary-gate + coverage-gate, depguard, MFA baseline doc` |
| T05 | `chore(release): v0.9.0 freeze` |

---

## 四、任务清单（按依赖顺序执行）

### T01 — 重命名基础设施（0.5d，P0）

**新建文件**：
| 文件 | 内容 |
|---|---|
| `scripts/maintenance/rename-module.sh` | sed 编排脚本，支持 `--dry-run` 预览 |
| `scripts/maintenance/verify-module-rename.sh` | 验证脚本：grep 残留，输出豁免白名单外的命中数 |
| `scripts/maintenance/README.md` | 用法说明（dry-run 流程 + 回退方法） |

**已存在（无需新建）**：`docs/system_design.md` / `docs/class-diagram.mermaid` / `docs/sequence-diagram.mermaid`

**`rename-module.sh` 关键 sed 命令骨架**：
```bash
#!/usr/bin/env bash
set -euo pipefail
DRY_RUN=${1:-}
SED_INPLACE=(-i)
[[ "$DRY_RUN" == "--dry-run" ]] && SED_INPLACE=(-n p)

# 1. Go 源码 + go.mod（仅替换带前导引号字面量）
find backend -type f -name '*.go' -print0 | xargs -0 sed "${SED_INPLACE[@]}" \
  -e 's|"pantheon-platform/|"pantheon-base/|g'
sed "${SED_INPLACE[@]}" -e '1s|^module pantheon-platform$|module pantheon-base|' backend/go.mod
sed "${SED_INPLACE[@]}" -e 's|pantheon-platform/|pantheon-base/|g' backend/DEV_DB_INIT_GUIDE.md

# 2. 生成器模板（5 处）
sed "${SED_INPLACE[@]}" -e 's|"pantheon-platform/|"pantheon-base/|g' \
  frontend/src/modules/lowcode/generator/backendGenerator.ts

# 3. smoke 断言（4 处，模板字符串带反引号，所以模式匹配 pantheon-platform/）
sed "${SED_INPLACE[@]}" -e 's|pantheon-platform/modules/business/|pantheon-base/modules/business/|g' \
  frontend/tests/smoke/business/generated/module-governance-host-real.spec.ts \
  frontend/tests/smoke/business/generated/module-governance-real.spec.ts

# 4. cleanup 正则（注意：旧正则是 pantheon-platform/backend/modules/business → 新的是 pantheon-base/modules/business，去掉 backend/）
sed "${SED_INPLACE[@]}" \
  -e 's|pantheon-platform\\/backend\\/modules\\/business\\/|pantheon-base\\/modules\\/business\\/|g' \
  frontend/scripts/cleanup-generated-modules.mjs

# 5. cleanup 测试 fixture（fixture 里写的是带 backend/ 的旧旧形态，需同步规范化）
sed "${SED_INPLACE[@]}" \
  -e 's|pantheon-platform/backend/modules/business/mdqaorder|pantheon-base/modules/business/mdqaorder|g' \
  frontend/scripts/cleanup-generated-modules.test.mjs

# 6. drift 脚本归一化
sed "${SED_INPLACE[@]}" \
  -e "s|replaceAll('pantheon-platform/backend', 'MODNAME/backend')|replaceAll('pantheon-base/backend', 'MODNAME/backend')|" \
  scripts/harness/triage-base-drift.mjs
```

**`verify-module-rename.sh` 关键逻辑**：
```bash
#!/usr/bin/env bash
set -euo pipefail
# 扫描所有应已迁移的位置；命中非豁免即 exit 1
hits=$(grep -rIn --include='*.go' --include='*.ts' --include='*.tsx' --include='*.mjs' \
       'pantheon-platform/' backend/ frontend/src/ frontend/scripts/ frontend/tests/ scripts/harness/ \
       | grep -v 'pantheon-platform/backend' || true)
# 期望只有 pantheon-platform/backend 这种物理路径残留（在 docs/，已豁免）
if [[ -n "$hits" ]]; then
  echo "FAIL: residual pantheon-platform/ imports:"
  echo "$hits"
  exit 1
fi
echo "OK: no residual pantheon-platform/ imports outside allowlist"
```

**Acceptance**：
- [ ] `bash scripts/maintenance/rename-module.sh --dry-run` 输出影响文件清单
- [ ] `bash scripts/maintenance/verify-module-rename.sh` 在改动前能识别出 339+ 处待改（exit 1），改动后 exit 0

---

### T02 — Go module 声明 + 源码批量改写（0.5d，P0）

**前置**：T01 完成

**改动**：
- `backend/go.mod` L1：`module pantheon-platform` → `module pantheon-base`
- 142 个 `.go` 文件 / 339 处 import（通过 `rename-module.sh` 第 1 步 sed 完成）
- `backend/DEV_DB_INIT_GUIDE.md` 5 处（Q1 决策）

**执行步骤**：
```bash
cd D:/workspace/go/pantheon-platform/pantheon-base
git checkout -b fix/v090-module-rename
bash scripts/maintenance/rename-module.sh   # 非 dry-run 真改
cd backend && gofmt -l .                    # 应无输出
cd backend && go mod tidy
cd backend && go build ./... && go vet ./... && go test -race -short ./...
```

**Acceptance**：
- [ ] 上述命令全绿
- [ ] `grep -rIn '"pantheon-platform/' backend/` 0 命中
- [ ] 单 commit 提交，commit message 按铁律 5

---

### T03 — 生成器模板 + 测试断言 + 脚本同步（0.5d，P0）

**前置**：T02 完成

**改动**（已由 `rename-module.sh` 第 2-6 步处理）：
| 文件 | 行号 | 改动 |
|---|---|---|
| `frontend/src/modules/lowcode/generator/backendGenerator.ts` | 283, 284, 575, 712, 713 | 5 处模板字符串 |
| `frontend/tests/smoke/business/generated/module-governance-host-real.spec.ts` | 329, 392 | 断言 |
| `frontend/tests/smoke/business/generated/module-governance-real.spec.ts` | 178, 218 | 断言 |
| `frontend/scripts/cleanup-generated-modules.mjs` | 192 | 正则 `pantheon-platform/backend/modules/business` → `pantheon-base/modules/business` |
| `frontend/scripts/cleanup-generated-modules.test.mjs` | 69, 114 | fixture 字符串同步规范化 |
| `scripts/harness/triage-base-drift.mjs` | 162 | `.replaceAll('pantheon-platform/backend', ...)` → `.replaceAll('pantheon-base/backend', ...)` |

**验证**：
```bash
cd frontend && npx tsc --noEmit
node frontend/scripts/cleanup-generated-modules.test.mjs
bash scripts/maintenance/verify-module-rename.sh
```

**Acceptance**：全绿后单 commit。

---

### T04 — 三项门禁接入（1d，P1）

**前置**：T02 完成（depguard 规则需要新 module 名）

#### T04a — depguard POC + 落地

**POC 步骤**：
1. 编辑 `backend/.golangci.yml`，在 `linters.enable` 加 `- depguard`
2. 在 `linters.settings` 加：
   ```yaml
   depguard:
     rules:
       business-boundary:
         files:
           - "modules/business/**.go"
         deny:
           - pkg: pantheon-base/modules/system
             desc: use pantheon-base/pkg/contracts instead
           - pkg: pantheon-base/modules/auth
             desc: use pantheon-base/pkg/contracts instead
           - pkg: pantheon-base/modules/platform
             desc: use pantheon-base/pkg/contracts instead
   ```
3. 手动在 `backend/modules/business/business.go` 临时加 `import _ "pantheon-base/modules/system/iam/user"`，跑 `golangci-lint run ./...`，应被拦截。
4. 撤回临时 import。

**如果 POC 失败**（files glob 不生效），改用反向规则：
```yaml
depguard:
  rules:
    no-business-cross-import:
      files:
        - $all
      deny:
        - pkg: pantheon-base/modules/business
          desc: business modules are entry-only, not for cross-import
```
（即只允许 business 被 cmd/server 引用，不允许反过来）

#### T04b — check-boundaries.mjs 加 `--repo` 参数 + 接入 CI

在 `scripts/harness/check-boundaries.mjs`：
- `parseArgs` 加 `--repo <name>` 处理
- `REPOSITORIES` 改为：根据 `--repo` 决定，默认全扫
- `main()` 用过滤后的 repo 列表

`.github/workflows/ci.yml` 新增 job：
```yaml
boundary-gate:
  name: Boundary Gate
  runs-on: ubuntu-latest
  timeout-minutes: 3
  permissions:
    contents: read
  steps:
    - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1
      with: { persist-credentials: false }
    - uses: actions/setup-node@820762786026740c76f36085b0efc47a31fe5020
      with: { node-version: ${{ env.NODE_VERSION }} }
    - run: node scripts/harness/check-boundaries.mjs --strict --repo pantheon-base
```

并把 `boundary-gate` 加入 `ci-summary.needs`。

#### T04c — 覆盖率门禁

**新建 `scripts/harness/check-coverage.mjs`**：
- 解析 `go tool cover -func` 输出（每行格式：`path/file.go:LINE:\tFUNC\tCOVER%`）
- 按包分组求平均（pantheon-base/modules/system/...）
- 总覆盖率 + 各包覆盖率 < threshold 则 exit 1
- CLI：`node check-coverage.mjs coverage.txt --threshold <number>`

**先实测现状**（在 T04 开始前跑）：
```bash
cd backend && go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out | tail -5   # 看 total 行
```
**初始阈值 = 实测值 × 0.9**（取整到小数点后 1 位）。

`.github/workflows/ci.yml` 新增 job：
```yaml
coverage-gate:
  name: Coverage Gate
  runs-on: ubuntu-latest
  needs: [unit-tests]
  timeout-minutes: 10
  permissions:
    contents: read
  steps:
    - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1
      with: { persist-credentials: false }
    - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e
      with: { go-version-file: ${{ env.GO_VERSION_FILE }}, cache: true }
    - uses: actions/setup-node@820762786026740c76f36085b0efc47a31fe5020
      with: { node-version: ${{ env.NODE_VERSION }} }
    - working-directory: backend
      run: |
        go test -coverprofile=coverage.out -covermode=atomic ./...
        go tool cover -func=coverage.out > coverage.txt
    - run: node scripts/harness/check-coverage.mjs backend/coverage.txt --threshold ${COVERAGE_THRESHOLD:-60}
```

#### T04d — MFA 基线文档

在 `docs/DEPLOYMENT_GUIDE.md` 的 §3（生产环境配置）新增 §3.1 "强制安全基线"，参考 fix-report 第四节：

```markdown
### 3.1 强制安全基线（生产必选）

| 配置项 | 必须值 | 说明 |
|---|---|---|
| `login.mfa_enabled` | `1` | 强制双因子认证，不可降级 |
| `session.secure_cookie` | `true` | Cookie 仅 HTTPS |
| `csrf.enabled` | `true` | CSRF 防护 |
| `audit.enabled` | `true` | 审计日志 |
| `rate_limit.enabled` | `true` | 全局限流 |

部署完成后，运维必须通过管理后台 → 安全设置确认上述开关状态为"启用"。
```

**Acceptance**：
- [ ] POC 验证 depguard 生效或兜底方案落地
- [ ] `node scripts/harness/check-boundaries.mjs --strict --repo pantheon-base` exit 0
- [ ] `node scripts/harness/check-coverage.mjs backend/coverage.txt --threshold <实测值×0.9>` exit 0
- [ ] `cd backend && golangci-lint run ./...` 通过
- [ ] ci.yml 通过 yamllint / actionlint（如已配置）
- [ ] docs/DEPLOYMENT_GUIDE.md 含 §3.1 章节
- [ ] 单 commit 提交

---

### T05 — 端到端验证 + 冻结 v0.9.0（0.5d，P0）

**前置**：T03 + T04 完成

**改动**：
| 文件 | 改动 |
|---|---|
| `VERSION` | 改为 `0.9.0`（如当前不是） |
| `CHANGELOG.md` | 新增 v0.9.0 entry：module 重命名 + 三项门禁 |
| `.harness/evidence/v090-module-rename.md` | 新建存证文档，含：执行命令、验证输出、commit hash、签批人 |

**验证清单**：
```bash
bash scripts/maintenance/verify-module-rename.sh   # exit 0
cd backend && go build ./... && go vet ./... && go test -race -short ./...
cd frontend && npx tsc --noEmit && npx eslint src --max-warnings 0
node frontend/scripts/cleanup-generated-modules.test.mjs
node scripts/harness/check-boundaries.mjs --strict --repo pantheon-base
# 推送 release 分支，观察 GitHub Actions 全绿
```

**Acceptance**：维护者 sign-off，打 `v0.9.0` tag。

---

## 五、关键文件路径速查

| 类别 | 路径 |
|---|---|
| go.mod | `backend/go.mod` |
| 入口 | `backend/cmd/server/main.go` |
| 生成器模板 | `frontend/src/modules/lowcode/generator/backendGenerator.ts` |
| smoke 断言 | `frontend/tests/smoke/business/generated/module-governance-{host-}real.spec.ts` |
| 清理脚本 | `frontend/scripts/cleanup-generated-modules.mjs` + `.test.mjs` |
| drift 脚本 | `scripts/harness/triage-base-drift.mjs` |
| 边界扫描 | `scripts/harness/check-boundaries.mjs` |
| Lint 配置 | `backend/.golangci.yml` |
| CI 配置 | `.github/workflows/ci.yml` |
| 部署文档 | `docs/DEPLOYMENT_GUIDE.md` |

---

## 六、风险与回退

| 风险 | 缓解 |
|---|---|
| sed 误伤 docs/ 物理路径 | 替换模式带前导引号，物理路径不带引号不会命中 |
| depguard glob 不生效 | T04a POC 先行，兜底反向规则 |
| 覆盖率现状 < 60% | 按 Q4 取现状 90% 作为初始阈值，不强求 60% |
| 需整体回退 | 每任务单 commit，`git revert <sha>` 即可 |
| ops 仓库受影响 | ops 锁定 `base-v0.8.x` artifact 不动；v0.9.0 切新版本线 |

---

## 七、联系方式

- **设计疑问** → Bob (Architect) via team-lead
- **决策疑问** → 小龙（维护者）via team-lead
- **进度同步** → team-lead（齐活林）

**祝顺利。**
