# Token 迁移指南

## 目标读者

- 前端开发者
- UI 设计师
- Code Reviewer

---

## 一、为什么需要 Token 系统？

### 问题：硬编码导致的不一致性

```css
/* ❌ 不同组件使用不同的灰色 */
.card-1 { color: #666; }
.card-2 { color: #777; }
.card-3 { color: #86909C; }

/* ❌ 间距随意使用，没有规律 */
.section-1 { padding: 15px; }
.section-2 { padding: 18px; }
.section-3 { padding: 20px; }
```

### 解决方案：Token 系统

```css
/* ✅ 统一使用语义化 token */
.card-1,
.card-2,
.card-3 {
  color: var(--text-secondary);
}

/* ✅ 标准化的间距阶梯 */
.section-1,
.section-2,
.section-3 {
  padding: var(--space-lg); /* 16px */
}
```

### Token 系统的优势

1. **一致性**：所有组件使用相同的颜色/间距
2. **可维护性**：修改 token 定义，全局生效
3. **主题切换**：四主题 + 暗色模式自动适配
4. **AI 友好**：明确的约束，减少"跑偏"

---

## 二、Pantheon Token 体系

### 2.1 颜色 Token

#### 文本颜色

| Token | 用途 | 示例场景 |
|-------|------|---------|
| `--text-primary` | 标题、正文 | 页面标题、段落文本 |
| `--text-secondary` | 描述、辅助信息 | 表单 label、卡片副标题 |
| `--text-tertiary` | 占位符、禁用态 | Input placeholder、禁用按钮 |

```css
/* ✅ 正确使用 */
.page-title { color: var(--text-primary); }
.form-label { color: var(--text-secondary); }
.input::placeholder { color: var(--text-tertiary); }
```

#### 容器背景

| Token | 用途 | 示例场景 |
|-------|------|---------|
| `--panel-bg-solid` | 卡片、弹窗、表单容器 | Card、Modal、Form |
| `--panel-border` | 卡片边框、分割线 | Card border、Divider |
| `--panel-muted` | 次级面板背景 | 表格偶行、嵌套卡片 |
| `--app-bg` | 页面最外层底色 | body background |
| `--surface-lift` | 悬浮层、弹窗 | Dropdown、Tooltip |

```css
/* ✅ 正确使用 */
.card {
  background: var(--panel-bg-solid);
  border: 1px solid var(--panel-border);
}

.table tbody tr:nth-child(even) {
  background: var(--panel-muted);
}
```

#### 品牌色

| Token | 用途 | 示例场景 |
|-------|------|---------|
| `--brand-primary` | 主按钮、选中态、链接 | Primary Button、Active Tab |
| `--brand-primary-hover` | 主按钮 hover | Button:hover |

```css
/* ✅ 正确使用 */
.btn-primary {
  background: var(--brand-primary);
}

.btn-primary:hover {
  background: var(--brand-primary-hover);
}
```

#### 语义色（Arco 提供）

| Token | 用途 | 示例场景 |
|-------|------|---------|
| `var(--color-error)` | 错误文本、边框 | 表单错误提示 |
| `var(--color-error-bg)` | 错误背景 | Alert 背景 |
| `var(--color-success)` | 成功文本、边框 | 成功提示 |
| `var(--color-success-bg)` | 成功背景 | Success Alert |
| `var(--color-warning)` | 警告文本、边框 | 警告提示 |
| `var(--color-warning-bg)` | 警告背景 | Warning Alert |

```css
/* ✅ 正确使用 - 这些是语义色，保留使用 */
.error-message {
  color: var(--color-error);
  border-color: color-mix(in srgb, var(--color-error) 24%, var(--panel-border));
  background: color-mix(in srgb, var(--color-error-bg) 42%, var(--surface-lift));
}
```

**⚠️ 重要**: `--color-error/success/warning/info` 等语义色是 **Arco 提供的色阶别名**，不是需要禁用的 token！

### 2.2 间距 Token

