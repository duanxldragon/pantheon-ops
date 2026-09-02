# 修复 business 模块的导入路径，改为从 npm 包导入

import os
import re

def fix_business_imports():
    print("=== 修复 business 模块导入路径 ===\n")

    business_dir = 'src/modules/business'

    # 需要替换的导入模式
    replacements = [
        # API 相关
        (r"from ['\"]\.\.\/\.\.\/api\/([^'\"]+)['\"]", r"from '@duanxldragon/pantheon-base-ui'"),
        (r"from ['\"]\.\.\/\.\.\/\.\.\/api\/([^'\"]+)['\"]", r"from '@duanxldragon/pantheon-base-ui'"),

        # Store 相关
        (r"from ['\"]\.\.\/\.\.\/store\/([^'\"]+)['\"]", r"from '@duanxldragon/pantheon-base-ui'"),
        (r"from ['\"]\.\.\/\.\.\/\.\.\/store\/([^'\"]+)['\"]", r"from '@duanxldragon/pantheon-base-ui'"),

        # Components 相关
        (r"from ['\"]\.\.\/\.\.\/components\/([^'\"]+)['\"]", r"from '@duanxldragon/pantheon-base-ui'"),
        (r"from ['\"]\.\.\/\.\.\/\.\.\/components\/([^'\"]+)['\"]", r"from '@duanxldragon/pantheon-base-ui'"),

        # Core/router 相关
        (r"from ['\"]\.\.\/\.\.\/core\/router\/([^'\"]+)['\"]", r"from '@duanxldragon/pantheon-base-ui'"),
        (r"from ['\"]\.\.\/\.\.\/\.\.\/core\/router\/([^'\"]+)['\"]", r"from '@duanxldragon/pantheon-base-ui'"),

        # Hooks 相关
        (r"from ['\"]\.\.\/\.\.\/hooks\/([^'\"]+)['\"]", r"from '@duanxldragon/pantheon-base-ui'"),
        (r"from ['\"]\.\.\/\.\.\/\.\.\/hooks\/([^'\"]+)['\"]", r"from '@duanxldragon/pantheon-base-ui'"),
    ]

    modified_files = 0
    total_replacements = 0

    for root, dirs, files in os.walk(business_dir):
        for file in files:
            if file.endswith(('.ts', '.tsx')):
                filepath = os.path.join(root, file)

                try:
                    with open(filepath, 'r', encoding='utf-8') as f:
                        content = f.read()

                    original_content = content
                    file_replacements = 0

                    for pattern, replacement in replacements:
                        new_content, count = re.subn(pattern, replacement, content)
                        if count > 0:
                            content = new_content
                            file_replacements += count

                    if content != original_content:
                        with open(filepath, 'w', encoding='utf-8') as f:
                            f.write(content)

                        modified_files += 1
                        total_replacements += file_replacements
                        print(f"[OK] {filepath} ({file_replacements} replacements)")

                except Exception as e:
                    print(f"[ERROR] {filepath}: {e}")

    print(f"\n[SUCCESS] 修复完成")
    print(f"- 修改文件数: {modified_files}")
    print(f"- 总替换数: {total_replacements}")

if __name__ == '__main__':
    os.chdir(r'D:\workspace\go\pantheon-platform\pantheon-ops\frontend')
    fix_business_imports()
