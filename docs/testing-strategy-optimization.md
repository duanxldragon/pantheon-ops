# Pantheon-Base 测试体系优化方案

## 一、现状分析

### 1.1 当前测试分层

| 测试层级 | 数量 | 工具 | 执行位置 | 平均耗时 |
|---------|------|------|---------|---------|
| **后端单元测试** | 86个 `*_test.go` | Go test + testify | `ci.yml: unit-tests` | ~2分钟 |
| **前端单元测试** | Vitest + node:test | Vitest | `ci.yml: frontend-unit-tests` | ~3分钟 |
| **冒烟测试** | 28个 `*.spec.ts` | Playwright | `smoke-full.yml` | **~120分钟** ⚠️ |
| **E2E测试** | 包含在冒烟测试中 | Playwright | 同上 | - |

### 1.2 核心痛点

```yaml
# 问题 1: 冒烟测试时间过长
smoke-full.yml:
  timeout-minutes: 120  # 2小时！
  trigger: 
    - push to main      # 每次主分支提交都跑
    - manual dispatch
  
# 问题 2: PR 流程无差异化
ci.yml:
  - 不包含冒烟测试
  - 只有快速检查 + 单元测试
  - 缺少关键路径覆盖
  
# 问题 3: 测试触发策略单一
- 每次 PR 机械跑相同的测试集
- 没有基于变更范围的智能触发
- 关键路径未被 PR 门禁保护
```

### 1.3 冒烟测试结构分析

```bash
frontend/tests/smoke/
├── platform/     # 平台层 (~10个spec)
│   ├── shell-visual-contract.spec.ts
│   ├── platform-shell.spec.ts
│   ├── full-system-pages.spec.ts  # 全页面扫描 ⚠️
│   └── ...
├── system/       # 系统模块 (~12个spec)
│   ├── system-pages.spec.ts
│   ├── system-form-state-matrix.spec.ts
│   ├── role-authorization.spec.ts
│   └── governance/  # 治理子套件
└── business/     # 业务模块 (~6个spec)
    └── generated/  # 生成模块测试

# 耗时分布估算 (基于120分钟总时长)
- 环境启动: MySQL + Redis + Backend + Frontend ~5分钟
- platform 套件: ~30分钟 (包含全页面审计)
- system 套件: ~50分钟 (包含表单状态矩阵、角色授权)
- business 套件: ~30分钟 (包含数据库导入、主从表、多对多)
- 清理 + 上传: ~5分钟
```

---

## 二、优化策略：分层 + 差异化触发

### 2.1 核心原则

```
快速反馈环 (Fast Feedback Loop):
  PR 阶段 → 快速门禁 (5-10分钟)
    ↓
  Merge 后 → 中等覆盖 (20-30分钟)
    ↓
  定时/发布前 → 完整回归 (120分钟)
```

### 2.2 三层测试金字塔

```
        ┌─────────────────┐
        │   E2E (Full)    │  120分钟 - 每日/发布前
        │   28个完整场景    │  
        └─────────────────┘
              ▲
              │
        ┌─────────────────┐
        │ Smoke (Core)    │  20分钟 - PR合并后
        │ 关键路径 8-10个   │  
        └─────────────────┘
              ▲
              │
    ┌───────────────────────┐
    │   Unit + Integration  │  5分钟 - 每次PR
    │   快速反馈 150+个测试   │
    └───────────────────────┘
```

---

## 三、具体实施方案

### 3.1 Phase 1: 拆分冒烟测试套件

#### 3.1.1 定义核心冒烟套件 (Critical Path Smoke)

**目标**: 20分钟内覆盖最高价值路径

**筛选标准**:
- ✅ 影响所有用户的功能（登录、Shell、菜单）
- ✅ 数据破坏风险高的操作（删除、批量操作）
- ✅ 历史上高频回归的区域
- ❌ 边缘场景
- ❌ 特性专项测试

**建议核心套件** (8-10个spec):

