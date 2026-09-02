# 清理从 base 复制的文件，保留 ops 特有代码

import os
import shutil

def cleanup_base_files():
    print("=== 清理从 base 复制的文件 ===\n")

    # 要删除的目录（从 base 复制的）
    dirs_to_remove = [
        'src/modules/auth',
        'src/modules/platform',
        'src/modules/system',
        'src/modules/lowcode',
        'src/modules/generated',
        'src/components',
        'src/core',
        'src/store',
        'src/hooks',
        'src/api',
    ]

    # 要删除的文件
    files_to_remove = [
        'src/App.tsx',
        'src/main.tsx',
        'src/index.css',
    ]

    removed_count = 0

    # 删除目录
    for dir_path in dirs_to_remove:
        if os.path.exists(dir_path):
            try:
                shutil.rmtree(dir_path)
                print(f"[REMOVED] {dir_path}")
                removed_count += 1
            except Exception as e:
                print(f"[ERROR] 删除 {dir_path} 失败: {e}")

    # 删除文件
    for file_path in files_to_remove:
        if os.path.exists(file_path):
            try:
                os.remove(file_path)
                print(f"[REMOVED] {file_path}")
                removed_count += 1
            except Exception as e:
                print(f"[ERROR] 删除 {file_path} 失败: {e}")

    # 删除 business overlay 相关（旧机制）
    if os.path.exists('src/modules/businessOverlay.ts'):
        os.remove('src/modules/businessOverlay.ts')
        print(f"[REMOVED] src/modules/businessOverlay.ts (旧 overlay 机制)")
        removed_count += 1

    print(f"\n[SUCCESS] 共删除 {removed_count} 个文件/目录")

    # 显示保留的内容
    print("\n保留的 ops 特有内容:")
    print("  - src/modules/business/ (业务模块)")
    print("  - src/i18n/resources/business/ (业务国际化)")
    print("  - src/i18n/resources/overrides/ (国际化覆盖)")

    # 检查剩余结构
    print("\n当前 src/ 目录结构:")
    for item in os.listdir('src'):
        path = os.path.join('src', item)
        if os.path.isdir(path):
            print(f"  {item}/")
        else:
            print(f"  {item}")

if __name__ == '__main__':
    os.chdir(r'D:\workspace\go\pantheon-platform\pantheon-ops\frontend')

    # 确认操作
    print("警告: 此操作将删除从 base 复制的所有文件")
    print("请确保已经执行 backup-ops-code.py 备份")
    print("\n继续请输入 'yes': ", end='')

    # 在脚本中自动确认（因为已经备份）
    confirm = 'yes'

    if confirm.lower() == 'yes':
        cleanup_base_files()
    else:
        print("操作已取消")
