# 测试体系优化 - 快速启动指南

> **TL;DR**: 当前每次 PR 机械等待冒烟测试 120 分钟。优化方案：拆分核心冒烟套件(20分钟) + 保留完整回归(每日)，开发效率提升 80%。

---

## 一、核心问题

```
❌ 当前状态:
   - smoke-full.yml: 120分钟，每次 push main 触发
   - PR 流程: 没有冒烟测试，关键路径未保护
   - 开发者: 长时间等待，反馈慢

✅ 优化目标:
   - PR 合并后: 20分钟核心冒烟 (提速 83%)
   - 每日定时: 120分钟完整回归
   - PR 阶段: 10分钟快速门禁 + 可选核心冒烟
```

---

## 二、三步实施 (2周见效)

### Step 1: 拆分核心冒烟套件 (Week 1, 预计 8 小时)

**A. 创建核心测试目录**

```bash
cd pantheon-base/frontend
mkdir tests/smoke-core
```

**B. 筛选 8-10 个核心场景**

从现有 28 个 spec 中迁移：

```bash
# 推荐优先级 (按业务影响排序):
tests/smoke/platform/platform-shell.spec.ts       → smoke-core/platform-shell-critical.spec.ts
tests/smoke/system/system-pages.spec.ts           → smoke-core/system-user-crud.spec.ts (精简)
tests/smoke/system/role-authorization.spec.ts     → smoke-core/system-role-authz.spec.ts
tests/smoke/system/governance/module-governance.spec.ts → smoke-core/api-import-export-basic.spec.ts

# 精简策略: 每个 spec 只保留 3-4 个最关键的 test case
# 例如: 完整的 system-pages.spec.ts 可能有 15 个 test
#      核心版本只保留: 创建、编辑、删除、批量操作
```

**C. 添加 package.json 脚本**

```bash
cd frontend
npm pkg set scripts.test:smoke:core="node scripts/run-smoke-suite.mjs --host 127.0.0.1 --port 5173 --cleanup-fixtures all --config playwright.config.ts -- tests/smoke-core/*.spec.ts --workers=2"
```

**D. 本地验证**

```bash
# 启动后端 (Terminal 1)
cd backend
go run ./cmd/server

# 运行核心冒烟 (Terminal 2)
cd frontend
npm run test:smoke:core

# 目标: < 25 分钟完成
```

---

### Step 2: 创建新 Workflow (Week 1, 预计 4 小时)

**A. 创建 `smoke-core.yml`**

```bash
cd pantheon-base/.github/workflows
# 复制 smoke-full.yml
cp smoke-full.yml smoke-core.yml
```

**B. 修改关键配置**

```yaml
# smoke-core.yml
name: Core Smoke Tests  # 改名

on:
  push:
    branches:
      - main            # 保留
      - release/**      # 保留
  # 移除 schedule (每日任务在 smoke-full)
  workflow_dispatch:    # 保留手动触发

jobs:
  smoke-core:
    timeout-minutes: 30  # 从 120 改为 30
    steps:
      # ... 其他步骤保持不变 ...
      
      - name: Run core smoke suite  # 改名
        working-directory: frontend
        run: npm run test:smoke:core  # 改为核心套件
```

**C. 更新 `smoke-full.yml`**

```yaml
# smoke-full.yml
on:
  schedule:
    - cron: "37 18 * * *"  # 每日 UTC 18:37
  workflow_dispatch:
  # 移除 push.branches.main - 这个交给 smoke-core

# ... 其余保持不变
```

**D. 提交验证**

```bash
git add .github/workflows/smoke-core.yml
git add .github/workflows/smoke-full.yml
git add frontend/tests/smoke-core/
git add frontend/package.json
git commit -m "feat: add core smoke test suite for faster feedback"
git push origin <your-branch>

# 创建 PR，观察 smoke-core workflow 运行时间
```

---

### Step 3: (可选) PR 门禁增强 (Week 2, 预计 4 小时)

**选项 A: 基于路径自动触发** (推荐给高频变更模块)

```yaml
# .github/workflows/quality.yml
jobs:
  pr-smoke-gate:
    name: PR Smoke Gate
    if: |
      github.event_name == 'pull_request' && 
      needs.change-scope.outputs.docs_only != 'true'
    runs-on: ubuntu-latest
    timeout-minutes: 35
    # ... 复用 smoke-core 的完整配置
```

**选项 B: 标签手动触发** (推荐给保守团队)

```yaml
# .github/workflows/pr-smoke-on-demand.yml
name: PR Smoke On Demand

on:
  pull_request:
    types: [labeled, synchronize]

jobs:
  smoke:
    if: contains(github.event.pull_request.labels.*.name, 'test:smoke')
    # ... 运行核心冒烟
```

使用方式：
```bash
# PR 上打标签
gh pr edit <PR号> --add-label "test:smoke"
```

---

## 三、预期效果对比

### 优化前

```
开发者提交 PR → 等待 10 分钟 (ci.yml) → 合并
                                          ↓
                          main 分支推送触发 smoke-full.yml
                                          ↓
                        等待 120 分钟 🕐🕑🕒 😴
                                          ↓
                          发现问题 → 修复 → 重新走流程
                          
总反馈周期: 130 分钟+
```

### 优化后

```
开发者提交 PR → 等待 10-15 分钟 (ci.yml + 可选 smoke-core)
               ↓
           快速反馈 ✅ → 修复小问题 → 合并
                                    ↓
                        main 分支触发 smoke-core.yml
                                    ↓
                          等待 20 分钟 ☕
                                    ↓
                        95% 的问题在这里发现 ✅
                                    ↓
                  每日 smoke-full (边缘场景兜底)
                  
PR 反馈周期: 10-15 分钟
合并后反馈: 20 分钟 (提速 83%)
```

