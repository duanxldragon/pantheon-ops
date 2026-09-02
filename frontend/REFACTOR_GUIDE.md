# Ops 前端重构操作指南

**目标**: 使用 npm 包方式消费 pantheon-base，彻底移除 overlay 机制

---

## 前置检查

✅ **已完成备份**: `.frontend-backup-20260902-102614/`
- business 模块 (62 个文件)
- business 国际化
- 差异文件 (auth, system)

---

## 步骤 1: 清理从 base 复制的文件

### 手动执行（Windows PowerShell）

```powershell
cd D:\workspace\go\pantheon-platform\pantheon-ops\frontend

# 删除从 base 复制的模块
Remove-Item -Recurse -Force src\modules\auth
Remove-Item -Recurse -Force src\modules\platform
Remove-Item -Recurse -Force src\modules\system
Remove-Item -Recurse -Force src\modules\lowcode
Remove-Item -Recurse -Force src\modules\generated

# 删除从 base 复制的核心代码
Remove-Item -Recurse -Force src\components
Remove-Item -Recurse -Force src\core
Remove-Item -Recurse -Force src\store
Remove-Item -Recurse -Force src\hooks
Remove-Item -Recurse -Force src\api

# 删除从 base 复制的入口文件
Remove-Item -Force src\App.tsx
Remove-Item -Force src\main.tsx
Remove-Item -Force src\index.css

# 删除旧 overlay 机制文件
Remove-Item -Force src\modules\businessOverlay.ts

Write-Host "[SUCCESS] 清理完成"
```

### 或运行 Python 脚本（需要授权）

```bash
python cleanup-base-files.py
```

---

## 步骤 2: 安装 base npm 包

### 方式 A: 使用本地包（推荐，立即可用）

```bash
cd ../pantheon-base/frontend

# 打包本地版本
npm pack
# 生成: duanxldragon-pantheon-base-ui-0.10.25.tgz

cd ../../pantheon-ops/frontend

# 安装本地包
npm install ../../pantheon-base/frontend/duanxldragon-pantheon-base-ui-0.10.25.tgz
```

### 方式 B: 使用 npm registry（需要先发布）

```bash
# 先发布 base（在 pantheon-base/frontend）
npm publish --access public

# 然后安装（在 pantheon-ops/frontend）
npm install @duanxldragon/pantheon-base-ui@0.10.25
```

---

## 步骤 3: 恢复 ops 特有代码

运行恢复脚本（已生成）：

```bash
python restore-ops-code.py
```

这会：
1. 恢复 `src/modules/business/` 
2. 恢复 business 国际化
3. 创建新的入口文件使用 npm 包

---

## 步骤 4: 创建新的入口文件

### src/main.tsx
```typescript
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { I18nextProvider } from 'react-i18next';
import { ConfigProvider } from '@arco-design/web-react';
import enUS from '@arco-design/web-react/es/locale/en-US';
import zhCN from '@arco-design/web-react/es/locale/zh-CN';

// 从 npm 包导入 base 组件
import { App as BaseApp } from '@duanxldragon/pantheon-base-ui';

// 导入 ops 特有的 i18n 和模块
import i18n from './i18n';
import { businessModules } from './modules/business';

// 导入 base 样式
import '@duanxldragon/pantheon-base-ui/dist/style.css';

const locale = i18n.language === 'en-US' ? enUS : zhCN;

createRoot(document.getElementById('root')!).render(
  <BrowserRouter>
    <ConfigProvider locale={locale}>
      <I18nextProvider i18n={i18n}>
        <BaseApp additionalModules={businessModules} />
      </I18nextProvider>
    </ConfigProvider>
  </BrowserRouter>
);
```

### src/modules/business/index.ts
```typescript
import type { ModuleConfig } from '@duanxldragon/pantheon-base-ui';

import { BizScopeModule } from './bizscope';
import { CmdbModule } from './cmdb';
import { DeployModule } from './deploy';
import { K8sModule } from './k8s';
import { ServiceModule } from './service';
import { CICDModule } from './cicd';
import { FiledistModule } from './filedist';
import { ObservabilityModule } from './observability';

export const businessModules: ModuleConfig[] = [
  BizScopeModule,
  CmdbModule,
  DeployModule,
  K8sModule,
  ServiceModule,
  CICDModule,
  FiledistModule,
  ObservabilityModule,
];
```

---

## 步骤 5: 修改 business 模块导入路径

所有 business 模块内的导入需要改为从 npm 包导入：

### 批量替换（Python 脚本已生成）

```bash
python fix-business-imports.py
```

这会将：
```typescript
// 旧
import { api } from '../../api/apiClient';
import { useAuthStore } from '../../store/useAuthStore';

// 改为
import { api, useAuthStore } from '@duanxldragon/pantheon-base-ui';
```

---

## 步骤 6: 验证构建

```bash
npm run build
```

预期结果：构建成功

---

## 步骤 7: 测试运行

```bash
npm run dev
```

访问 http://localhost:5173 验证功能正常

---

## 回滚方案

如果出现问题，可以快速回滚：

```bash
# 1. 恢复备份
python rollback-from-backup.py .frontend-backup-20260902-102614

# 2. 卸载 npm 包
npm uninstall @duanxldragon/pantheon-base-ui

# 3. 重新构建
npm run build
```

---

## 当前状态

- ✅ 步骤 0: 备份完成
- ⏳ 步骤 1: 等待清理 base 文件
- ⏳ 步骤 2: 等待安装 npm 包
- ⏳ 步骤 3-7: 后续步骤

---

## 下一步

**选择执行方式**：

1. **手动执行** - 按照上述 PowerShell 命令逐步操作
2. **授予权限后自动执行** - 运行 `python cleanup-base-files.py`

然后告诉我执行结果，我会继续后续步骤。