```typescript
// frontend/tests/smoke-core/ (新建目录)
smoke-core/
├── auth-critical.spec.ts           // 登录+登出+token刷新
├── platform-shell-critical.spec.ts // Shell结构+菜单导航
├── system-user-crud.spec.ts        // 用户CRUD关键路径
├── system-role-authz.spec.ts       // 角色授权基础场景
├── system-dept-tree.spec.ts        // 部门树操作
├── system-menu-permission.spec.ts  // 菜单权限联动
├── business-generated-basic.spec.ts // 生成模块基础CRUD
└── api-import-export-basic.spec.ts // 导入导出核心场景
```

#### 3.1.2 package.json 新增脚本

```json
{
  "scripts": {
    "test:smoke:core": "node scripts/run-smoke-suite.mjs --host 127.0.0.1 --port 5173 --cleanup-fixtures all --config playwright.config.ts -- tests/smoke-core/*.spec.ts --workers=2",
    "test:smoke:full": "npm run test:smoke:all",
    "test:smoke:pr": "npm run test:smoke:core"
  }
}
```

### 3.2 Phase 2: 重构 GitHub Workflows

#### 3.2.1 新增 `smoke-core.yml` (PR合并后运行)

```yaml
name: Core Smoke Tests

on:
  push:
    branches:
      - main
      - release/**
  workflow_dispatch:

concurrency:
  group: smoke-core-${{ github.ref }}
  cancel-in-progress: true

env:
  NODE_VERSION: "24"
  GO_VERSION_FILE: backend/go.mod

jobs:
  smoke-core:
    name: Core Smoke
    runs-on: ubuntu-latest
    timeout-minutes: 30  # 从120分钟降到30分钟
    services:
      mysql:
        image: mysql:8.0
        env:
          MYSQL_ROOT_PASSWORD: root
          MYSQL_DATABASE: pantheon_base
        ports:
          - 3306:3306
        options: >-
          --health-cmd="mysqladmin ping -h 127.0.0.1 -uroot -proot"
          --health-interval=10s
          --health-timeout=5s
          --health-retries=12
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd="redis-cli ping"
          --health-interval=10s
          --health-timeout=5s
          --health-retries=12
    env:
      PANTHEON_DSN: "root:root@tcp(127.0.0.1:3306)/pantheon_base?charset=utf8mb4&parseTime=True&loc=UTC"
      PANTHEON_REDIS_ADDR: 127.0.0.1:6379
      PANTHEON_PORT: "8080"
      PANTHEON_API_BASE_URL: http://127.0.0.1:8080/api/v1
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          persist-credentials: false

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version-file: ${{ env.GO_VERSION_FILE }}
          cache: true

      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: npm
          cache-dependency-path: frontend/package-lock.json

      - name: Install frontend dependencies
        working-directory: frontend
        run: |
          npm ci --ignore-scripts
          npm run patch:arco-react19

      - name: Install Playwright browsers
        working-directory: frontend
        run: ./node_modules/.bin/playwright install --with-deps chromium

      - name: Start backend
        working-directory: backend
        run: |
          nohup go run ./cmd/server >"$RUNNER_TEMP/pantheon-backend.log" 2>&1 &
          echo $! >"$RUNNER_TEMP/pantheon-backend.pid"

      - name: Wait for backend health
        run: |
          for attempt in $(seq 1 60); do
            if curl -fsS http://127.0.0.1:8080/api/v1/health >/dev/null; then
              echo "Backend is healthy"
              exit 0
            fi
            sleep 2
          done
          echo "Backend did not become healthy" >&2
          cat "$RUNNER_TEMP/pantheon-backend.log" >&2
          exit 1

      - name: Run core smoke suite
        working-directory: frontend
        run: npm run test:smoke:core

      - name: Upload artifacts on failure
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: smoke-core-artifacts
          path: |
            frontend/test-results
            ${{ runner.temp }}/pantheon-backend.log
          if-no-files-found: ignore

      - name: Stop backend
        if: always()
        run: |
          if [ -f "$RUNNER_TEMP/pantheon-backend.pid" ]; then
            kill "$(cat "$RUNNER_TEMP/pantheon-backend.pid")" || true
          fi
```

