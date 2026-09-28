# 前端设计工程指南

## 1. 设计系统概述

Pantheon Base 使用**组件化设计系统**，包含三个层次：

```
Design Tokens (设计 Token)
    ↓
Component Styles (组件样式)
    ↓
UI Patterns (UI 模式)
```

### 1.1 设计原则

1. **一致性优先**：相同场景使用相同的组件和样式
2. **Token 驱动**：所有设计决策通过 Token 表达，不硬编码
3. **工具化美学**：克制、可信、专业的企业后台风格
4. **响应式就绪**：从设计阶段考虑多设备适配
5. **可维护性**：样式隔离、命名规范、文档齐全

### 1.2 与蓝鲸/主流设计系统的对比

| 维度 | 蓝鲸 MagicBox | Ant Design | Pantheon Base |
|------|--------------|------------|---------------|
| **基础组件库** | Vue 2/3 | React | Arco Design (React) |
| **Token 层** | CSS 变量 | Less 变量 | CSS 变量（三层语义） |
| **主题能力** | 多主题 | 多主题 | 四主题制 + 亮暗模式 |
| **间距体系** | 8px 栅格 | 8px 栅格 | 2/4/8/12/16/24/32/48px |
| **圆角策略** | 统一圆角 | 可配置 | 语义化圆角 token |
| **容器分类** | 无明确分类 | 无明确分类 | 交互/展示/操作三类 |
| **设计资源** | Sketch/Figma | Figma | Token + 文档 |

### 1.3 核心差异化

Pantheon Base 相比主流设计系统的独特点：

1. **三层容器语义**：明确区分交互容器、展示容器、操作容器
2. **Arco Token 隔离**：禁用 Arco 原始 token，强制使用 Pantheon token
3. **机械门禁**：通过 `check-ui-contract.mjs` 自动检查违规
4. **反模式清单**：明确禁止渐变、光晕、营销式设计

## 2. Design Tokens 设计

### 2.1 Token 架构

```
全局 Token (Global Tokens)
    ├── 色彩基元 (Color Primitives)
    │   ├── --brand-*        # 品牌色阶
    │   ├── --gray-*         # 灰度阶
    │   └── --semantic-*     # 语义色（red/green/yellow/blue）
    │
    ├── 间距基元 (Spacing Primitives)
    │   └── --space-*        # 2xs/xs/sm/md/lg/xl/2xl/3xl
    │
    └── 圆角基元 (Radius Primitives)
        └── --radius-*       # xs/sm/md/lg/xl/pill

语义 Token (Semantic Tokens)
    ├── 文本 (Text)
    │   ├── --text-primary
    │   ├── --text-secondary
    │   └── --text-tertiary
    │
    ├── 容器 (Container)
    │   ├── --container-interactive-*   # 交互容器
    │   ├── --container-display-*       # 展示容器
    │   └── --container-action-*        # 操作容器
    │
    └── 布局 (Layout)
        └── --app-bg / --panel-*
```

### 2.2 Token 命名规范

**颜色 Token**：
```
--{category}-{variant}-{state}

示例：
--container-interactive-bg           # 交互容器背景
--container-interactive-hover-bg     # 交互容器悬停背景
--container-interactive-focus-border # 交互容器聚焦边框
```

**间距 Token**：
```
--space-{size}

示例：
--space-sm   # 8px
--space-md   # 12px
--space-lg   # 16px
```

**圆角 Token**：
```
--radius-{size} 或 --radius-{semantic}

示例：
--radius-md        # 6px
--radius-control   # 输入控件圆角（= --radius-md）
--radius-action    # 按钮圆角（= --radius-sm）
```

### 2.3 添加新 Token 的流程

1. **确认需求**：这个样式值是否在多个组件中复用？
2. **选择层级**：是基元 token 还是语义 token？
3. **命名**：遵循命名规范
4. **定义**：在 `index.css` 的对应位置添加
5. **四主题适配**：在所有主题中定义对应的值
6. **暗色模式**：在 `prefers-color-scheme: dark` 中调整
7. **文档**：在 `DESIGN.md` 和 `COMPONENT_STYLING_GUIDE.md` 中记录

**示例**：添加一个新的警告容器

```css
/* 1. 在浅色模式定义 */
:root {
  /* ... 现有 token ... */
  
  /* 警告容器 */
  --container-warning-bg: color-mix(in srgb, rgb(var(--orange-2)) 50%, var(--panel-bg-solid));
  --container-warning-border: rgb(var(--orange-4));
}

/* 2. 在暗色模式调整 */
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    --container-warning-bg: color-mix(in srgb, rgb(var(--orange-9)) 15%, var(--panel-bg-solid));
    --container-warning-border: rgb(var(--orange-7));
  }
}

:root[data-theme="dark"] {
  --container-warning-bg: color-mix(in srgb, rgb(var(--orange-9)) 15%, var(--panel-bg-solid));
  --container-warning-border: rgb(var(--orange-7));
}
```