| Token | 值 | 用途 | 示例场景 |
|-------|---|------|---------|
| `--space-2xs` | 2px | 极紧密微间距 | 文本与图标的紧密间距 |
| `--space-xs` | 4px | 标签内部 | Tag padding、紧密图标间距 |
| `--space-sm` | 8px | 同类控件 | 按钮组间距、表单控件间距 |
| `--space-md` | 12px | 表单行、局部区段 | Form field margin、工具栏内边距 |
| `--space-lg` | 16px | 卡片内边距 | Card padding、标准区段间距 |
| `--space-xl` | 24px | 页面内容区 | Page padding、大区段间距 |
| `--space-2xl` | 32px | 大区块 | Section 间距 |
| `--space-3xl` | 48px | 页面级分区 | 页面头部与内容的间距 |

```css
/* ✅ 正确使用 */
.card {
  padding: var(--space-lg); /* 16px */
}

.form-field {
  margin-bottom: var(--space-md); /* 12px */
}

.button-group {
  gap: var(--space-sm); /* 8px */
}
```

### 2.3 圆角 Token

| Token | 值 | 用途 |
|-------|---|------|
| `--radius-xs` | 4px | 标签、徽章 |
| `--radius-sm` | 4px | 紧凑按钮 |
| `--radius-md` | 6px | 输入控件、卡片 |
| `--radius-lg` | 8px | 较大面板 |
| `--radius-xl` | 12px | 大型表面 |
| `--radius-overlay` | 8px | 弹窗、抽屉 |
| `--radius-control` | `var(--radius-md)` | 输入框、选择器 |
| `--radius-action` | `var(--radius-sm)` | 按钮、分页 |
| `--radius-pill` | 999px | 胶囊标签 |

```css
/* ✅ 正确使用 */
.card {
  border-radius: var(--radius-md);
}

.button {
  border-radius: var(--radius-action);
}

.tag {
  border-radius: var(--radius-pill);
}
```

---

## 三、迁移规则

### 规则 1: 迁移标准值，保留精细调整

#### ✅ 应该迁移的（标准值）

```css
/* Before */
.card {
  padding: 16px;
  margin: 12px;
  gap: 8px;
}

/* After */
.card {
  padding: var(--space-lg);
  margin: var(--space-md);
  gap: var(--space-sm);
}
```

#### ❌ 不应该迁移的（精细调整）

```css
/* Before - 保持不变 */
.app-shell__brand {
  padding: 14px 14px 14px 12px; /* 设计师精细调整 */
}

/* After - 保持不变 */
.app-shell__brand {
  padding: 14px 14px 14px 12px; /* 精细调整，保留硬编码 */
}
```

#### 允许的精细调整值

- `6px` - 菜单项微调
- `10px` - 菜单容器微调
- `14px` - 品牌区微调
- `18px` - 特殊区域微调
- `20px` - 接近 xl 的微调
- `28px` - 接近 2xl 的微调

**原因**: 这些是视觉设计师经过精心调整的值，强行对齐到标准 token 会破坏视觉平衡。

### 规则 2: 禁用 Arco 文本/边框/填充 Token

#### ❌ 禁止使用（Arco 原始 Token）

```css
/* ❌ 错误 - 使用 Arco 原始 token */
.text {
  color: var(--color-text-1);
  color: var(--color-text-2);
}

.card {
  border: 1px solid var(--color-border-2);
  background: var(--color-fill-1);
}
```

#### ✅ 应该使用（Pantheon Token）

```css
/* ✅ 正确 - 使用 Pantheon token */
.text {
  color: var(--text-primary);
  color: var(--text-secondary);
}

.card {
  border: 1px solid var(--panel-border);
  background: var(--panel-bg-solid);
}
```

### 规则 3: 语义色 Token 保留使用

#### ✅ 正确使用（语义色）

```css
/* ✅ 正确 - 语义色是合法的 */
.error-alert {
  color: var(--color-error);
  border-color: var(--color-error);
  background: var(--color-error-bg);
}

.success-message {
  color: var(--color-success);
  background: var(--color-success-bg);
}
```

**原因**: 这些不是需要禁用的 token，它们是：
- `var(--color-error)` = `rgb(var(--red-6))`
- `var(--color-success)` = `rgb(var(--green-6))`
- `var(--color-warning)` = `rgb(var(--orange-6))`
- `var(--color-info)` = `rgb(var(--blue-6))`

---

## 四、机械门禁

### 门禁规则

文件: `frontend/scripts/check-ui-contract.mjs`

```bash
# 运行门禁检查
npm run check:ui-contract

# 或
node scripts/check-ui-contract.mjs
```

