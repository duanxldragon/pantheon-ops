# 备份 ops 特有的业务代码和差异文件
import os
import shutil
from datetime import datetime

def backup_ops_code():
    # 创建备份目录
    timestamp = datetime.now().strftime('%Y%m%d-%H%M%S')
    backup_dir = f'.frontend-backup-{timestamp}'
    os.makedirs(backup_dir, exist_ok=True)

    print("=== 备份 ops 特有代码 ===\n")

    # 1. 备份整个 business 模块
    print("备份 business 模块...")
    if os.path.exists('src/modules/business'):
        shutil.copytree('src/modules/business', f'{backup_dir}/business')
        print(f"  [OK] business 模块已备份")

    # 2. 备份 business 国际化
    print("备份 business 国际化...")
    os.makedirs(f'{backup_dir}/i18n', exist_ok=True)
    if os.path.exists('src/i18n/resources/business'):
        shutil.copytree('src/i18n/resources/business', f'{backup_dir}/i18n/business')
        print(f"  [OK] business 国际化已备份")

    # 3. 备份差异文件
    print("备份差异文件...")
    files_to_backup = [
        ('src/modules/auth/index.ts', 'auth-index.ts'),
        ('src/modules/businessOverlay.ts', 'businessOverlay.ts'),
    ]

    for src, dst in files_to_backup:
        if os.path.exists(src):
            shutil.copy2(src, f'{backup_dir}/{dst}')
            print(f"  [OK] {src} 已备份")

    # 4. 备份可能有差异的列表页
    print("备份 system 差异文件...")
    os.makedirs(f'{backup_dir}/system-diffs', exist_ok=True)
    system_diffs = [
        'src/modules/system/audit/OperationLogList.tsx',
        'src/modules/system/menu/MenuList.tsx',
        'src/modules/system/dept/DeptList.tsx',
        'src/modules/system/role/RoleList.tsx',
    ]

    for file in system_diffs:
        if os.path.exists(file):
            filename = os.path.basename(file)
            shutil.copy2(file, f'{backup_dir}/system-diffs/{filename}')
            print(f"  [OK] {file} 已备份")

    # 5. 备份 auth 模块差异
    print("备份 auth 差异文件...")
    os.makedirs(f'{backup_dir}/auth-diffs', exist_ok=True)
    auth_diffs = [
        'src/modules/auth/security/components/LoginLogList.tsx',
        'src/modules/auth/session/components/SessionList.tsx',
    ]

    for file in auth_diffs:
        if os.path.exists(file):
            filename = os.path.basename(file)
            shutil.copy2(file, f'{backup_dir}/auth-diffs/{filename}')
            print(f"  [OK] {file} 已备份")

    # 6. 记录备份清单
    manifest = """# Ops Frontend Backup

## 备份时间
{timestamp}

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
""".format(timestamp=timestamp)

    with open(f'{backup_dir}/BACKUP_MANIFEST.md', 'w', encoding='utf-8') as f:
        f.write(manifest)

    print(f"\n[SUCCESS] 备份完成: {backup_dir}")

    # 显示备份内容
    print("\n备份目录结构:")
    for root, dirs, files in os.walk(backup_dir):
        level = root.replace(backup_dir, '').count(os.sep)
        indent = ' ' * 2 * level
        print(f'{indent}{os.path.basename(root)}/')
        subindent = ' ' * 2 * (level + 1)
        for file in files[:5]:  # 只显示前5个文件
            print(f'{subindent}{file}')
        if len(files) > 5:
            print(f'{subindent}... 还有 {len(files) - 5} 个文件')

if __name__ == '__main__':
    os.chdir(r'D:\workspace\go\pantheon-platform\pantheon-ops\frontend')
    backup_ops_code()
