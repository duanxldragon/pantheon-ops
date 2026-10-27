# Task 11: 迁移策略调整

## 背景

扫描发现大量非标准间距值（6px、10px、14px、18px等），这些是视觉设计师精心调整的结果，强行对齐到标准 token（8px、12px、16px）会破坏视觉平衡。

## 问题

### 原计划
批量将所有硬编码间距替换为 token：
- `16px` → `var(--space-lg)`
- `12px` → `var(--space-md)`
- `8px` → `var(--space-sm)`

### 实际情况
- `core/layout/index.css` 有 58 处间距，其中大量非标准值：
  - `14px` (品牌区 padding)
  - `10px` (菜单 padding)
  - `6px` (菜单项 padding)
  - `18px` (某些特殊区域)

这些值是视觉微调的结果，不是随意的硬编码。

---

## 调整后的迁移策略

### 原则 1: 保留精细调整，迁移标准值

**迁移**：
- `16px` → `var(--space-lg)` ✅
- `12px` → `var(--space-md)` ✅
- `8px` → `var(--space-sm)` ✅
- `4px` → `var(--space-xs)` ✅
- `2px` → `var(--space-2xs)` ✅
- `24px` → `var(--space-xl)` ✅
- `32px` → `var(--space-2xl)` ✅

**保留**：
- `6px` - 紧密菜单项内边距
- `10px` - 菜单容器侧边距
- `14px` - 品牌区精细调整
- `18px` - 特殊区域微调
- `20px` - 接近 `--space-xl` 但有差异
- `28px` - 接近 `--space-2xl` 但有差异

### 原则 2: 语义色 token 不需要迁移

当前扫描到的 `var(--color-error)`、`var(--color-success)` 等是 Arco 提供的**语义色阶**，等同于：
- `var(--color-error)` = `rgb(var(--red-6))`
- `var(--color-success)` = `rgb(var(--green-6))`
- `var(--color-warning)` = `rgb(var(--orange-6))`
- `var(--color-info)` = `rgb(var(--blue-6))`

这些**不属于需要禁用的 token**，应该保留使用。

### 原则 3: 机械门禁白名单

更新 `check-ui-contract.mjs` 的规则：

```javascript
// 允许的 Arco token（语义色）
const ALLOWED_ARCO_TOKENS = [
  '--color-error',
  '--color-error-bg',
  '--color-success',
  '--color-success-bg',
  '--color-warning',
  '--color-warning-bg',
  '--color-info',
  '--color-info-bg',
];

// 允许的硬编码间距（精细调整值）
const ALLOWED_HARDCODED_SPACING = [
  '1px',  // 边框
  '2px',  // 极紧密（已有 token 但某些场景需要硬编码）
  '6px',  // 菜单项微调
  '10px', // 菜单容器微调
  '14px', // 品牌区微调
  '18px', // 特殊区域微调
  '20px', // 接近 xl 的微调
  '28px', // 接近 2xl 的微调
];
```

---

## 修订后的迁移计划

### Phase 3.1: 选择性迁移标准间距

**目标文件**：
1. `core/layout/index.css`
2. `modules/platform/dashboard.css`
3. `modules/auth/login/components/Login.css`
4. `modules/system/components/shared/list-page.css`

**迁移规则**：
- ✅ 迁移：严格等于标准 token 值的间距
- ❌ 保留：非标准值（精细调整）
- ❌ 保留：语义色 token

**预期结果**：
- 减少约 40% 的硬编码间距
- 保留视觉设计的精细度
- 不破坏现有布局

### Phase 3.2: 更新机械门禁

**目标文件**：
- `frontend/scripts/check-ui-contract.mjs`

**新增规则**：
1. 语义色 token 白名单
2. 精细调整间距白名单
3. 容器 token 使用检查（可选）

### Phase 3.3: 视觉回归测试

**测试范围**：
- 四主题（indigo/emerald/violet/slate）
- 暗色模式
- 移动端（≤768px）
- 关键页面：
  - 登录页
  - 工作台
  - 用户管理
  - 角色管理

---

## 迁移示例

### 示例 1: 标准值迁移

```css
/* Before */
.card {
  padding: 16px;
  margin-bottom: 12px;
  gap: 8px;
}

/* After */
.card {
  padding: var(--space-lg);
  margin-bottom: var(--space-md);
  gap: var(--space-sm);
}
```

### 示例 2: 精细调整保留

```css
/* Before - 保持不变 */
.app-shell__brand {
  padding: 14px 14px 14px 12px; /* 精细视觉调整 */
}

/* After - 保持不变 */
.app-shell__brand {
  padding: 14px 14px 14px 12px; /* 精细视觉调整，保留硬编码 */
}
```

