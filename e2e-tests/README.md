# Pantheon-Ops E2E 冒烟测试

基于 Playwright 的端到端冒烟测试套件，用于验证 v1.2.0 版本的核心功能。

## 测试范围

根据 QA 验收文档 (`QA_ACCEPTANCE_CRITERIA.md`)，本测试套件覆盖以下核心功能：

### ✅ 模块 A: ArgoCD GitOps 集成
- A.1.1 - 添加 ArgoCD 服务器
- A.1.2 - 测试 ArgoCD 连接
- A.1.3 - 查询 ArgoCD 服务器列表
- A.2.1 - 添加 Git 仓库
- A.3.1 - 创建 ArgoCD 应用
- A.3.2 - 查询应用列表

### ✅ 模块 B: Velero 备份恢复
- B.1.1 - 添加存储位置
- B.2.1 - 创建备份
- B.2.2 - 查询备份列表
- B.3.1 - 创建定时备份
- B.5.1 - 创建灾难演练

### ✅ 模块 C: 监控告警系统
- C.1.1 - 添加 Prometheus 数据源
- C.1.2 - 测试数据源连接
- C.2.1 - 创建告警规则
- C.2.2 - 查询告警规则列表
- C.4.2 - 查询告警事件列表
- C.5.1 - 添加通知渠道
- C.5.2 - 查询通知渠道列表

### ✅ 模块 D: 文件分发系统
- D.1.1 - 创建文件存储
- D.1.2 - 查询存储列表
- D.2.1 - 上传文件
- D.2.2 - 查询文件列表
- D.3.1 - 创建分发任务
- D.3.4 - 查询任务列表

### ✅ 模块 E: 服务实例管理
- E.1.1 - 查询服务实例列表

### ✅ 模块 F: 安全与加密
- F.1.3 - 验证敏感信息已加密
- F.2.1 - 验证 API 需要认证

## 环境要求

- Node.js 16+
- 运行中的 Pantheon-Ops 后端服务

## 安装

```bash
cd e2e-tests
npm install
npx playwright install chromium
```

## 配置

环境变量配置：

```bash
# API 基础 URL (默认: http://localhost:8080)
export BASE_URL="http://localhost:8080"

# API 认证 Token
export API_TOKEN="your-api-token"
```

## 运行测试

### 运行所有冒烟测试

```bash
npm test
```

### 运行特定模块测试

```bash
# 仅 ArgoCD 模块
npx playwright test --grep "模块 A"

# 仅 Velero 模块
npx playwright test --grep "模块 B"

# 仅监控告警模块
npx playwright test --grep "模块 C"
```

### 带界面运行 (调试)

```bash
npm run test:headed
```

### 调试模式

```bash
npm run test:debug
```

### UI 模式 (推荐)

```bash
npm run test:ui
```

## 查看测试报告

```bash
npm run test:report
```

报告将在浏览器中打开，显示：
- 测试用例通过率
- 失败截图
- 网络请求详情
- 测试时长统计

## 测试结果示例

```
Running 30 tests using 1 worker

  ✓ 模块 A: ArgoCD GitOps 集成 - 冒烟测试
    ✓ A.1.1 - 添加 ArgoCD 服务器 (1.2s)
    ✓ A.1.3 - 查询 ArgoCD 服务器列表 (0.5s)
    ✓ A.1.2 - 测试 ArgoCD 连接 (1.0s)
    ✓ A.2.1 - 添加 Git 仓库 (0.8s)
    ✓ A.3.1 - 创建 ArgoCD 应用 (1.5s)
    ✓ A.3.2 - 查询应用列表 (0.6s)

  ✓ 模块 B: Velero 备份恢复 - 冒烟测试
    ✓ B.1.1 - 添加存储位置 (1.0s)
    ✓ B.2.1 - 创建备份 (1.2s)
    ✓ B.2.2 - 查询备份列表 (0.5s)
    ✓ B.3.1 - 创建定时备份 (0.9s)
    ✓ B.5.1 - 创建灾难演练 (1.1s)

  ✓ 模块 C: 监控告警系统 - 冒烟测试
    ✓ C.1.1 - 添加 Prometheus 数据源 (0.8s)
    ✓ C.1.2 - 测试数据源连接 (0.7s)
    ✓ C.2.1 - 创建告警规则 (1.0s)
    ✓ C.2.2 - 查询告警规则列表 (0.5s)
    ✓ C.4.2 - 查询告警事件列表 (0.6s)
    ✓ C.5.1 - 添加通知渠道 (Webhook) (0.7s)
    ✓ C.5.2 - 查询通知渠道列表 (0.5s)

  ✓ 模块 D: 文件分发系统 - 冒烟测试
    ✓ D.1.1 - 创建文件存储 (0.9s)
    ✓ D.1.2 - 查询存储列表 (0.5s)
    ✓ D.2.1 - 上传文件 (1.5s)
    ✓ D.2.2 - 查询文件列表 (0.6s)
    ✓ D.3.1 - 创建分发任务 (1.1s)
    ✓ D.3.4 - 查询分发任务列表 (0.5s)

  ✓ 模块 E: 服务实例管理 - 冒烟测试
    ✓ E.1.1 - 查询服务实例列表 (0.5s)

  ✓ 模块 F: 安全与加密 - 冒烟测试
    ✓ F.1.3 - 验证敏感信息已加密 (1.0s)
    ✓ F.2.1 - 验证 API 需要认证 (0.4s)

30 passed (25.3s)
```

## CI/CD 集成

### GitHub Actions

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      
      - name: Install dependencies
        run: |
          cd e2e-tests
          npm install
          npx playwright install --with-deps chromium
      
      - name: Run smoke tests
        run: |
          cd e2e-tests
          npm test
        env:
          BASE_URL: http://localhost:8080
          API_TOKEN: ${{ secrets.API_TOKEN }}
      
      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: e2e-tests/test-results/
```

## 测试覆盖说明

本冒烟测试套件覆盖了 QA 验收文档中的 **P0 (阻塞性)** 核心功能点，共计 **30 个测试用例**。

### 测试策略

1. **API 级别测试**: 直接测试 REST API 端点
2. **功能完整性**: 验证创建、查询、更新等核心操作
3. **数据验证**: 验证响应数据结构和内容
4. **安全性**: 验证敏感信息不暴露、需要认证

### 未覆盖的内容

以下内容需要更深入的测试（建议后续补充）：

- **P1/P2 功能**: 次要功能点
- **性能测试**: 并发、响应时间
- **UI 测试**: 前端界面交互
- **集成测试**: 多模块协同
- **端到端流程**: 完整业务场景

## 故障排查

### 测试失败

1. **连接失败**: 检查 `BASE_URL` 是否正确
2. **认证失败**: 检查 `API_TOKEN` 是否有效
3. **数据依赖**: 某些测试依赖先前测试创建的数据

### 调试技巧

```bash
# 查看详细日志
DEBUG=pw:api npm test

# 运行单个测试
npx playwright test --grep "A.1.1"

# 保留测试数据
npx playwright test --repeat-each=1
```

## 维护建议

1. **定期更新**: 随 API 变更更新测试
2. **添加断言**: 增加更多数据验证
3. **清理数据**: 测试后清理创建的测试数据
4. **并行执行**: 增加 workers 提高执行速度

## 参考文档

- [QA 验收标准文档](../.harness/v1.2-tasks/QA_ACCEPTANCE_CRITERIA.md)
- [Playwright 官方文档](https://playwright.dev)
- [API 文档](../docs/USER_MANUAL.md)

---

**测试版本**: v1.2.0  
**最后更新**: 2026-08-28
