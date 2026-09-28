# v0.11.1 Release Status

## 当前状态
⏳ **PR #292 CI 验证中**

## 最新更新
**时间**: 2026-09-06 00:20 UTC

### 问题排查与解决

#### 问题 1: Docs Governance 反复失败
**原因**: PR body 中使用了 markdown 加粗语法（`**yes**`）
- 检查脚本的 `normalizeValue()` 函数只移除反引号，不移除星号
- 导致 `Trivial change: **yes**` 被解析为包含星号的值
- 不匹配 `YES_VALUES` 集合（只包含纯文本 "yes"）

**解决**: 
- 移除所有 markdown 格式化标记
- 使用纯文本值：`Trivial change: yes`
- 重新触发 CI

#### 问题 2: CI 不自动读取更新的 PR body
**原因**: GitHub Actions workflow 在触发时就捕获了 PR body
- 后续更新 PR body 不会影响已运行的 CI
- 必须触发新的 workflow run

**解决**: 
- 使用 `gh pr close` + `gh pr reopen` 触发新 CI
- 或推送空提交（但遇到网络问题）

## 经验教训

### ✅ 正确的做法
1. **测试字段值格式**
   - 在本地测试解析逻辑
   - 使用纯文本值，避免 markdown 格式

2. **理解 CI 触发机制**
   - PR body 更新不自动重跑 CI
   - 需要显式触发新 run

3. **阅读源码理解验证逻辑**
   - 查看 `check-pr-governance.mjs`
   - 理解 `parseField()` 和 `normalizeValue()` 的精确行为

### 📚 工具改进建议

#### 1. 改进 PR body 生成器
在 `scripts/harness/generate-pr-body.mjs` 中添加警告：
```javascript
// 生成纯文本值，避免 markdown 格式
// ❌ 错误: Trivial change: **yes**
// ✅ 正确: Trivial change: yes
```

#### 2. 改进解析函数
在 `normalizeValue()` 中移除常见的 markdown 标记：
```javascript
function normalizeValue(value) {
  return value
    .trim()
    .replace(/^`+/, '')
    .replace(/`+$/, '')
    .replace(/^\*\*/, '')  // 移除开头的 **
    .replace(/\*\*$/, '')  // 移除结尾的 **
    .trim();
}
```

#### 3. 添加本地验证脚本
创建 `scripts/validate-pr-body.mjs` 用于本地测试：
```bash
node scripts/validate-pr-body.mjs /path/to/pr-body.md
```

## 时间线

| 时间 | 事件 |
|------|------|
| 00:00 | 创建 PR #292 (release/v0.11.1) |
| 00:03 | Docs Governance 失败 - 缺少必需字段 |
| 00:05 | 添加完整的 governance 字段到 PR body |
| 00:08 | 发现 CI 不自动读取更新的 body |
| 00:10 | 关闭并重新打开 PR 触发新 CI |
| 00:12 | Docs Governance 仍然失败 |
| 00:15 | 发现 `**yes**` markdown 格式问题 |
| 00:18 | 移除所有 markdown 格式，更新 PR body |
| 00:20 | 再次触发 CI，等待结果 |

## 下一步

1. ⏳ 等待 Docs Governance 检查通过
2. ✅ 确认所有 CI 检查通过
3. 🔀 合并 PR #292 到 main
4. 🏷️ 创建 GitHub Release v0.11.1
5. 📝 更新 CHANGELOG.md

---

**监控命令运行中**: `bhki0k598`