#### 3.2.2 更新 `smoke-full.yml` (保留完整回归)

```yaml
name: Full Smoke Suite

on:
  schedule:
    - cron: "37 18 * * *"  # 每日UTC 18:37 (北京时间 02:37)
  workflow_dispatch:       # 手动触发
  # 移除 push.branches.main - 改用 smoke-core.yml

# ... 其余保持不变
```

#### 3.2.3 在 PR 流程中添加智能触发

**方案 A: 基于路径的条件触发** (推荐)

```yaml
# .github/workflows/quality.yml 新增 job
  pr-smoke-gate:
    name: PR Smoke Gate
    runs-on: ubuntu-latest
    needs: [change-scope]
    if: |
      github.event_name == 'pull_request' && 
      needs.change-scope.outputs.docs_only != 'true' &&
      needs.change-scope.outputs.governance_only != 'true'
    timeout-minutes: 35
    # ... 复用 smoke-core 的 services 和 steps
    steps:
      # ... (与smoke-core.yml相同)
      - name: Run PR smoke gate
        working-directory: frontend
        run: npm run test:smoke:core
```

**方案 B: 使用 PR 标签手动触发** (保守方案)

```yaml
# .github/workflows/pr-smoke-on-demand.yml
name: PR Smoke On Demand

on:
  pull_request:
    types: [labeled, synchronize]

jobs:
  smoke-on-demand:
    if: contains(github.event.pull_request.labels.*.name, 'smoke-required')
    # ... 运行核心冒烟测试
```

### 3.3 Phase 3: 优化单元测试覆盖率

#### 3.3.1 后端单元测试增强

**目标**: 将覆盖率从当前 11% 提升到 30%+ (渐进式)

**重点补充区域**:

```go
// backend/internal/middleware/ - 中间件关键逻辑
// 现有: 
// - casbin_middleware_test.go ✓
// - data_scope_middleware_test.go ✓
// 缺失:
// - auth_middleware 需补充 token 过期、刷新场景
// - rate_limit_middleware 需补充限流触发场景

// backend/modules/system/ - 系统模块核心服务
// 需补充:
// - user/service.go - 批量操作、状态切换
// - role/service.go - 权限分配、级联删除
// - dept/service.go - 树操作、循环依赖检测

// backend/pkg/ - 公共库
// 现有覆盖不错 (audit, capability, impexp 都有测试)
// 需补充:
// - pkg/authsession - 会话并发、过期清理
```

**实施步骤**:

1. **Week 1-2**: 补充中间件测试，覆盖率 → 15%
2. **Week 3-4**: 补充核心服务测试，覆盖率 → 25%
3. **Week 5+**: 持续迭代，目标 30%

#### 3.3.2 前端单元测试增强

**当前状态**: Vitest + node:test，覆盖率未强制门禁

**重点方向**:

```typescript
// 1. 关键 Hooks 测试
// frontend/src/shared/hooks/
// - useAuth.test.ts - 认证状态管理
// - usePermission.test.ts - 权限计算逻辑
// - useDataScope.test.ts - 数据范围过滤

// 2. 工具函数测试
// frontend/src/shared/utils/
// - validation.test.ts - 表单校验规则
// - formatter.test.ts - 数据格式化
// - apiClient.test.ts - 请求拦截、错误处理

// 3. 状态管理测试
// frontend/src/shared/store/
// - authStore.test.ts - token 刷新、登出流程
// - uiStore.test.ts - 主题、布局状态
```

**覆盖率门禁建议**:

