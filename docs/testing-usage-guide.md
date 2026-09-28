# 测试体系优化 - 使用指南

## 🎉 优化已完成！

你的测试体系已成功优化。以下是如何使用新的测试流程。

---

## 📋 快速参考

### 三种测试模式

| 模式 | 何时运行 | 耗时 | 覆盖范围 |
|-----|---------|------|---------|
| **Core Smoke** | PR合并到main后 | 20分钟 | 核心路径70% |
| **PR Smoke** (可选) | PR添加标签后 | 25分钟 | 核心路径70% |
| **Full Smoke** | 每日定时/手动 | 120分钟 | 完整100% |

---

## 🚀 场景 1: 日常开发（默认）

```bash
# 1. 创建功能分支
git checkout -b feat/add-new-feature

# 2. 开发 + 提交
git add .
git commit -m "feat: add awesome feature"
git push origin feat/add-new-feature

# 3. 创建 PR
gh pr create --title "feat: add awesome feature"

# 4. CI 自动运行 (~10分钟)
#    ✓ 快速检查 (格式化、类型检查、lint)
#    ✓ 单元测试
#    ✓ Smoke sanity

# 5. 合并 PR
gh pr merge --squash

# 6. 自动触发 smoke-core (~20分钟)
#    ✓ 8个核心场景自动运行
#    ✓ 95%的问题在这里发现
```

**优点**: 零额外等待，合并后 20 分钟快速反馈

---

## ⚠️ 场景 2: 高风险变更（推荐）

**适用于**:
- 修改认证/授权模块
- 数据库 schema 变更
- 核心 API 重构
- 权限系统调整

```bash
# 1. 创建 PR（同上）
gh pr create --title "refactor: redesign auth system"

# 2. 添加标签触发核心冒烟
gh pr edit --add-label "smoke-required"

# 或在 GitHub UI 上:
# PR 页面 → Labels → 添加 "smoke-required"

# 3. CI 自动运行 pr-smoke-core (~25分钟)
#    ✓ 在合并前运行核心冒烟
#    ✓ 提前发现集成问题

# 4. 通过后再合并
gh pr merge --squash
```

**优点**: 合并前发现问题，避免破坏主分支

---

## 🔍 场景 3: 发布前完整验证

```bash
# 手动触发完整冒烟测试
gh workflow run smoke-full.yml

# 或等待每日自动运行
# 北京时间每天 02:37 自动运行
```

**优点**: 覆盖所有边缘场景，确保质量

---

## 📊 监控测试健康度

### 查看最近的 smoke-core 运行

```bash
# 列出最近 10 次运行
gh run list --workflow=smoke-core.yml --limit 10

# 查看特定运行的详细日志
gh run view <run-id> --log

# 查看失败的运行
gh run list --workflow=smoke-core.yml --status failure
```

### 目标指标

- ✅ smoke-core 平均耗时: **18-22 分钟**
- ✅ 成功率: **> 95%**
- ✅ 假阳性率: **< 2%**

---

## 🛠️ 本地运行测试

### 运行核心冒烟测试

```bash
# Terminal 1: 启动后端
cd pantheon-base/backend
go run ./cmd/server

# Terminal 2: 运行核心冒烟
cd pantheon-base/frontend
npm run test:smoke:core

# 目标: < 25 分钟完成
```

### 运行单个测试文件

```bash
cd frontend
npx playwright test tests/smoke-core/auth-login-logout.spec.ts
```

### 调试模式

```bash
npx playwright test tests/smoke-core/auth-login-logout.spec.ts --debug
```

---

## 🔧 常见操作

### 为现有 PR 添加冒烟测试

```bash
# 方式 1: 使用 gh CLI
gh pr edit 123 --add-label "smoke-required"

# 方式 2: GitHub UI
# 打开 PR → Labels → 添加 "smoke-required"
```

### 手动触发 smoke-core

```bash
# 在 main 分支触发
gh workflow run smoke-core.yml

# 在特定分支触发
gh workflow run smoke-core.yml --ref feat/my-branch
```