## 3. 组件开发流程

### 3.1 新建组件的标准步骤

```bash
# 1. 创建组件目录
frontend/src/components/
  └── YourComponent/
      ├── index.tsx
      ├── YourComponent.css
      └── types.ts

# 2. 编写组件逻辑 (index.tsx)
# 3. 编写组件样式 (YourComponent.css)
# 4. 编写类型定义 (types.ts)
# 5. 导出组件 (components/index.ts)
```

### 3.2 组件开发检查清单

**设计阶段**：
- [ ] 确认组件的交互性质（交互/展示/操作）
- [ ] 确认组件的状态（默认/hover/focus/disabled/loading/error）
- [ ] 确认组件的响应式行为
- [ ] 确认组件的无障碍需求

**开发阶段**：
- [ ] 使用正确的容器 token（interactive/display/action）
- [ ] 使用 BEM 命名约定
- [ ] 所有间距使用 `--space-*` token
- [ ] 所有圆角使用 `--radius-*` token
- [ ] 所有颜色使用 Pantheon token，不使用 Arco 原始 token
- [ ] 组件有独立的 CSS 文件
- [ ] 移动端有响应式适配

**测试阶段**：
- [ ] 浅色模式正常显示
- [ ] 暗色模式正常显示
- [ ] 四个主题（indigo/emerald/violet/slate）正常显示
- [ ] 移动端（< 768px）正常显示
- [ ] 各种状态（hover/focus/disabled）正常显示
- [ ] 通过 `npm run check:ui-contract` 检查

**文档阶段**：
- [ ] 在 `UI_PATTERN_LIBRARY.md` 中添加使用示例
- [ ] 如果是新模式，在 `COMPONENT_STYLING_GUIDE.md` 中记录

## 4. 样式调试技巧

### 4.1 Token 调试

查看当前生效的 token 值：

```javascript
// 在浏览器控制台
const root = document.documentElement;
const styles = getComputedStyle(root);

// 查看单个 token
styles.getPropertyValue('--container-interactive-bg');

// 查看所有 Pantheon token
[...document.styleSheets]
  .flatMap(sheet => [...sheet.cssRules])
  .filter(rule => rule.style?.getPropertyValue('--container-interactive-bg'))
```

### 4.2 主题切换调试

```javascript
// 切换主题
document.documentElement.setAttribute('data-pantheon-theme', 'emerald');

// 切换暗色模式
document.documentElement.setAttribute('data-theme', 'dark');
```

### 4.3 响应式调试

使用浏览器开发者工具：

1. 打开 DevTools (F12)
2. 点击设备工具栏图标（Ctrl+Shift+M）
3. 选择预设设备或自定义宽度
4. 检查 `@media (max-width: 768px)` 规则是否生效

### 4.4 常见问题排查

**问题 1：颜色不正确**
- 检查是否使用了 Arco 原始 token（`--color-text-1` 等）
- 检查 token 是否在所有主题中都有定义
- 检查暗色模式下的 token 值

**问题 2：间距不统一**
- 检查是否硬编码了数值（`padding: 12px` 而非 `var(--space-md)`）
- 检查是否使用了非标准间距值

**问题 3：组件在移动端显示异常**
- 检查是否有 `@media (max-width: 768px)` 规则
- 检查父容器是否有 `overflow` 限制
- 检查表格是否设置了 `scroll={{ x: true }}`

**问题 4：样式被覆盖**
- 检查 CSS 优先级（特异性）
- 检查是否有重复的类名
- 使用 DevTools 的 "Computed" 面板查看最终生效的样式

## 5. 设计协作流程

### 5.1 设计师 → 开发者交接

**设计师提供**：
1. 设计稿（Figma/Sketch）
2. 间距标注（使用 8px 栅格）
3. 颜色值（十六进制）
4. 字体规格（字号、行高、字重）
5. 交互状态（hover/focus/disabled）
6. 响应式断点

**开发者职责**：
1. 将设计稿的颜色映射到现有 token
2. 将间距值映射到 `--space-*` token
3. 将圆角值映射到 `--radius-*` token
4. 识别组件的交互性质，选择正确的容器 token
5. 实现响应式布局
6. 补充设计稿中未明确的状态样式

### 5.2 颜色映射指南

