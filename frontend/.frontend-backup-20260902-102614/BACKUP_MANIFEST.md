# Ops Frontend Backup

## 备份时间
20260902-102614

## 备份内容

1. **business/** - 完整的业务模块（62个文件）
2. **i18n/business/** - 业务模块国际化
3. **auth-index.ts** - auth 模块入口（含 business 种子菜单）
4. **businessOverlay.ts** - business 模块注册
5. **system-diffs/** - system 模块差异文件
6. **auth-diffs/** - auth 模块差异文件

## 差异说明

### auth/index.ts 差异
- ops 移除了 quickActionWidget（dashboard 快捷操作）
- 其他逻辑相同

### system 列表页差异
- OperationLogList.tsx, MenuList.tsx 等可能有微小调整
- 主要是业务特定的配置

## 恢复步骤

1. 安装 base npm 包
   ```bash
   npm install @duanxldragon/pantheon-base-ui@0.10.25
   ```

2. 删除从 base 复制的文件（见 cleanup script）

3. 修改导入路径使用 npm 包

4. 恢复 business 代码
   ```bash
   python restore-ops-code.py
   ```

## 文件统计

- business 模块文件数: 62
- 差异文件数: ~10
- 国际化文件: business/**