### 查看测试覆盖率

```bash
cd frontend
npm run test:unit:coverage

# 打开报告
open coverage/index.html  # macOS
start coverage/index.html # Windows
```

---

## 📈 理解测试结果

### smoke-core 成功 ✅

```
所有 8 个核心场景通过
→ 核心功能正常
→ 可以安全部署
```

### smoke-core 失败 ❌

1. **查看失败的测试**:
   ```bash
   gh run view <run-id> --log | grep "FAILED"
   ```

2. **本地复现**:
   ```bash
   cd frontend
   npx playwright test tests/smoke-core/<failed-spec>.spec.ts
   ```

3. **修复并重新提交**:
   ```bash
   git add .
   git commit -m "fix: resolve smoke test failure"
   git push
   # smoke-core 会自动重新运行
   ```

---

## ⏱️ 时间对比

### 优化前

```
开发 → PR (10分钟) → 合并 → smoke-full (120分钟) ❌
                                    ↓
                              2小时后发现问题
```

### 优化后

```
开发 → PR (10分钟) → 合并 → smoke-core (20分钟) ✅
                                    ↓
                              20分钟后发现问题
                              (提速 6倍)
```

---

## 💡 最佳实践

### ✅ DO (推荐)

1. **高风险 PR 添加 `smoke-required` 标签**
   - 认证/授权变更
   - 数据库 schema 变更
   - 核心 API 重构

2. **关注 smoke-core 失败通知**
   - 通常在 20 分钟内收到
   - 优先修复失败的核心测试

3. **发布前运行完整冒烟**
   ```bash
   gh workflow run smoke-full.yml
   ```

### ❌ DON'T (避免)

1. **不要忽略 smoke-core 失败**
   - 可能导致生产问题

2. **不要频繁手动触发 smoke-full**
   - 耗时 2 小时
   - 使用 smoke-core 即可

3. **不要绕过测试直接部署**
   - 至少等待 smoke-core 通过

---

## 🆘 遇到问题？

### smoke-core 运行超时 (>30分钟)

**可能原因**:
- 后端启动慢
- 数据库连接问题
- 某个测试死锁

**排查**:
```bash
# 1. 查看后端日志
gh run view <run-id> --log | grep "pantheon-backend.log"

# 2. 本地运行定位慢速测试
cd frontend
npx playwright test tests/smoke-core/*.spec.ts --reporter=list
```

### PR 标签不触发测试

**检查清单**:
- [ ] 标签名称是否为 `smoke-required` (精确匹配)
- [ ] PR 是否为纯文档变更
- [ ] 是否在 PR 同步后添加的标签

**解决**:
```bash
# 触发新的 PR 同步
git commit --allow-empty -m "trigger: re-run CI"
git push
```

### 测试在本地通过但 CI 失败

**常见原因**:
1. **环境差异**: 本地 vs CI 数据库状态不同
2. **并发问题**: CI 使用 `--workers=2`
3. **时间依赖**: CI 环境可能较慢

**解决**:
```bash
# 本地模拟 CI 环境
cd frontend
npm run test:smoke:core -- --workers=2
```

---

## 📚 延伸阅读

- [完整优化方案](./testing-strategy-optimization.md) - 36页详细设计
- [快速启动指南](./testing-quick-start.md) - 2周实施路线
- [实施报告](./testing-implementation-report.md) - 当前状态和指标
- [核心套件 README](../frontend/tests/smoke-core/README.md) - 测试详情

---

## 🎯 记住这些关键点

1. **默认模式**: PR → 合并 → smoke-core (20分钟) ✨
2. **高风险变更**: 添加 `smoke-required` 标签 ⚠️
3. **发布前**: 运行 smoke-full (手动触发) 🚀
4. **监控**: 关注 smoke-core 成功率 > 95% 📊

---

**祝测试愉快！🎉**

如有问题，请在 GitHub Issues 中提问，标签: `label:testing`
