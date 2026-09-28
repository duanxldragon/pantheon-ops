# 组件样式规范

## 1. 样式隔离原则

### 1.1 样式文件组织

每个组件必须有独立的样式文件：

```
ComponentName/
  ├── index.tsx          # 组件逻辑
  ├── ComponentName.css  # 组件样式
  └── types.ts           # 类型定义
```

**禁止**：
- 在 `index.css` 中定义业务组件样式
- 在组件文件中使用内联样式对象（除非是动态计算的值）
- 跨组件直接引用其他组件的样式类名

### 1.2 BEM 命名约定

使用 BEM（Block Element Modifier）命名：

```css
/* Block */
.stat-card { }

/* Element */
.stat-card__title { }
.stat-card__value { }
.stat-card__trend { }

/* Modifier */
.stat-card--loading { }
.stat-card__trend--up { }
.stat-card__trend--down { }
```

**命名规则**：
- Block：组件名的 kebab-case 形式
- Element：双下划线 `__` 连接
- Modifier：双中划线 `--` 连接
- 避免嵌套超过 3 层

## 2. Token 使用规范

### 2.1 容器类型选择

根据组件的交互性质选择正确的 token 类别：

| 组件类型 | Token 类别 | 示例组件 |
|---------|-----------|---------|
| **交互容器** | `--container-interactive-*` | Input、Select、Picker、可编辑表单 |
| **展示容器** | `--container-display-*` | Card、Descriptions、只读表格、统计面板 |
| **操作容器** | `--container-action-*` | Button、操作工具栏、分页控件 |

### 2.2 交互容器样式

```css
.custom-input {
  /* 基础状态 */
  background-color: var(--container-interactive-bg);
  border: 1px solid var(--container-interactive-border);
  
  /* Hover 状态 */
  &:hover {
    background-color: var(--container-interactive-hover-bg);
  }
  
  /* Focus 状态 */
  &:focus-within {
    border-color: var(--container-interactive-focus-border);
    background-color: var(--container-interactive-bg);
  }
  
  /* Disabled 状态 */
  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}
```

### 2.3 展示容器样式

```css
.info-card {
  /* 提升层级的卡片 */
  background-color: var(--container-display-elevated);
  border: 1px solid var(--container-display-border);
  border-radius: var(--radius-md);
  
  /* 嵌套的次级面板 */
  &__section {
    background-color: var(--container-display-subtle);
    padding: var(--space-md);
  }
}
```

### 2.4 操作容器样式

```css
.toolbar {
  background-color: var(--container-action-bg);
  border: 1px solid var(--container-action-border);
  padding: var(--space-sm) var(--space-md);
  
  &__button {
    /* 使用 Arco Button 组件，它会自动应用 action token */
  }
}
```

## 3. 间距与布局

### 3.1 间距 Token

使用语义化的间距 token，不要硬编码数值：

```css
.component {
  /* ❌ 错误 */
  padding: 8px 12px;
  gap: 16px;
  
  /* ✅ 正确 */
  padding: var(--space-sm) var(--space-md);
  gap: var(--space-lg);
}
```

### 3.2 间距使用指南

| Token | 值 | 典型场景 |
|-------|---|---------|
| `--space-2xs` | 2px | 图标与文本的紧密间距 |
| `--space-xs` | 4px | 标签、徽章内部间距 |
| `--space-sm` | 8px | 同类控件间距、紧凑表单 |
| `--space-md` | 12px | 工具栏、表单行间距 |
| `--space-lg` | 16px | 卡片内边距、区段间距 |
| `--space-xl` | 24px | 页面内容区 padding |
| `--space-2xl` | 32px | 大区块间隔 |
| `--space-3xl` | 48px | 页面级分区 |

### 3.3 Flexbox 布局模式

优先使用 Flexbox，配合 `gap` 属性：

```css
.toolbar {
  display: flex;
  align-items: center;
  gap: var(--space-md);
}

.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: var(--space-lg);
}
```

## 4. 圆角规范

### 4.1 圆角 Token