```yaml
# .github/workflows/ci.yml
# coverage-gate job 中:
- name: Check frontend coverage threshold
  run: node scripts/harness/check-coverage.mjs frontend/coverage/coverage-summary.json --format json --threshold 15
  # 从 0 → 15%，后续逐步提升到 30%
```

### 3.4 Phase 4: E2E 测试分层优化

#### 3.4.1 当前 E2E 结构

```
frontend/tests/smoke/ (实际是 E2E)
├── platform/ - 偏向集成测试
├── system/ - 偏向 E2E (完整业务流)
└── business/ - 纯 E2E (跨模块场景)
```

#### 3.4.2 优化建议

**A. 标记测试优先级**

```typescript
// tests/smoke-core/auth-critical.spec.ts
test.describe('Auth Critical Path', () => {
  test('login and logout @priority:critical @smoke:core', async ({ page }) => {
    // ...
  });
});

// tests/smoke/system/system-form-state-matrix.spec.ts  
test.describe('Form State Matrix', () => {
  test('create form validation @priority:high @smoke:full', async ({ page }) => {
    // ...
  });
});
```

**B. 运行器支持标签过滤**

```json
// package.json
{
  "scripts": {
    "test:smoke:critical": "playwright test --grep @priority:critical",
    "test:smoke:high": "playwright test --grep '@priority:(critical|high)'",
    "test:e2e:full": "playwright test --grep @smoke:full"
  }
}
```

---

## 四、实施时间表

### Week 1-2: 快速见效 (Quick Wins)

```
□ 创建 smoke-core 测试套件目录
□ 从现有 28个 spec 中筛选 8-10个核心场景
□ 新增 smoke-core.yml workflow
□ 更新 smoke-full.yml 触发条件
□ 验证 smoke-core 运行时间 < 25分钟
```

**预期效果**:
- ✅ PR 合并后反馈时间从 120分钟 → 25分钟 (提速 80%)
- ✅ 开发者可在 coffee break 内得到反馈

### Week 3-4: 结构优化

```
□ 在 quality.yml 中添加 PR smoke gate (可选，基于路径过滤)
□ 补充后端中间件单元测试 (覆盖率 +5%)
□ 补充前端 hooks 单元测试
□ 更新覆盖率门禁阈值: Go 11% → 15%, Frontend 0% → 10%
```

**预期效果**:
- ✅ 高风险 PR 在合并前即可发现冒烟级问题
- ✅ 单元测试覆盖率明显提升

### Week 5-6: 完善阶段

```
□ 为所有 smoke spec 添加优先级标签
□ 实施 E2E 测试分层 (critical/high/medium)
□ 补充核心服务单元测试 (覆盖率 +10%)
□ 完善测试文档和团队培训
```

### Month 2+: 持续优化

```
□ 监控 CI 时间和稳定性指标
□ 定期审查冒烟测试有效性 (季度)
□ 单元测试覆盖率逐步提升到 30%+
□ 引入测试数据管理策略 (fixture 复用)
```

---

## 五、衡量指标 (Metrics)

### 5.1 性能指标

| 指标 | 优化前 | 目标 | 测量方式 |
|-----|--------|------|---------|
| PR 门禁总时长 | ~10分钟 | ~15分钟 | GitHub Actions workflow duration |
| Merge 后反馈时间 | 120分钟 | 25分钟 | smoke-core.yml duration |
| 每日完整回归时间 | 120分钟 | 100分钟 | smoke-full.yml duration (持续优化) |
| 平均 PR 等待时间 | ~10分钟 | ~15分钟 | 开发者主观感受 |

### 5.2 质量指标

| 指标 | 优化前 | 目标 | 测量方式 |
|-----|--------|------|---------|
| 后端单元测试覆盖率 | 11% | 30% | `go test -cover` |
| 前端单元测试覆盖率 | ~0% | 15% | Vitest coverage |
| 冒烟测试成功率 | 未监控 | >95% | GitHub Actions success rate |
| 生产缺陷逃逸率 | 未监控 | <5% | Issue tracker |

