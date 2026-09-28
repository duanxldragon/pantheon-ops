# Harness PR Governance Generator

## 问题
每次创建 PR 都需要手动填写复杂的 governance 模板，容易遗漏字段导致 CI 失败。

## 解决方案
创建标准化工具链：

### 1. PR Body 生成器
```bash
node scripts/harness/generate-pr-body.mjs <task-id>
```

自动从 `.harness/tasks/<task-id>/manifest.json` 生成完全合规的 PR body。

### 2. 使用流程

#### Step 1: 创建 Task Manifest
```json
{
  "taskId": "your-task-id",
  "title": "Task title",
  "description": "中文描述",
  "scope": "frontend/backend/platform",
  "impact": "trivial/minor/major",
  "qualityProfile": "none",
  "ratchetDecision": "no-repeat-observed",
  "githubSignal": "repo-quality-gate",
  "boundaries": "描述边界",
  "files": ["file1.ts", "file2.ts"]
}
```

#### Step 2: 创建 Evidence 文件
- `.harness/evidence/<task-id>/commands.json` - 执行的命令和结果
- `.harness/evidence/<task-id>/summary.md` - 验证摘要
- `.harness/evidence/<task-id>/review.md` - 代码审查

#### Step 3: 生成并使用 PR Body
```bash
# 生成 PR body
node scripts/harness/generate-pr-body.mjs your-task-id > pr-body.txt

# 创建 PR
gh pr create --body-file pr-body.txt

# 或更新现有 PR
node scripts/harness/generate-pr-body.mjs your-task-id | gh pr edit <pr-number> --body-file -
```

### 3. Git Hook（可选）
安装 pre-push hook 自动验证：
```bash
cp scripts/hooks/pre-push .git/hooks/pre-push
chmod +x .git/hooks/pre-push
```

## 必需字段映射

### Manifest → PR Body
- `taskId` → Task ID, task id
- `description` → 目标问题
- `scope` → 改动层级
- `files` → 改动模块
- `impact` → 预期影响, Trivial change
- `ratchetDecision` → Ratchet Decision (必须是允许值之一)
- `githubSignal` → GitHub Signal

### Ratchet Decision 允许值
- `no-repeat-observed` - 未观察到重复问题
- `guide-updated` - 更新了指南
- `sensor-added` - 添加了传感器
- `gate-updated` - 更新了门禁
- `template-updated` - 更新了模板
- `adapter-updated` - 更新了适配器
- `registry-only` - 仅注册表

### GitHub Signal 允许值
- `method-gate` - 方法门禁
- `repo-quality-gate` - 仓库质量门禁
- `runtime-evidence-gate` - 运行时证据门禁
- `external-flaky` - 外部不稳定
- `not-applicable` - 不适用

## 示例

参考 `.harness/tasks/sonarcloud-s4666/` 作为完整示例。