```css
.component {
  /* ❌ 错误 */
  border-radius: 6px;
  
  /* ✅ 正确 */
  border-radius: var(--radius-md);
}
```

### 4.2 圆角使用指南

| Token | 值 | 适用场景 |
|-------|---|---------|
| `--radius-xs` | 4px | 标签、徽章 |
| `--radius-sm` | 4px | 紧凑按钮 |
| `--radius-md` | 6px | 卡片、输入框 |
| `--radius-lg` | 8px | 大型面板 |
| `--radius-xl` | 12px | 需要强分组感的表面 |
| `--radius-pill` | 999px | 胶囊标签、状态指示 |

**语义化别名**：
- `--radius-control`：输入框、选择器（= `--radius-md`）
- `--radius-action`：按钮、分页控件（= `--radius-sm`）
- `--radius-overlay`：弹窗、抽屉（= `--radius-lg`）

## 5. 颜色使用规范

### 5.1 文本颜色

```css
.component {
  /* 主要文本 */
  color: var(--text-primary);
  
  /* 次要描述文本 */
  &__description {
    color: var(--text-secondary);
  }
  
  /* 占位符、禁用文本 */
  &__placeholder {
    color: var(--text-tertiary);
  }
}
```

### 5.2 品牌色

```css
.highlight {
  /* 品牌主色 */
  color: var(--brand-primary);
  
  /* 柔和背景 */
  background-color: var(--brand-primary-soft);
  
  /* 极淡背景 */
  background-color: var(--brand-primary-faint);
}
```

### 5.3 禁止使用 Arco 原始 Token

**❌ 禁止**：
```css
.component {
  color: var(--color-text-1);
  background: var(--color-fill-2);
  border-color: var(--color-border-3);
}
```

**✅ 正确**：
```css
.component {
  color: var(--text-primary);
  background: var(--container-display-elevated);
  border-color: var(--container-display-border);
}
```

## 6. 响应式设计

### 6.1 断点定义

使用项目统一的断点：

```css
/* 移动端 */
@media (max-width: 768px) {
  .component {
    flex-direction: column;
    padding: var(--space-md);
  }
}

/* 平板端 */
@media (min-width: 769px) and (max-width: 1024px) {
  .component {
    grid-template-columns: repeat(2, 1fr);
  }
}

/* 桌面端 */
@media (min-width: 1025px) {
  .component {
    grid-template-columns: repeat(3, 1fr);
  }
}
```

### 6.2 移动端优化

```css
.toolbar {
  display: flex;
  gap: var(--space-md);
  
  @media (max-width: 768px) {
    flex-direction: column;
    gap: var(--space-sm);
  }
}
```

## 7. 状态样式

### 7.1 加载状态

```css
.component--loading {
  opacity: 0.6;
  pointer-events: none;
  position: relative;
  
  &::after {
    content: '';
    position: absolute;
    inset: 0;
    background: var(--container-display-elevated);
    opacity: 0.5;
  }
}
```

### 7.2 空状态

```css
.component--empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-3xl);
  color: var(--text-tertiary);
}
```

### 7.3 错误状态

```css
.component--error {
  border-color: rgb(var(--red-6));
  
  &__message {
    color: rgb(var(--red-6));
    font-size: 12px;
    margin-top: var(--space-xs);
  }
}
```

## 8. 性能优化

### 8.1 避免重复声明

**❌ 错误**：
```css
.card {
  background: white;
  border: 1px solid #ddd;
}

.info-card {
  background: white;
  border: 1px solid #ddd;
}
```

**✅ 正确**：
```css
.card,
.info-card {
  background: var(--container-display-elevated);
  border: 1px solid var(--container-display-border);
}
```

### 8.2 合理使用 CSS 变量

```css
.component {
  /* 本地 token，方便内部复用 */
  --local-padding: var(--space-md);
  --local-gap: var(--space-sm);
  
  padding: var(--local-padding);
  gap: var(--local-gap);
  
  &__item {
    padding: calc(var(--local-padding) / 2);
  }
}
```

