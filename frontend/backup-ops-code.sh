#!/bin/bash
# 备份 ops 特有的业务代码和差异文件

BACKUP_DIR=".frontend-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "=== 备份 ops 特有代码 ==="

# 1. 备份整个 business 模块
echo "备份 business 模块..."
cp -r src/modules/business "$BACKUP_DIR/"

# 2. 备份 business 国际化
echo "备份 business 国际化..."
mkdir -p "$BACKUP_DIR/i18n"
cp -r src/i18n/resources/business "$BACKUP_DIR/i18n/"

# 3. 备份差异文件（种子菜单相关）
echo "备份差异文件..."
cp src/modules/auth/index.ts "$BACKUP_DIR/auth-index.ts"
cp src/modules/businessOverlay.ts "$BACKUP_DIR/"

# 4. 备份可能有差异的列表页
mkdir -p "$BACKUP_DIR/system-diffs"
cp src/modules/system/audit/OperationLogList.tsx "$BACKUP_DIR/system-diffs/" 2>/dev/null
cp src/modules/system/menu/MenuList.tsx "$BACKUP_DIR/system-diffs/" 2>/dev/null

# 5. 记录备份清单
cat > "$BACKUP_DIR/BACKUP_MANIFEST.md" << 'EOF'
# Ops Frontend Backup

## 备份内容

1. **business/** - 完整的业务模块
2. **i18n/business/** - 业务模块国际化
3. **auth-index.ts** - auth 模块入口（含 business 种子菜单）
4. **businessOverlay.ts** - business 模块注册
5. **system-diffs/** - system 模块差异文件

## 恢复步骤

1. 安装 base npm 包
2. 删除从 base 复制的文件
3. 修改导入路径
4. 恢复本备份中的文件
EOF

echo "备份完成: $BACKUP_DIR"
ls -lh "$BACKUP_DIR"