---

## 四、监控指标

部署后持续观察：

```bash
# 1. CI 时间趋势
gh run list --workflow=smoke-core.yml --limit 10 --json displayTitle,conclusion,updatedAt,createdAt

# 2. 成功率
gh run list --workflow=smoke-core.yml --limit 50 --json conclusion | jq '[.[] | .conclusion] | group_by(.) | map({status: .[0], count: length})'

# 3. 平均耗时
# 在 GitHub Actions UI 查看 workflow insights
```

**目标基线**:
- ✅ smoke-core 平均耗时: 18-22 分钟
- ✅ 成功率: >95%
- ✅ 假阳性率 (误报): <2%

---

## 五、常见问题

### Q1: 核心冒烟遗漏了边缘场景怎么办？

**A**: 
- 保留完整 smoke-full 每日运行 (兜底)
- 每季度审查核心套件有效性
- 生产问题回溯时补充到核心套件

### Q2: 20 分钟还是太长了？

**A**:
有进一步优化空间：
1. **并行度提高**: `--workers=2` → `--workers=4` (需要更多 runner 资源)
2. **测试数据复用**: 减少 fixture setup 时间
3. **选择性运行**: 基于 git diff 只跑相关模块 (advanced)

但建议先运行 2-4 周，收集数据后再优化。

### Q3: 如何决定哪些测试放入核心套件？

**A**: 
三个标准：
1. **影响面**: 所有用户都会用到的功能 (登录、导航)
2. **风险度**: 数据破坏性操作 (删除、批量更新)
3. **历史回归**: 过去 6 个月反复出问题的区域

查看历史：
```bash
# 查看最近 3 个月的 smoke-full 失败记录
gh run list --workflow=smoke-full.yml --limit 100 --json conclusion,createdAt \
  | jq '.[] | select(.conclusion == "failure")'
```

### Q4: 单元测试覆盖率如何快速提升？

**A**:
**后端 (Go)**:
```bash
# 1. 找出覆盖率低的关键模块
cd backend
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | sort -k3 -n | head -20

# 2. 使用 gotests 生成测试骨架
go install github.com/cweill/gotests/gotests@latest
gotests -all -w modules/system/user/service.go

# 3. 补充关键逻辑测试
```

**前端 (TypeScript)**:
```bash
# 1. 查看覆盖率报告
cd frontend
npm run test:unit:coverage
open coverage/index.html

# 2. 优先补充 utils 和 hooks (投入产出比高)
# 3. 使用 @testing-library/react 测试组件
```

---

## 六、后续优化路线

完成 2 周快速见效后，可以考虑：

### 第 1 个月
- [ ] 补充后端中间件单元测试 (覆盖率 +5%)
- [ ] 补充前端 hooks 单元测试
- [ ] 为所有 spec 添加 `@priority` 标签

### 第 2 个月
- [ ] 实施 E2E 测试分层 (critical/high/medium)
- [ ] 补充核心服务单元测试 (覆盖率 +10%)
- [ ] 优化测试数据管理 (fixture 复用)

### 第 3 个月
- [ ] 引入测试数据工厂 (减少 setup 时间)
- [ ] 实施视觉回归测试 (Playwright screenshots)
- [ ] 建立测试质量看板

---

## 七、需要决策的问题

在开始实施前，请确认：

### 决策 1: PR 门禁策略

- [ ] **选项 A**: PR 默认只跑单元测试 (~10分钟)，合并后跑核心冒烟 (~20分钟)
- [ ] **选项 B**: PR 自动跑单元测试 + 核心冒烟 (~30分钟)
- [ ] **选项 C**: PR 默认只跑单元测试，打标签手动触发冒烟

**推荐**: 选项 A (先快后稳，开发体验最佳)

### 决策 2: 核心冒烟粒度

- [ ] **激进** (8 spec, ~15分钟): 只覆盖最核心路径
- [ ] **平衡** (10 spec, ~20分钟): 推荐 ✅
- [ ] **保守** (15 spec, ~30分钟): 覆盖更全但稍慢

### 决策 3: 完整冒烟运行频率

- [ ] **每日一次** (UTC 18:37 = 北京时间 02:37): 推荐 ✅
- [ ] **每日两次** (早晚各一次): 成本翻倍
- [ ] **仅发布前手动触发**: 风险较高

---

## 八、立即开始

```bash
# 1. 阅读完整方案
cat docs/testing-strategy-optimization.md

# 2. 创建实施分支
git checkout -b feat/test-suite-optimization

# 3. 按照 Step 1-2 执行

# 4. 遇到问题查看 FAQ 或提 Issue
```

**预计投入**: 
- Week 1: 12 小时 (拆分 + workflow)
- Week 2: 4 小时 (PR 门禁 + 调优)
- **总计**: 2 周，16 小时

**预期收益**:
- 反馈时间减少 83%
- 开发者等待时间减少 80%
- 每周节省团队 10+ 小时

---

## 参考资料

- **完整方案**: [docs/testing-strategy-optimization.md](./testing-strategy-optimization.md)
- **现有测试**: `frontend/tests/smoke/`
- **运行器脚本**: `frontend/scripts/run-smoke-suite.mjs`
- **工作流配置**: `.github/workflows/smoke-full.yml`

---

**最后更新**: 2026-09-05  
**维护者**: Kiro  
**反馈渠道**: GitHub Issues
