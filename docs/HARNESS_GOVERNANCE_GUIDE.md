# Pantheon-Base Harness Governance 完整指南

## 问题历史
Docs Governance 检查反复失败，原因是 PR body 格式复杂且 manifest.json 结构要求严格。

## 根本原因
1. **Manifest 结构复杂**：需要 `goal`, `primaryLayer`, `scope`, `linkage` 等 10+ 个必需字段
2. **PR body 模板严格**：必须包含所有必需的 section 和 field
3. **人工填写易错**：字段遗漏、格式不对、路径错误
4. **缺少验证工具**：推送前无法提前发现问题

## 完整解决方案

### 1. 标准 Manifest 模板

参考 `.harness/tasks/sonarcloud-s4666/manifest.json`，必需字段：

```json
{
  "taskId": "task-id",
  "title": "Task title",
  "status": "completed | planned | in-progress | abandoned",
  "goal": "明确的目标描述",
  "primaryLayer": "frontend|backend|platform|system",
  "dependencyLayers": [],
  "scope": {
    "in": ["包含的改动范围"],
    "out": ["明确不包含的内容"]
  },
  "taskDoc": "相关文档",
  "contractAnchors": [],
  "expectedFiles": {
    "create": [],
    "modify": [],
    "doNotTouch": []
  },
  "executionRoles": {
    "implementerPosture": "implementer",
    "reviewerPosture": ["mechanical"]
  },
  "verificationPlan": {
    "backend": [],
    "frontend": [],
    "browser": []
  },
  "runtimeSensitive": false,
  "linkage": {
    "taskId": "task-id",
    "taskPacket": ".harness/tasks/task-id/manifest.json",
    "evidenceDir": ".harness/evidence/task-id/",
    "reviewFile": ".harness/evidence/task-id/review.md",
    "summaryFile": ".harness/evidence/task-id/summary.md",
    "changeRef": "branch-name",
    "planRefs": []
  },
  "evidenceRequired": [],
  "humanGates": []
}
```

### 2. 自动生成工具

**脚本**: `scripts/harness/generate-pr-body.mjs`

```bash
# 生成 PR body
node scripts/harness/generate-pr-body.mjs <task-id>

# 直接创建 PR
node scripts/harness/generate-pr-body.mjs <task-id> | gh pr create --body-file -

# 更新现有 PR
node scripts/harness/generate-pr-body.mjs <task-id> | gh pr edit <pr-number> --body-file -
```

### 3. 标准工作流程

#### Step 1: 创建 Task 目录结构
```bash
TASK_ID="your-task-id"
mkdir -p .harness/tasks/$TASK_ID
mkdir -p .harness/evidence/$TASK_ID
```

#### Step 2: 编写 manifest.json
复制 `sonarcloud-s4666` 的 manifest.json 作为模板，修改：
- `taskId`, `title`, `goal`
- `primaryLayer`, `scope`
- `expectedFiles.modify`
- `linkage.changeRef` (分支名)

#### Step 3: 创建 Evidence 文件
```bash
cd .harness/evidence/$TASK_ID

# commands.json - 执行的命令记录
cat > commands.json << 'EOF'
{
  "commands": [
    {"cmd": "command here", "result": "success"}
  ]
}
EOF

# summary.md - 验证摘要
cat > summary.md << 'EOF'
# Verification Summary
- All checks passed
EOF

# review.md - 审查结果
cat > review.md << 'EOF'
# Review
- Code quality: OK
- No issues found
EOF
```

#### Step 4: 提交并推送
```bash
git add .harness
git commit -m "chore: add harness files for $TASK_ID"
git push
```

#### Step 5: 生成并创建 PR
```bash
node scripts/harness/generate-pr-body.mjs $TASK_ID | gh pr create --body-file -
```

### 4. Pre-Push Hook（可选）

安装后会在每次 push 前自动验证：
```bash
cp scripts/hooks/pre-push .git/hooks/pre-push
chmod +x .git/hooks/pre-push
```

验证项：
- ✅ 所有 evidence 文件存在
- ✅ manifest.json 包含必需字段
- ✅ 文件结构完整

> `status` 是可选字段。一旦写了，就必须是 `planned | in-progress | completed | abandoned`
> 之一：schema（`scripts/task-manifest.mjs`）会拒绝 `done` / `complete` 之类的同义漂移，
> 以免将来任何 `status === 'completed'` 的查询静默漏掉已关闭的 packet。

### 5. 常见问题

#### Q: 为什么 CI 仍然失败？
A: 检查这些常见错误：
1. `manifest.json` 缺少必需字段（`goal`, `primaryLayer`, `scope`, `linkage`）
2. `scope` 不是对象格式（必须有 `in` 和 `out` 数组）
3. `linkage` 对象不完整
4. Evidence 文件路径不匹配

#### Q: 如何快速修复失败的 PR？
```bash
# 1. 查看具体错误
gh run view --log-failed | grep -E "invalid|must|missing"

# 2. 修复 manifest.json（参考 sonarcloud-s4666）

# 3. 重新生成 PR body
node scripts/harness/generate-pr-body.mjs <task-id> | gh pr edit <pr-number> --body-file -

# 4. 提交并推送
git add .harness
git commit --amend --no-edit
git push -f
```

#### Q: 可以简化这个流程吗？
A: 暂时不能。这是项目的企业级治理要求，必须遵守。但有了自动生成工具后，只需：
1. 复制 manifest 模板
2. 修改几个关键字段
3. 运行生成器
4. 一次通过 ✅

### 6. 示例对比

**之前（手动填写）**：
- ❌ 花费 30+ 分钟手写 PR body
- ❌ 遗漏字段导致 CI 失败
- ❌ 反复修改、推送、等待
- ❌ 浪费大量时间在格式上

**现在（自动生成）**：
- ✅ 5 分钟填写 manifest 关键字段
- ✅ 1 行命令生成完整 PR body
- ✅ 预推送自动验证
- ✅ CI 一次通过

### 7. 参考示例

完整工作示例：
- Manifest: `.harness/tasks/sonarcloud-s4666/manifest.json`
- Evidence: `.harness/evidence/sonarcloud-s4666/`
- PR: #291

复杂任务示例：
- `.harness/tasks/2026-07-15-code-review-remediation/`
- 展示了完整的企业级任务结构

## 关键要点

1. **Never 手写 PR body**：总是用生成器
2. **Always 参考示例**：复制 sonarcloud-s4666 的结构
3. **Use pre-push hook**：提前发现问题
4. **Keep manifest complete**：不要偷工减料
5. **Follow the structure**：这是项目标准，不是建议

---

更新时间: 2026-09-06
维护者: 项目团队