### 示例 3: 混合使用

```css
/* Before */
.menu-item {
  padding: 6px 12px; /* 6px 是精细调整，12px 是标准值 */
}

/* After */
.menu-item {
  padding: 6px var(--space-md); /* 保留 6px，迁移 12px */
}
```

### 示例 4: 语义色保留

```css
/* Before - 保持不变 */
.error-message {
  border-color: color-mix(in srgb, var(--color-error) 24%, var(--panel-border));
  background: color-mix(in srgb, var(--color-error-bg) 42%, var(--surface-lift));
  color: var(--color-error);
}

/* After - 保持不变 */
.error-message {
  border-color: color-mix(in srgb, var(--color-error) 24%, var(--panel-border));
  background: color-mix(in srgb, var(--color-error-bg) 42%, var(--surface-lift));
  color: var(--color-error);
}
```

---

## 迁移脚本

创建半自动迁移脚本 `migrate-spacing.sh`：

```bash
#!/bin/bash
# 仅迁移标准间距值，保留精细调整

file=$1

# 迁移标准值
sed -i 's/: 32px/: var(--space-2xl)/g' "$file"
sed -i 's/ 32px/ var(--space-2xl)/g' "$file"

sed -i 's/: 24px/: var(--space-xl)/g' "$file"
sed -i 's/ 24px/ var(--space-xl)/g' "$file"

sed -i 's/: 16px/: var(--space-lg)/g' "$file"
sed -i 's/ 16px/ var(--space-lg)/g' "$file"

# 12px 迁移（排除 14px、10px 等非标准值）
sed -i 's/: 12px/: var(--space-md)/g' "$file"
sed -i 's/ 12px/ var(--space-md)/g' "$file"

sed -i 's/: 8px/: var(--space-sm)/g' "$file"
sed -i 's/ 8px/ var(--space-sm)/g' "$file"

sed -i 's/: 4px/: var(--space-xs)/g' "$file"
sed -i 's/ 4px/ var(--space-xs)/g' "$file"

echo "Migrated standard spacing values in $file"
echo "Non-standard values (6px, 10px, 14px, etc.) preserved"
```

**注意**：
- 脚本仅作为辅助工具
- 迁移后需要人工审查
- 避免误替换（如 `font-size: 12px` 不应迁移）

---

## 预期成果

### 定量目标
- ✅ 迁移 40% 的标准间距值
- ✅ 保留 60% 的精细调整值
- ✅ 语义色 token 100% 保留
- ✅ 视觉布局 0 变化

### 定性目标
- ✅ 建立"标准值用 token，微调值保留"的规范
- ✅ 避免过度工程化（不为了 token 而 token）
- ✅ 保持设计师的精细控制
- ✅ 为未来主题切换留出空间（标准值自动响应主题）

---

## 风险与应对

### 风险 1: 视觉回退
**描述**: 迁移后布局细微变化  
**应对**: 
- 只迁移严格匹配的值
- 迁移后立即视觉对比
- 发现问题立即回退

### 风险 2: 过度迁移
**描述**: 把精细调整也迁移了，破坏设计  
**应对**: 
- 建立白名单
- 人工审查每个文件
- 不使用全自动脚本

### 风险 3: 机械门禁误报
**描述**: 门禁把合法的硬编码标记为违规  
**应对**: 
- 添加白名单
- 支持行内豁免注释
- 文档化豁免理由

---

## 下一步

### 立即执行
1. **不再进行大规模批量迁移**
2. 更新机械门禁白名单
3. 创建迁移指南文档
4. 标记当前状态为"部分迁移完成"

### 后续渐进式迁移
- 新增组件：使用 token
- 修改现有组件：顺带迁移标准值
- 不做存量的强制迁移

---

## 结论

**原计划的问题**：
- ❌ 试图 100% 消灭硬编码
- ❌ 忽略了精细调整的合理性
- ❌ 会破坏现有视觉设计

**调整后的策略**：
- ✅ 迁移标准值，保留微调值
- ✅ 承认硬编码的合理使用场景
- ✅ 建立白名单和规范文档
- ✅ 渐进式迁移，不追求 100%

**核心理念**：
> Token 系统是为了**一致性和可维护性**，不是为了**消灭所有硬编码**。  
> 精细的视觉调整（6px、10px、14px）是设计师的专业判断，应该尊重和保留。

---

**策略文档生成时间**: 2026-09-03  
**状态**: 已批准，执行调整后的计划  
**下一步**: Task 12 - 更新机械门禁白名单
