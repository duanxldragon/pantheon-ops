# Task 10: 组件迁移扫描报告

## 扫描时间
2026-09-03

## 扫描范围
`frontend/src/` 下所有 `.css` 文件

---

## 一、Arco 原始 Token 使用情况

### 总体统计
- **使用文件数**: 6 个
- **总使用次数**: 89 次

### 详细清单

| 文件 | 使用次数 | 优先级 | 影响范围 |
|------|---------|-------|---------|
| `index.css` | 42 | P0 | 全局样式 |
| `modules/platform/dashboard.css` | 20 | P1 | 工作台 |
| `core/layout/index.css` | 15 | P0 | 应用壳层 |
| `components/operational/operational.css` | 6 | P2 | 操作日志组件 |
| `modules/system/components/shared/list-page.css` | 5 | P1 | 系统管理列表页 |
| `modules/lowcode/generator/pages/ModuleWizard.css` | 1 | P2 | 低代码生成器 |

### 主要使用的 Arco Token

基于扫描结果，最常用的 Arco token：

1. `var(--color-error)` / `var(--color-error-bg)` - 错误状态
2. `var(--color-success)` / `var(--color-success-bg)` - 成功状态
3. `var(--color-warning)` / `var(--color-warning-bg)` - 警告状态
4. `var(--color-info)` - 信息提示

**迁移方向**：
- 语义色应该保留使用 `rgb(var(--red-6))`、`rgb(var(--green-6))` 等
- 这些是 Arco 提供的色阶，不是需要禁用的文本/边框 token
- 真正需要迁移的是 `--color-text-1`、`--color-border-2` 等

---

## 二、需要重点迁移的 Token

### 文本颜色 Token（未发现）
扫描未发现 `--color-text-1/2/3/4` 的使用，说明文本颜色已经迁移完成 ✅

### 边框颜色 Token（未发现）
扫描未发现 `--color-border-1/2/3/4` 的使用，说明边框颜色已经迁移完成 ✅

### 填充背景 Token（未发现）
扫描未发现 `--color-fill-1/2/3/4` 的使用，说明填充背景已经迁移完成 ✅

### 语义色 Token（保留使用）
`--color-error/success/warning/info` 及其 `-bg` 变体是 **Arco 语义色**，应该保留使用。
这些不是需要禁用的 token，它们是：
- `rgb(var(--red-6))` 的别名
- `rgb(var(--green-6))` 的别名
- `rgb(var(--orange-6))` 的别名
- `rgb(var(--blue-6))` 的别名

**结论**: 当前扫描到的 `var(--color-*)` 主要是语义色，不需要迁移。

---

## 三、硬编码颜色使用情况

### 十六进制颜色
- **使用文件数**: 1 个（`index.css`）
- **结论**: ✅ 仅在 token 定义中使用，符合规范

---

## 四、硬编码间距使用情况

### 总体统计
- **使用文件数**: 10 个
- **需要迁移**: 是

### 详细清单

| 文件 | 硬编码次数 | 优先级 | 典型模式 |
|------|-----------|-------|---------|
| `index.css` | 123 | P2 | 全局基础样式，部分需要保留 |
| `modules/system/components/shared/list-page.css` | 127 | P1 | 列表页组件 |
| `core/layout/index.css` | 58 | P0 | 应用壳层 |
| `modules/platform/dashboard.css` | 23 | P1 | 工作台 |
| `modules/auth/login/components/Login.css` | 18 | P1 | 登录页 |
| `modules/lowcode/generator/pages/ModuleWizard.css` | 9 | P2 | 低代码生成器 |
| `components/patterns/filters/time-range-filter.css` | 9 | P2 | 时间筛选器 |
| `modules/system/user/user.css` | 8 | P1 | 用户管理 |
| `modules/auth/auth.css` | 4 | P1 | 认证模块 |
| `modules/lowcode/components/CodePreview.css` | 3 | P2 | 代码预览 |