## 9. 禁止模式

### 9.1 禁止内联样式

**❌ 错误**：
```tsx
<div style={{ padding: '12px', background: '#fff' }}>
  Content
</div>
```

**✅ 正确**：
```tsx
<div className="component">
  Content
</div>
```

```css
.component {
  padding: var(--space-md);
  background: var(--container-display-elevated);
}
```

**例外**：动态计算的值可以使用内联样式：
```tsx
<div style={{ width: `${progress}%` }}>
  Progress
</div>
```

### 9.2 禁止魔法数字

**❌ 错误**：
```css
.component {
  padding: 13px 17px;
  margin-top: 19px;
  border-radius: 5.5px;
}
```

**✅ 正确**：
```css
.component {
  padding: var(--space-md) var(--space-lg);
  margin-top: var(--space-lg);
  border-radius: var(--radius-md);
}
```

### 9.3 禁止渐变和特效

根据 DESIGN.md §7.9，禁止以下模式：

```css
/* ❌ 禁止 */
.component {
  background: linear-gradient(to right, #667eea, #764ba2);
  background: radial-gradient(circle, rgba(102, 126, 234, 0.1), transparent);
  box-shadow: 0 8px 18px rgba(102, 126, 234, 0.3);
}

/* ✅ 正确 */
.component {
  background: var(--container-display-elevated);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}
```

## 10. 检查清单

在提交代码前，确认：

- [ ] 所有颜色都使用 Pantheon token，没有 Arco 原始 token
- [ ] 所有间距都使用 `--space-*` token
- [ ] 所有圆角都使用 `--radius-*` token
- [ ] 容器根据交互性质选择了正确的 `--container-*` token
- [ ] 组件有独立的 CSS 文件，使用 BEM 命名
- [ ] 没有内联样式（除了动态计算值）
- [ ] 没有硬编码的魔法数字
- [ ] 没有渐变、光晕等禁止的视觉特效
- [ ] 移动端有适当的响应式适配
- [ ] 加载、空态、错误态有明确的样式定义

## 11. 示例：完整的组件样式

```css
/* StatCard.css */

.stat-card {
  /* 容器 */
  background-color: var(--container-display-elevated);
  border: 1px solid var(--container-display-border);
  border-radius: var(--radius-md);
  padding: var(--space-lg);
  
  /* 布局 */
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
  
  /* 响应式 */
  @media (max-width: 768px) {
    padding: var(--space-md);
  }
}

.stat-card__label {
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-card__value {
  color: var(--text-primary);
  font-size: 28px;
  font-weight: 600;
  font-feature-settings: 'tnum';
}

.stat-card__trend {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2xs);
  font-size: 13px;
  font-weight: 500;
}

.stat-card__trend--up {
  color: rgb(var(--green-6));
}

.stat-card__trend--down {
  color: rgb(var(--red-6));
}

.stat-card--loading {
  opacity: 0.6;
  pointer-events: none;
}

.stat-card--empty {
  justify-content: center;
  align-items: center;
  min-height: 120px;
  color: var(--text-tertiary);
}
```

```tsx
// StatCard.tsx
import './StatCard.css';

interface StatCardProps {
  label: string;
  value: string | number;
  trend?: {
    direction: 'up' | 'down';
    value: string;
  };
  loading?: boolean;
  empty?: boolean;
}

export function StatCard({ label, value, trend, loading, empty }: StatCardProps) {
  const className = [
    'stat-card',
    loading && 'stat-card--loading',
    empty && 'stat-card--empty',
  ].filter(Boolean).join(' ');
  
  if (empty) {
    return (
      <div className={className}>
        暂无数据
      </div>
    );
  }
  
  return (
    <div className={className}>
      <div className="stat-card__label">{label}</div>
      <div className="stat-card__value">{value}</div>
      {trend && (
        <div className={`stat-card__trend stat-card__trend--${trend.direction}`}>
          {trend.direction === 'up' ? '↑' : '↓'} {trend.value}
        </div>
      )}
    </div>
  );
}
```
