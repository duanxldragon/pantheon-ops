# 治理栏样式诊断报告

## 问题描述

用户反馈：pantheon-ops business 模块的治理栏样式有问题。

## 调查过程

### 1. 样式源文件检查

**主样式文件**: `frontend/src/modules/system/components/shared/list-page.css`
- 治理栏样式定义：第 77-185 行
- 响应式断点：920px

**关键样式规则**:

```css
.governance-summary-bar {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr) auto;
  gap: 12px 16px;
  align-items: center;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--panel-border) 94%, transparent);
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--panel-muted) 48%, var(--surface-lift));
}
```

**响应式布局** (< 920px):

```css
@media (max-width: 920px) {
  .governance-summary-bar {
    grid-template-columns: 1fr;  /* 单列布局 */
  }

  .governance-summary-bar__metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));  /* 指标卡改为2列 */
  }
}
```

### 2. Business 模块使用情况

**样式引用检查**:
- ✅ CMDB Host: 正确引用 `../../../system/components/shared/list-page.css`
- ✅ BizScope: 正确引用
- ✅ Deploy Package: 正确引用
- ✅ K8s Cluster: 正确引用

**无样式覆盖**: business 模块的专属样式文件（`cmdb.css`, `deploy.css`）均未覆盖治理栏样式。

### 3. Icon 使用一致性检查

| 模块 | Icon 配置 | 组件 |
|------|-----------|------|
| CMDB Host | ✅ `<IconStorage />` | CmdbHostList.tsx |
| CMDB Group | ❌ 无 | - |
| CMDB Label | ❌ 无 | - |
| BizScope | ❌ 无 | BizScopeList.tsx |
| Deploy Package | ✅ `<IconApps />` | DeployPackageList.tsx |
| Deploy Template | ❌ 无 | - |
| Deploy Task | ❌ 无 | - |
| K8s Cluster | ❌ 无 | ClusterList.tsx |
| K8s Workload | ❌ 无 | WorkloadList.tsx |
| K8s Release | ❌ 无 | - |

**发现**: Icon 使用不一致，但这不是样式问题，而是功能不完整。

### 4. 治理栏组件实现对比

**System 模块 (UserList.tsx)** - 标准实现:
```tsx
<GovernanceSummaryBar
  eyebrow={t('operations.user.menu')}
  title={t('system.user.hero.title')}
  description={t('system.user.hero.description')}
  metrics={heroStats}
  action={<GovernanceRailToggleButton />}
/>
```

**Business 模块 (CmdbHostList.tsx)** - 带 Icon:
```tsx
<GovernanceSummaryBar
  icon={<IconStorage />}
  eyebrow={t('operations.cmdb.host.menu')}
  title={t('business.cmdb.host.hero.title')}
  description={t('business.cmdb.host.hero.description')}
  metrics={heroStats}
  action={<GovernanceRailToggleButton />}
/>
```

### 5. 可能的样式问题推测

#### 问题 A: Icon 布局异常
- **症状**: 当 `icon` prop 存在时，布局可能错位
- **原因**: `.governance-summary-bar__icon` 的 `flex: 0 0 auto` 可能与 grid 布局冲突
- **影响范围**: CMDB Host, Deploy Package

#### 问题 B: 指标卡密度过高
- **症状**: 指标卡 `min-height: 60px` 可能在某些内容下显得拥挤
- **原因**: 设计收敛为"两行文案"后，60px 可能不够
- **影响范围**: 所有使用治理栏的页面

#### 问题 C: 响应式断点不合理
- **症状**: 在中等屏幕（920-1200px）时布局可能不理想
- **原因**: 920px 断点可能过早触发单列布局
- **影响范围**: 所有使用治理栏的页面

#### 问题 D: 颜色对比度不足
- **症状**: `color-mix()` 可能导致边框/背景对比度不足
- **原因**: `var(--panel-border) 94%` 透明度过高
- **影响范围**: 所有使用治理栏的页面

## 待确认事项

需要实际浏览器渲染截图来确认具体问题：

1. **Icon 显示位置**: icon 是否正确显示在 eyebrow 上方？
2. **Grid 布局**: 三列布局（copy | metrics | action）是否正常？
3. **指标卡高度**: 60px 是否足够容纳内容？
4. **响应式断点**: 920px 时的单列布局是否合理？
5. **颜色对比**: 边框和背景是否清晰可见？

## 下一步行动

1. ✅ 前端服务已启动: http://localhost:5173
2. ✅ 后端服务已启动: http://localhost:8080
3. ⏳ 需要浏览器截图或用户描述具体问题
4. ⏳ 根据实际问题制定修复方案

## 技术栈信息

- **CSS**: 使用 CSS Grid + color-mix()
- **组件**: React + TypeScript
- **样式变量**: CSS custom properties (--brand-primary, --surface-lift 等)
- **响应式**: 单一断点 920px