### 常见硬编码模式

```css
/* ❌ 需要迁移 */
padding: 16px 20px;
margin-top: 12px;
gap: 8px;

/* ✅ 应该使用 */
padding: var(--space-lg) var(--space-xl);
margin-top: var(--space-md);
gap: var(--space-sm);
```

---

## 五、迁移优先级

### P0 - 立即迁移（应用壳层）
1. **`core/layout/index.css`** (58 处硬编码间距)
   - 影响: 全局布局、侧边栏、顶栏、面包屑
   - 工作量: 中等
   - 风险: 中（需要全页面回归测试）

### P1 - 优先迁移（高频页面）
2. **`modules/system/components/shared/list-page.css`** (127 处硬编码间距)
   - 影响: 所有系统管理列表页
   - 工作量: 大
   - 风险: 中

3. **`modules/platform/dashboard.css`** (23 处硬编码间距 + 20 处语义色)
   - 影响: 平台工作台
   - 工作量: 中等
   - 风险: 低

4. **`modules/auth/login/components/Login.css`** (18 处硬编码间距)
   - 影响: 登录页
   - 工作量: 小
   - 风险: 低

5. **`modules/auth/auth.css`** (4 处硬编码间距)
   - 影响: 认证模块
   - 工作量: 小
   - 风险: 低

6. **`modules/system/user/user.css`** (8 处硬编码间距)
   - 影响: 用户管理
   - 工作量: 小
   - 风险: 低

### P2 - 后续迁移（低频组件）
7. **`components/operational/operational.css`** (6 处语义色)
   - 影响: 操作日志组件
   - 工作量: 小
   - 风险: 低

8. **`modules/lowcode/generator/pages/ModuleWizard.css`** (9 处硬编码间距 + 1 处语义色)
   - 影响: 低代码生成器
   - 工作量: 小
   - 风险: 低

9. **`components/patterns/filters/time-range-filter.css`** (9 处硬编码间距)
   - 影响: 时间筛选器
   - 工作量: 小
   - 风险: 低

10. **`modules/lowcode/components/CodePreview.css`** (3 处硬编码间距)
    - 影响: 代码预览
    - 工作量: 极小
    - 风险: 低

### P2 - 选择性迁移（全局样式）
11. **`index.css`** (123 处硬编码间距 + 42 处语义色)
    - 说明: 全局基础样式，包含 token 定义、reset、Arco 覆盖
    - 策略: 选择性迁移，保留必要的硬编码（如 1px、2px 等）
    - 工作量: 大
    - 风险: 高

---

## 六、关键发现

### ✅ 已完成的迁移
1. **文本颜色**: 未发现 `--color-text-*` 使用
2. **边框颜色**: 未发现 `--color-border-*` 使用
3. **填充背景**: 未发现 `--color-fill-*` 使用
4. **容器背景**: 已使用 `--panel-bg-solid`、`--surface-lift` 等

### ⚠️ 需要明确的问题
1. **语义色 token**: `--color-error/success/warning/info` 是否需要迁移？
   - 当前判断: **不需要**，这些是 Arco 提供的语义色阶，等同于 `rgb(var(--red-6))` 等
   - 建议: 在 `check-ui-contract.mjs` 中将这些列为白名单

2. **硬编码间距**: 哪些是必要的，哪些需要迁移？
   - 1px、2px 的细微间距: 可以保留
   - 4px 及以上: 应该使用 `--space-*` token

---

## 七、迁移策略

### 阶段 3.1: P0 迁移（应用壳层）
- **文件**: `core/layout/index.css`
- **重点**: 侧边栏、顶栏、面包屑、页面容器
- **验证**: 四主题 + 暗色模式 + 移动端

### 阶段 3.2: P1 迁移（高频页面）
- **文件**: 
  - `modules/platform/dashboard.css`
  - `modules/auth/login/components/Login.css`
  - `modules/system/components/shared/list-page.css`
  - `modules/system/user/user.css`
