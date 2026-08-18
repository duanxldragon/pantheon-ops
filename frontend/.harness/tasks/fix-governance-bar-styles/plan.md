# Task: 修复治理栏样式问题

**Task ID**: fix-governance-bar-styles  
**Priority**: P1  
**Created**: 2026-08-16  
**Status**: In Progress

---

## Context

用户反馈 pantheon-ops business 模块的治理栏样式有问题。经过代码分析发现：

1. **Icon 使用不一致**: CMDB Host、Deploy Package/Template 有 icon，但 BizScope、K8s 模块没有
2. **与 System 模块不一致**: System 模块（如 UserList）不使用 icon
3. **K8s 模块功能不完整**: 缺少侧边抽屉（GovernanceInsightDrawer）

---

## Chosen Approach

**方案 A: 统一移除 Icon**

**Why**:
- ✅ 与 System 模块保持视觉一致性
- ✅ 简化维护，避免对齐问题
- ✅ 快速修复（5分钟）
- ✅ 不引入新的样式复杂度

**Why not 方案 B (统一添加 Icon)**:
- ❌ 需要为所有页面设计 icon
- ❌ 需要修复潜在的样式冲突
- ❌ 与 System 模块产生视觉差异

---

## Implementation Steps

### Phase 1: 移除现有 Icon (5分钟)

修改以下文件，移除 `icon` 属性：

1. **CMDB Host**
   - 文件: `src/modules/business/cmdb/host/CmdbHostList.tsx`
   - 行号: ~475
   - 变更: 删除 `icon={<IconStorage />}`

2. **Deploy Package**
   - 文件: `src/modules/business/deploy/package/DeployPackageList.tsx`
   - 行号: ~457
   - 变更: 删除 `icon={<IconApps />}`

3. **Deploy Template**
   - 文件: `src/modules/business/deploy/template/DeployTemplateList.tsx`
   - 行号: 待确认
   - 变更: 删除 `icon={<IconCode />}` (如果有)

### Phase 2: 补全 K8s 模块治理栏 (15分钟)

为 K8s 三个页面添加完整的治理栏功能：

1. **Cluster**
   - 文件: `src/modules/business/k8s/cluster/ClusterList.tsx`
   - 添加: `GovernanceInsightDrawer` + `useGovernanceRail` hook
   
2. **Workload**
   - 文件: `src/modules/business/k8s/workload/WorkloadList.tsx`
   - 添加: `GovernanceInsightDrawer` + `useGovernanceRail` hook

3. **Release**
   - 文件: `src/modules/business/k8s/release/ReleaseList.tsx`
   - 添加: `GovernanceInsightDrawer` + `useGovernanceRail` hook

参考模板: `src/modules/system/user/UserList.tsx` (行 722-730)

### Phase 3: 验证 (10分钟)

- [ ] 浏览器目视检查所有页面
- [ ] 深色/浅色模式测试
- [ ] 响应式布局测试 (移动端/平板/桌面)
- [ ] Playwright 截图对比

---

## Acceptance Criteria

### Must Have
- ✅ 所有 business 模块页面的治理栏不使用 icon
- ✅ K8s 三个页面都有完整的治理栏（含侧边抽屉）
- ✅ 视觉风格与 System 模块一致
- ✅ 响应式布局正常

### Nice to Have
- ✅ Playwright 截图覆盖
- ✅ 跨浏览器测试通过

---

## Risk Assessment

**Low Risk** ⭐

- 仅删除属性，不修改核心逻辑
- 不影响数据处理
- 样式回退到默认布局
- 可立即回滚

---

## Dependencies

- Frontend dev server running (✅)
- Backend server running (✅)
- Playwright installed (⏳ 下载中)

---

## Rollback Plan

如果修复后发现问题：
1. Git revert 提交
2. 或手动恢复 `icon` 属性

---

## Implementation

由 Codex 执行代码修改。Claude 负责：
- ✅ 问题诊断
- ✅ 方案设计
- ✅ 测试验证
- ✅ 视觉审查

---

## Notes

- 诊断报告: `docs/governance-bar-diagnostic-report.md`
- 相关任务: #2 (修复治理栏样式问题)
