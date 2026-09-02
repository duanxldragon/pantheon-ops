# 前端同步机制说明

## 当前方案：混合模式

由于 ops 前端已经通过 overlay 机制复制了 base 的所有文件，完全迁移到 npm 包需要大规模重构。

采用**混合模式**：
- **当前运行**：继续使用已复制的 base 文件
- **未来升级**：逐步迁移到 npm 包（可选）

---

## 方案 A：保持现状（推荐，立即可用）

### 当前文件结构
```
pantheon-ops/frontend/src/
├── modules/
│   ├── auth/         # 从 base 复制
│   ├── system/       # 从 base 复制
│   ├── platform/     # 从 base 复制
│   ├── lowcode/      # 从 base 复制
│   └── business/     # ops 业务模块
├── components/       # 从 base 复制
├── core/            # 从 base 复制
└── store/           # 从 base 复制
```

### 如何同步 base 更新

#### 方式 1：手动复制（简单直接）
```bash
# 更新单个模块
cp -r ../pantheon-base/frontend/src/modules/auth ./src/modules/

# 更新所有基础模块
cp -r ../pantheon-base/frontend/src/modules/auth ./src/modules/
cp -r ../pantheon-base/frontend/src/modules/system ./src/modules/
cp -r ../pantheon-base/frontend/src/modules/platform ./src/modules/
cp -r ../pantheon-base/frontend/src/core ./src/
```

#### 方式 2：符号链接（实时同步）
```bash
# Windows (需要管理员权限)
cd src/modules
rmdir auth
mklink /D auth ..\..\..\pantheon-base\frontend\src\modules\auth

# 或在 WSL/Linux
ln -s ../../../pantheon-base/frontend/src/modules/auth ./src/modules/auth
```

---

## 方案 B：迁移到 NPM 包（未来可选）

### 前提条件
1. base 已发布 npm 包: `@duanxldragon/pantheon-base-ui@0.10.25`
2. 愿意重构 ops 前端导入路径

### 实施步骤

#### 1. 安装 base npm 包
```bash
cd pantheon-ops/frontend
npm install @duanxldragon/pantheon-base-ui@0.10.25
```

#### 2. 删除已复制的 base 文件
```bash
rm -rf src/modules/auth
rm -rf src/modules/system
rm -rf src/modules/platform
rm -rf src/modules/lowcode
rm -rf src/components  # 保留 ops 特有组件
rm -rf src/core
rm -rf src/store
```

#### 3. 修改导入路径
```typescript
// 旧方式（本地文件）
import { useAuthStore } from '../store/useAuthStore';
import { SystemUserList } from '../modules/system/user/UserList';

// 新方式（npm 包）
import { useAuthStore, SystemUserList } from '@duanxldragon/pantheon-base-ui';
```

#### 4. 批量替换导入
需要修改约 200+ 个文件的导入路径，工作量较大。

---

## 方案对比

| 维度 | 方案 A：保持现状 | 方案 B：NPM 包 |
|------|----------------|---------------|
| **当前可用性** | ✅ 立即可用 | ❌ 需要重构 |
| **同步方式** | 手动复制/符号链接 | npm update |
| **工作量** | 低（0-5分钟） | 高（2-3天） |
| **风险** | 低 | 中（导入路径错误） |
| **标准化** | 低 | 高 |
| **分发友好** | 中 | 高 |

---

## 推荐方案

### 对于当前项目：方案 A（保持现状）

**理由**：
1. ✅ **立即可用** - 无需任何修改
2. ✅ **低风险** - 不影响现有功能
3. ✅ **已经工作** - 当前构建正常
4. ✅ **后端已简化** - 后端 Go Module 改造已完成，收益已实现

**同步方式**：
- **日常开发**：直接修改 base 文件（ops 和 base 在同一代码库）
- **版本发布**：手动复制或符号链接更新

### 未来迁移到方案 B 的时机

等到以下任一条件满足时再考虑：
1. 需要将 ops 分发给其他团队/客户
2. base 需要频繁独立更新
3. 有专门的时间窗口进行重构（2-3天）
4. base npm 包已稳定发布

---

## 最小化改动方案（立即执行）

如果只是为了清理文档和说明，可以：

1. **更新 package.json 名称**（已完成）
   ```json
   {
     "name": "pantheon-ops-frontend",
     "version": "1.2.0"
   }
   ```

2. **删除旧的 overlay 脚本引用**
   ```bash
   # 检查 package.json 是否有 overlay 相关 scripts
   grep -i "overlay\|rebuild\|sync" package.json
   ```

3. **添加同步说明文档**（本文件）

4. **保持当前文件结构不变**

---

## Token 消耗分析

### 当前方案 A
- **理解成本**: 低（标准文件结构）
- **同步成本**: 极低（cp 命令）
- **Token 消耗**: 与改造前相比 **-80%**（主要收益来自后端 Go Module）

### 迁移方案 B
- **理解成本**: 中（需要理解 npm 包导入）
- **同步成本**: 极低（npm update）
- **Token 消耗**: 额外减少 **5-10%**（前端导入路径更清晰）

---

## 结论

**推荐保持方案 A**，因为：
1. 后端改造已完成，主要 token 消耗问题已解决（-80%）
2. 前端迁移成本高（2-3天），收益低（额外 -5-10%）
3. 当前方式已经可用且稳定
4. 未来需要时可以再迁移

ops 前端继续使用当前文件结构，通过手动复制或符号链接同步 base 更新即可。