设计稿颜色 → Pantheon Token：

| 设计意图 | 十六进制示例 | Pantheon Token |
|---------|------------|----------------|
| 主要文本 | `#1D2129` | `--text-primary` |
| 次要文本 | `#4E5969` | `--text-secondary` |
| 占位符 | `#86909C` | `--text-tertiary` |
| 品牌色 | `#6366F1` | `--brand-primary` |
| 卡片背景 | `#FFFFFF` | `--container-display-elevated` |
| 输入框背景 | `#F7F8FA` | `--container-interactive-bg` |
| 边框 | `#E5E6EB` | `--panel-border` 或 `--container-*-border` |

**如果设计稿中的颜色没有对应的 token**：
1. 询问设计师是否可以使用最接近的现有 token
2. 如果确实需要新颜色，按照 §2.3 流程添加新 token
3. 避免直接硬编码十六进制值

### 5.3 间距映射指南

设计稿间距 → Pantheon Token：

| 设计稿值 | Pantheon Token | 备注 |
|---------|---------------|------|
| 2px | `--space-2xs` | 极紧密间距 |
| 4px | `--space-xs` | 标签内部 |
| 8px | `--space-sm` | 同类控件 |
| 12px | `--space-md` | 表单行 |
| 16px | `--space-lg` | 卡片内边距 |
| 24px | `--space-xl` | 页面内容区 |
| 32px | `--space-2xl` | 大区块 |
| 48px | `--space-3xl` | 页面级分区 |

**如果设计稿使用了非标准间距（如 10px、14px）**：
1. 询问设计师是否可以调整为 8px 或 12px
2. 如果确实需要，可以使用 `calc(var(--space-sm) + 2px)`
3. 避免添加过多非标准间距 token

## 6. 设计质量保证

### 6.1 机械门禁

Pantheon Base 有两个自动检查脚本：

**`check-ui-contract.mjs`**：
- 扫描所有 `.css` 和 `.tsx` 文件
- 检查是否使用了禁止的 Arco token
- 检查是否使用了禁止的视觉特效（渐变、光晕）
- 检查是否使用了非标准字重
- 在 `prebuild` 时自动执行

**`check-shell-visual-contract.mjs`**：
- 检查壳层（登录页、Layout）的结构完整性
- 确保关键元素存在且符合规范

**豁免机制**：
如果某个样式确实需要违规（如特殊的第三方库样式），可以添加行内注释：

```css
.special-component {
  background: linear-gradient(to right, #667eea, #764ba2); /* ui-contract-allow: gradient */
}
```

在 PR 中必须说明豁免理由。

### 6.2 人工审查清单

在提交 PR 前，手动检查：

**视觉一致性**：
- [ ] 与现有页面风格一致
- [ ] 四个主题下都正常显示
- [ ] 暗色模式下阅读舒适
- [ ] 没有明显的视觉突兀

**交互流畅性**：
- [ ] Hover 状态平滑过渡
- [ ] Focus 状态清晰可见
- [ ] Disabled 状态明确不可操作
- [ ] Loading 状态有明确反馈

**响应式适配**：
- [ ] 移动端布局合理
- [ ] 表格在窄屏下可横向滚动
- [ ] 工具栏在窄屏下不重叠
- [ ] 表单在移动端使用垂直布局

**无障碍**：
- [ ] 交互元素可用键盘操作
- [ ] 表单控件有正确的 label
- [ ] 颜色不作为唯一信息传达方式
- [ ] 对比度符合 WCAG AA 标准

### 6.3 性能检查

- [ ] CSS 文件大小合理（< 50KB）
- [ ] 没有大量重复的样式规则
- [ ] 动画使用 `transform` 和 `opacity`，避免触发重排
- [ ] 图片使用了适当的格式和尺寸

## 7. 迁移与重构指南

### 7.1 旧代码迁移到新 Token 体系

**步骤 1：识别旧样式**
```bash
# 搜索硬编码颜色
grep -r "#[0-9a-fA-F]\{6\}" frontend/src --include="*.css"

# 搜索 Arco 原始 token
grep -r "var(--color-" frontend/src --include="*.css"
```

**步骤 2：建立映射表**

| 旧样式 | 新 Token |
|-------|---------|
| `background: white;` | `background: var(--container-display-elevated);` |
| `color: var(--color-text-1);` | `color: var(--text-primary);` |
| `border: 1px solid var(--color-border-2);` | `border: 1px solid var(--panel-border);` |
| `padding: 12px 16px;` | `padding: var(--space-md) var(--space-lg);` |

**步骤 3：批量替换**