### 5.3 效率指标

| 指标 | 优化前 | 目标 |
|-----|--------|------|
| 开发者 CI 等待时间占比 | ~30% | <15% |
| 测试维护成本 (人时/月) | 未量化 | 建立基线后优化 |
| False positive 率 | 未监控 | <2% |

---

## 六、风险与缓解措施

### 风险 1: 核心冒烟套件遗漏关键场景

**缓解**:
- ✅ 保留完整 smoke-full 每日运行
- ✅ 基于历史回归数据选择核心场景
- ✅ 每季度审查核心套件有效性

### 风险 2: PR 门禁时间增加影响开发体验

**缓解**:
- ✅ 仅对高风险 PR 运行冒烟测试 (路径过滤 or 标签触发)
- ✅ 核心套件严格控制在 20-25 分钟
- ✅ 并行运行 (workers=2)

### 风险 3: 单元测试补充成本高

**缓解**:
- ✅ 渐进式提升，不追求一步到位
- ✅ 优先覆盖高风险、高复杂度模块
- ✅ 使用测试生成工具 (如 gotests)

---

## 七、落地检查清单

### Phase 1 (Week 1-2) - 核心冒烟拆分

- [ ] 创建 `frontend/tests/smoke-core/` 目录
- [ ] 筛选并迁移 8-10 个核心 spec
- [ ] 新增 `package.json` 脚本: `test:smoke:core`
- [ ] 创建 `.github/workflows/smoke-core.yml`
- [ ] 更新 `smoke-full.yml` 触发条件 (移除 push.branches.main)
- [ ] 在 staging/dev 环境验证 smoke-core 运行时间
- [ ] 更新 README.md 测试部分文档

### Phase 2 (Week 3-4) - PR 门禁增强

- [ ] 在 `quality.yml` 添加 `pr-smoke-gate` job (可选)
- [ ] 或创建 `pr-smoke-on-demand.yml` (标签触发)
- [ ] 补充后端中间件单元测试 (目标 +5个文件)
- [ ] 补充前端 hooks 单元测试 (目标 +3个文件)
- [ ] 更新 coverage-gate 阈值配置
- [ ] PR 测试验证

### Phase 3 (Week 5-6) - E2E 分层优化

- [ ] 为现有 spec 添加 `@priority` 和 `@smoke` 标签
- [ ] 实现标签过滤脚本
- [ ] 补充核心服务单元测试
- [ ] 编写测试最佳实践文档
- [ ] 团队培训 (测试分层理念)

### Phase 4 (持续) - 监控与优化

- [ ] 配置 CI 时间监控看板
- [ ] 建立测试质量周报
- [ ] 每季度审查冒烟测试有效性
- [ ] 持续优化单元测试覆盖率

---

## 八、参考资源

### 内部文档
- `frontend/tests/smoke/README.md` - 现有冒烟测试文档
- `.github/workflows/ci.yml` - 当前 CI 配置
- `scripts/run-smoke-suite.mjs` - 冒烟测试运行器