- **重点**: 工作台、登录页、列表页、用户管理
- **验证**: 主要业务流程走查

### 阶段 3.3: P2 迁移（低频组件）
- **文件**: 其余 5 个文件
- **策略**: 批量迁移，统一测试

### 阶段 3.4: 全局样式优化
- **文件**: `index.css`
- **策略**: 选择性迁移，保留必要的硬编码
- **风险控制**: 分批提交，逐步验证

---

## 八、迁移模板

### 间距迁移模板

```css
/* Before */
.component {
  padding: 16px 20px;
  margin-bottom: 12px;
  gap: 8px;
}

/* After */
.component {
  padding: var(--space-lg) var(--space-xl);
  margin-bottom: var(--space-md);
  gap: var(--space-sm);
}
```

### 间距映射表

| 硬编码值 | Token | 备注 |
|---------|-------|------|
| `2px` | `var(--space-2xs)` | 极紧密 |
| `4px` | `var(--space-xs)` | 标签内部 |
| `8px` | `var(--space-sm)` | 同类控件 |
| `10px` | `var(--space-sm)` + 适当调整 | 非标准值 |
| `12px` | `var(--space-md)` | 表单行 |
| `16px` | `var(--space-lg)` | 卡片内边距 |
| `18px` | `var(--space-lg)` + 适当调整 | 非标准值 |
| `20px` | `var(--space-xl)` | 页面内容区 |
| `24px` | `var(--space-xl)` | 页面内容区 |
| `32px` | `var(--space-2xl)` | 大区块 |
| `48px` | `var(--space-3xl)` | 页面级分区 |

---

## 九、下一步行动

### Task 11: 批量迁移（预计 3-4 小时）
1. 迁移 `core/layout/index.css`（P0）
2. 迁移 `modules/platform/dashboard.css`（P1）
3. 迁移 `modules/auth/login/components/Login.css`（P1）
4. 迁移 `modules/system/components/shared/list-page.css`（P1）

### Task 12: 更新机械门禁（预计 1 小时）
1. 在 `check-ui-contract.mjs` 中添加语义色白名单
2. 添加硬编码间距检查（排除 1px、2px）
3. 添加容器 token 使用检查

### Task 13: 视觉回归测试（预计 1-2 小时）
1. 四主题 + 暗色模式手动走查
2. 移动端响应式测试
3. 关键页面截图对比

---

## 附录

### A. 完整文件路径

```
frontend/src/
  ├── index.css (123 间距 + 42 语义色)
  ├── core/layout/index.css (58 间距 + 15 语义色)
  ├── components/
  │   ├── operational/operational.css (6 语义色)
  │   └── patterns/filters/time-range-filter.css (9 间距)
  └── modules/
      ├── auth/
      │   ├── auth.css (4 间距)
      │   └── login/components/Login.css (18 间距)
      ├── platform/dashboard.css (23 间距 + 20 语义色)
      ├── system/
      │   ├── components/shared/list-page.css (127 间距 + 5 语义色)
      │   └── user/user.css (8 间距)
      └── lowcode/
          ├── components/CodePreview.css (3 间距)
          └── generator/pages/ModuleWizard.css (9 间距 + 1 语义色)
```

### B. 扫描命令记录

```bash
# 搜索 Arco color token
grep -r "var(--color-" --include="*.css" -l

# 统计使用次数
grep -c "var(--color-" <file>

# 搜索硬编码颜色
grep -r "#[0-9a-fA-F]\{3,6\}" --include="*.css" -l

# 搜索硬编码间距
grep -rE "padding:\s*[0-9]+px|margin:\s*[0-9]+px|gap:\s*[0-9]+px" --include="*.css" -c
```

---

**报告生成时间**: 2026-09-03  
**扫描工具**: grep + bash  
**下一步**: Task 11 - 批量迁移 P0/P1 文件