使用 VS Code 的全局搜索替换：
1. 搜索：`var\(--color-text-1\)`
2. 替换：`var(--text-primary)`
3. 确认每个替换是否合理

**步骤 4：验证**

1. 运行 `npm run check:ui-contract`
2. 在浏览器中测试所有主题
3. 测试暗色模式
4. 测试移动端

### 7.2 组件重构指南

**重构场景 1：组件样式分散**

问题：组件样式写在 `index.css` 或内联样式中

解决：
1. 创建 `ComponentName.css`
2. 将样式迁移到独立文件
3. 使用 BEM 命名
4. 在 `index.tsx` 中 import

**重构场景 2：颜色不一致**

问题：同类元素使用了不同的颜色值

解决：
1. 识别语义（交互容器 vs 展示容器）
2. 统一使用对应的 token
3. 删除重复的颜色定义

**重构场景 3：间距混乱**

问题：间距值五花八门（9px、13px、17px 等）

解决：
1. 将所有间距映射到最接近的标准 token
2. 与设计师确认调整后的视觉效果
3. 统一使用 `--space-*` token

## 8. 常见场景速查

### 8.1 新增一个列表页

1. 复制 `UI_PATTERN_LIBRARY.md` 中的列表页模板
2. 使用 `SearchToolbar` 组件
3. 表格包裹在 `Card` 中
4. 操作列使用 `Space` 组件
5. 删除操作添加二次确认

### 8.2 新增一个表单页

1. 复制 `UI_PATTERN_LIBRARY.md` 中的表单页模板
2. 表单 `layout="vertical"`
3. 表单最大宽度 640px
4. 底部操作栏有分割线
5. 提交按钮显示 loading 状态

### 8.3 新增一个统计卡片

1. 使用 `--container-display-elevated` 作为背景
2. 使用 `--container-display-border` 作为边框
3. 数值使用 `font-feature-settings: 'tnum'` 保证等宽
4. 趋势指示使用语义色（green-6 / red-6）
5. 圆角使用 `--radius-md`

### 8.4 新增一个自定义输入框

1. 使用 `--container-interactive-bg` 作为背景
2. 使用 `--container-interactive-border` 作为边框
3. Hover 使用 `--container-interactive-hover-bg`
4. Focus 使用 `--container-interactive-focus-border`
5. 圆角使用 `--radius-control`

## 9. 参考资源

### 9.1 内部文档

- `DESIGN.md`：总体设计文档
- `COMPONENT_STYLING_GUIDE.md`：组件样式规范
- `UI_PATTERN_LIBRARY.md`：UI 模式库
- `frontend/docs/FRONTEND_UI_SPEC.md`：前端 UI 规范

### 9.2 外部参考

- [Arco Design 官方文档](https://arco.design/)
- [CSS Variables MDN](https://developer.mozilla.org/en-US/docs/Web/CSS/Using_CSS_custom_properties)
- [BEM 命名规范](https://getbem.com/)
- [WCAG 无障碍指南](https://www.w3.org/WAI/WCAG21/quickref/)

### 9.3 工具推荐

- **VS Code 插件**：
  - CSS Variable Autocomplete
  - Stylelint
  - Prettier
- **浏览器插件**：
  - React DevTools
  - Arco DevTools（如有）
- **设计工具**：
  - Figma（推荐）
  - Sketch

## 10. FAQ

**Q: 为什么禁用 Arco 原始 token？**
A: 为了确保主题切换时所有组件都使用统一的 Pantheon token，避免颜色不一致。

**Q: 什么时候可以使用内联样式？**
A: 只有动态计算的值（如进度条宽度、拖拽位置）可以使用内联样式。

**Q: 如何添加新的间距值？**
A: 优先使用现有的 `--space-*` token。如果确实需要，可以使用 `calc()` 计算，或添加新的 token。

**Q: 移动端断点为什么是 768px？**
A: 这是行业通用的平板/手机分界线，与主流设计系统（Bootstrap、Tailwind）保持一致。

**Q: 如何处理第三方库的样式？**
A: 尽量使用第三方库自带的主题配置。如果必须覆盖，在全局 `index.css` 中统一处理，并添加注释说明理由。

**Q: 设计稿要求的颜色与现有 token 不匹配怎么办？**
A: 先与设计师沟通，看是否可以使用最接近的现有 token。如果确实需要新颜色，按照 §2.3 流程添加新 token。

**Q: 如何确保暗色模式下的可读性？**
A: 使用 Pantheon 的 `--text-*` token，系统已经为暗色模式优化了对比度。如果自定义颜色，确保对比度 ≥ 4.5:1（WCAG AA）。