### 最佳实践
- [Google Testing Blog - Test Pyramid](https://testing.googleblog.com/)
- [Martin Fowler - Practical Test Pyramid](https://martinfowler.com/articles/practical-test-pyramid.html)
- [Playwright Best Practices](https://playwright.dev/docs/best-practices)

### 工具
- **后端**: `go test`, `testify`, `golangci-lint`
- **前端**: `vitest`, `@testing-library/react`, `playwright`
- **覆盖率**: `go tool cover`, `vitest --coverage`
- **CI**: GitHub Actions, workflow artifacts

---

## 附录 A: 核心冒烟套件详细清单

### A.1 认证模块 (auth-critical.spec.ts)

```typescript
test.describe('Auth Critical Path @priority:critical @smoke:core', () => {
  test('user can login with valid credentials', async ({ page }) => {
    // 登录成功
  });
  
  test('token refresh works before expiration', async ({ page }) => {
    // token 自动刷新
  });
  
  test('user can logout and clear session', async ({ page }) => {
    // 登出清理
  });
  
  test('expired token redirects to login', async ({ page }) => {
    // 过期跳转
  });
});
```

### A.2 平台壳层 (platform-shell-critical.spec.ts)

```typescript
test.describe('Platform Shell @priority:critical @smoke:core', () => {
  test('shell renders with correct menu structure', async ({ page }) => {
    // 菜单渲染
  });
  
  test('navigation between routes works', async ({ page }) => {
    // 路由导航
  });
  
  test('breadcrumb updates correctly', async ({ page }) => {
    // 面包屑
  });
});
```

### A.3 系统模块核心路径

**用户管理** (system-user-crud.spec.ts):
- 创建用户
- 编辑用户
- 删除用户
- 批量启用/禁用

**角色授权** (system-role-authz.spec.ts):
- 创建角色
- 分配菜单权限
- 分配用户
- 验证权限生效

**部门树** (system-dept-tree.spec.ts):
- 添加根部门
- 添加子部门
- 移动部门
- 删除部门 (级联检查)

### A.4 预期运行时间分解

| Spec | 场景数 | 预估耗时 | 并发 |
|------|--------|---------|-----|
| auth-critical | 4 | 2分钟 | Worker 1 |
| platform-shell-critical | 3 | 1.5分钟 | Worker 1 |
| system-user-crud | 4 | 3分钟 | Worker 2 |
| system-role-authz | 4 | 3分钟 | Worker 1 |
| system-dept-tree | 4 | 2.5分钟 | Worker 2 |
| system-menu-permission | 3 | 2分钟 | Worker 1 |
| business-generated-basic | 3 | 3分钟 | Worker 2 |
| api-import-export-basic | 2 | 2分钟 | Worker 1 |
| **总计** | **27** | **~20分钟** | **workers=2** |

---

## 附录 B: 决策记录

### DR-001: 为什么选择 20 分钟作为核心冒烟目标？

**背景**: 
- 当前完整冒烟 120 分钟
- 开发者 coffee break 通常 15-20 分钟

**决策**:
选择 20 分钟是基于：
1. **心理阈值**: 开发者愿意等待的最长时间
2. **覆盖率平衡**: 可以覆盖 70%+ 的关键路径
3. **工程可行性**: 8-10 个 spec，每个 2-3 分钟

**替代方案**:
- 10 分钟: 太激进，覆盖率不足
- 30 分钟: 开发者体验下降

### DR-002: PR 门禁是否应包含冒烟测试？

**决策**: 采用 **条件触发** 策略

**理由**:
1. **正方**: 提前发现集成问题
2. **反方**: 延长 PR 等待时间，影响体验

**最终方案**:
- **默认**: PR 只运行单元测试 (~10分钟)
- **条件 1**: PR 修改 `backend/` 或 `frontend/src/modules/system/` → 自动触发核心冒烟
- **条件 2**: PR 带 `smoke-required` 标签 → 手动触发核心冒烟
- **保底**: 合并后 main 分支自动运行核心冒烟

### DR-003: 单元测试覆盖率目标为何定为 30%？

**背景**:
- 当前后端覆盖率 11%
- 前端覆盖率几乎为 0

**决策**: 渐进式提升到 30%

**理由**:
1. **业界标准**: 中小型项目通常 30-50%
2. **投入产出比**: 30% 可覆盖核心路径，继续提升边际收益递减
3. **可行性**: 2-3 个月可达成

**路线图**:
- Month 1: 11% → 20% (补充中间件和工具函数)
- Month 2: 20% → 30% (补充核心服务)
- Month 3+: 保持 30%，重点提升测试质量

---

## 修订历史

| 版本 | 日期 | 作者 | 变更说明 |
|-----|------|------|---------|
| 1.0 | 2026-09-05 | Kiro | 初始版本 |