### 检查的违规

1. **no-radial-gradient**: 禁止 `radial-gradient()` 光晕装饰
2. **no-linear-gradient**: 禁止 `linear-gradient()` 大面积渐变
3. **standard-font-weight**: `font-weight` 必须是 400/500/600/700
4. **no-inter-font**: 禁止使用 Inter 字体
5. **no-raw-arco-token**: 禁止使用 `--color-text-1`、`--color-border-2` 等
6. **no-module-hex-color**: 模块 CSS 禁止硬编码十六进制颜色

### 白名单

#### 语义色白名单

以下 Arco token 是**允许**使用的：

```javascript
const ALLOWED_ARCO_SEMANTIC_TOKENS = [
  'color-error',
  'color-error-bg',
  'color-success',
  'color-success-bg',
  'color-warning',
  'color-warning-bg',
  'color-info',
  'color-info-bg',
];
```

#### 精细调整间距白名单

以下间距值是**允许**硬编码的：

```javascript
const FINE_TUNED_SPACING = [
  '1px',  // 边框
  '2px',  // 极紧密
  '6px',  // 菜单项微调
  '10px', // 菜单容器微调
  '14px', // 品牌区微调
  '18px', // 特殊区域微调
  '20px', // 接近 xl 的微调
  '28px', // 接近 2xl 的微调
];
```

### 豁免语法

当某条规则**确实不适用**时，可以使用行内豁免：

```css
/* ui-contract-allow: no-linear-gradient */
.special-background {
  background: linear-gradient(to right, #000, #333);
}
```

**⚠️ 重要**: 
- 豁免必须在 PR body 中说明理由
- 不应该滥用豁免绕过规范

---

## 五、迁移检查清单

### 新增组件

- [ ] 颜色：使用 Pantheon token（`--text-primary`、`--panel-bg-solid` 等）
- [ ] 间距：使用标准 token（`--space-lg` 等），微调值可以硬编码
- [ ] 圆角：使用 token（`--radius-md` 等）
- [ ] 字体：使用 `system-ui` 栈，权重 400/500/600/700
- [ ] 语义色：直接使用 `var(--color-error)` 等

### 修改现有组件

- [ ] 检查是否有 Arco 原始 token（`--color-text-1` 等）
- [ ] 将标准间距值迁移到 token（16px → `var(--space-lg)`）
- [ ] 保留精细调整值（6px、10px、14px 等）
- [ ] 运行 `npm run check:ui-contract` 确认无违规

### Code Review

- [ ] 检查是否使用了禁止的 token
- [ ] 检查是否有不必要的硬编码颜色
- [ ] 确认精细调整值有合理理由
- [ ] 验证四主题 + 暗色模式下的显示效果

---

## 六、常见问题

### Q1: 为什么不能使用 `--color-text-1`？

**A**: `--color-text-1` 是 Arco 的内部 token，当我们切换主题时，它不会跟随 Pantheon 主题变化。应该使用 `--text-primary`，它在四个主题中都有正确的定义。

### Q2: 为什么 `var(--color-error)` 可以用？

**A**: `--color-error` 不是文本/边框/填充 token，它是 **Arco 提供的语义色阶**，等同于 `rgb(var(--red-6))`。这是合法且推荐的用法。

### Q3: 为什么有些间距可以硬编码？

**A**: 设计师会根据视觉需要进行微调（如 14px、10px），这些精细调整值是专业判断的结果，不应该强行对齐到标准 token。Token 系统是为了一致性，不是为了消灭所有硬编码。

### Q4: 如何判断一个间距值应该迁移还是保留？

**决策树**:

```
该间距值是否严格等于标准 token？
  ├─ 是（如 16px = --space-lg）→ 迁移到 token
  └─ 否（如 14px）→ 检查是否是精细调整
       ├─ 是（视觉微调）→ 保留硬编码
       └─ 否（随意硬编码）→ 调整到最接近的标准值，然后迁移
```

### Q5: 机械门禁报错了怎么办？

**步骤**:

1. **理解错误**: 阅读错误信息，确认违反了哪条规则
2. **修复违规**: 按照规则要求修改代码
3. **合理豁免**: 如果规则确实不适用，使用 `ui-contract-allow` 并在 PR 中说明
4. **重新运行**: `npm run check:ui-contract`

