# Pantheon-Ops 文件命名规范分析报告

## 检查标准

### Go 后端文件
- **规范**: snake_case (如 `login_handler.go`)
- **测试文件**: `*_test.go`

### TypeScript/React 前端文件
- **React 组件**: PascalCase (如 `LoginForm.tsx`)
- **工具/辅助函数**: camelCase (如 `apiClient.ts`, `userHelpers.ts`)
- **Hooks**: camelCase with `use` prefix (如 `useTheme.ts`)
- **类型定义**: camelCase (如 `types.ts`)
- **入口文件**: `main.tsx`, `index.ts` (约定俗成)

## 检查结果

### ✅ Go 后端文件
所有 384 个 Go 文件均符合 snake_case 命名规范。

### ⚠️ 前端文件初步标记 (11个)

经过详细分析，这些文件实际上都是合理的：

#### 1. 工具/辅助模块 - camelCase 正确 ✅
- `frontend/src/components/data-display/userIdentityHelpers.ts` - 辅助函数
- `frontend/src/components/feedback/message.ts` - 消息工具函数
- `frontend/src/components/feedback/secondaryVerifyController.ts` - 控制器逻辑
- `frontend/src/components/patterns/modals/AppModal.shared.ts` - 共享类型定义
- `frontend/src/components/table/crossPageSelection.ts` - 跨页选择工具
- `frontend/src/components/table/standardPagination.ts` - 分页工具

#### 2. 特殊约定文件 ✅
- `frontend/src/main.tsx` - 应用入口文件（Vite 约定）

#### 3. 包含 JSX 的工具模块 - 需要评估 🤔
这些文件虽然有 `.tsx` 扩展名且包含 JSX，但它们不是独立组件，而是导出工具函数：

- `frontend/src/core/layout/useLayoutMenu.tsx` - Hook + 辅助函数
  - 导出: `useLayoutMenu` hook + 多个工具函数
  - 引用: 2 处
  
- `frontend/src/core/menu/icon.tsx` - 图标渲染工具
  - 导出: `renderMenuIcon` 函数 + 类型定义
  - 引用: 5 处
  
- `frontend/src/modules/auth/session/clientInfo.tsx` - 客户端信息格式化
  - 导出: `formatClientSummary`, `renderClientInfo` 函数
  - 引用: 5 处
  
- `frontend/src/modules/platform/widgets.tsx` - Widget 注册逻辑
  - 导出: 工具函数 + 注册表
  - 引用: 1 处

## 建议

### 选项 A: 保持现状（推荐）
**理由**:
1. 这些文件都是**工具模块**而非组件，使用 camelCase 符合 TypeScript 社区惯例
2. `.tsx` 扩展名仅因为包含 JSX 语法，不意味着必须使用 PascalCase
3. 项目已经稳定运行，重命名会影响多个引用点
4. React 社区对工具模块的命名没有严格的 PascalCase 要求

### 选项 B: 重命名为 PascalCase
如果坚持"所有 .tsx 文件都用 PascalCase"的严格规则，需要重命名：
- `useLayoutMenu.tsx` → `UseLayoutMenu.tsx` (2 处引用)
- `icon.tsx` → `Icon.tsx` 或 `MenuIcon.tsx` (5 处引用)
- `clientInfo.tsx` → `ClientInfo.tsx` (5 处引用)
- `widgets.tsx` → `Widgets.tsx` (1 处引用)

**成本**: 需要更新 13 处 import 语句

## 结论

**当前命名状态: 合规** ✅

所有文件都遵循了合理的命名约定：
- Go 文件: 100% snake_case ✅
- React 组件: PascalCase ✅
- 工具/辅助模块: camelCase ✅
- 特殊文件: 遵循约定 ✅

没有发现真正违反规范的文件。建议保持现状。