### Q6: 如何在 color-mix() 中使用 token？

**示例**:

```css
/* ✅ 正确 - 使用 Pantheon token */
.card {
  background: color-mix(in srgb, var(--brand-primary) 6%, var(--panel-bg-solid));
  border-color: color-mix(in srgb, var(--brand-primary) 12%, var(--panel-border));
}

/* ✅ 正确 - 使用语义色 token */
.error-card {
  background: color-mix(in srgb, var(--color-error-bg) 42%, var(--surface-lift));
  border-color: color-mix(in srgb, var(--color-error) 24%, var(--panel-border));
}
```

---

## 七、迁移示例

### 示例 1: 简单卡片组件

```css
/* Before - 硬编码 */
.simple-card {
  padding: 16px;
  margin-bottom: 12px;
  background: #FFFFFF;
  border: 1px solid #E5E6EB;
  border-radius: 6px;
  color: #1D2129;
}

.simple-card__title {
  font-size: 14px;
  font-weight: 600;
  color: #1D2129;
  margin-bottom: 8px;
}

.simple-card__description {
  font-size: 12px;
  color: #86909C;
}

/* After - 使用 Token */
.simple-card {
  padding: var(--space-lg);
  margin-bottom: var(--space-md);
  background: var(--panel-bg-solid);
  border: 1px solid var(--panel-border);
  border-radius: var(--radius-md);
  color: var(--text-primary);
}

.simple-card__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: var(--space-sm);
}

.simple-card__description {
  font-size: 12px;
  color: var(--text-secondary);
}
```

### 示例 2: 表单组件

```css
/* Before - 混合硬编码和 Arco token */
.form-field {
  margin-bottom: 16px;
}

.form-label {
  display: block;
  margin-bottom: 8px;
  font-size: 13px;
  color: var(--color-text-2); /* ❌ Arco 原始 token */
}

.form-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--color-border-2); /* ❌ Arco 原始 token */
  border-radius: 4px;
  background: var(--color-fill-1); /* ❌ Arco 原始 token */
}

.form-error {
  margin-top: 4px;
  font-size: 12px;
  color: #F53F3F; /* ❌ 硬编码颜色 */
}

/* After - 使用 Pantheon Token */
.form-field {
  margin-bottom: var(--space-lg);
}

.form-label {
  display: block;
  margin-bottom: var(--space-sm);
  font-size: 13px;
  color: var(--text-secondary); /* ✅ Pantheon token */
}

.form-input {
  width: 100%;
  padding: var(--space-sm) var(--space-md);
  border: 1px solid var(--container-interactive-border); /* ✅ 容器 token */
  border-radius: var(--radius-control);
  background: var(--container-interactive-bg); /* ✅ 容器 token */
}

.form-input:focus {
  border-color: var(--container-interactive-focus-border); /* ✅ 容器 token */
}

.form-error {
  margin-top: var(--space-xs);
  font-size: 12px;
  color: var(--color-error); /* ✅ 语义色 token */
}
```

### 示例 3: 精细调整保留

```css
/* Before - 设计师调整的值 */
.app-sidebar {
  padding: 14px 14px 14px 12px; /* 不对称 padding，视觉平衡调整 */
}

.menu-item {
  padding: 6px 12px; /* 6px 是紧凑菜单的精细调整 */
}

/* After - 保留精细调整，迁移标准值 */
.app-sidebar {
  padding: 14px 14px 14px var(--space-md); /* 保留 14px，迁移 12px */
}

.menu-item {
  padding: 6px var(--space-md); /* 保留 6px，迁移 12px */
}
```

---

## 八、参考文档

- [DESIGN.md](../../DESIGN.md) - 总体设计规范
- [FRONTEND_UI_SPEC.md](../designs/FRONTEND_UI_SPEC.md) - 前端 UI 规范
- [COMPONENT_STYLING_GUIDE.md](./COMPONENT_STYLING_GUIDE.md) - 组件样式规范
- [UI_PATTERN_LIBRARY.md](./UI_PATTERN_LIBRARY.md) - UI 模式库
- [DESIGN_ENGINEERING_GUIDE.md](./DESIGN_ENGINEERING_GUIDE.md) - 设计工程指南

---

**文档版本**: 1.0  
**最后更新**: 2026-09-03  
**维护者**: Pantheon 前端团队
