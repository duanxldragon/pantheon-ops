# 批量创建所有需要的占位文件

import os

# 所有需要创建的占位文件
placeholder_files = [
    'src/modules/auth/login/components/Login.css',
    'src/modules/auth/mfa/components/MFAVerify.css',
    'src/modules/auth/security/components/SecurityCenter.css',
    'src/modules/system/user/user.css',
    'src/modules/system/role/role.css',
    'src/modules/system/menu/menu.css',
    'src/modules/system/dept/dept.css',
    'src/modules/system/post/post.css',
    'src/modules/system/permission/permission.css',
    'src/modules/system/setting/setting.css',
    'src/modules/system/dict/dict.css',
    'src/modules/system/i18n/i18n.css',
    'src/modules/system/audit/audit.css',
    'src/modules/lowcode/dynamicmodule/dynamicmodule.css',
    'src/modules/lowcode/generator/generator.css',
]

def create_all_placeholders():
    print("=== 批量创建占位文件 ===\n")

    created = 0
    skipped = 0

    for file_path in placeholder_files:
        # 创建目录
        dir_path = os.path.dirname(file_path)
        if dir_path and not os.path.exists(dir_path):
            os.makedirs(dir_path, exist_ok=True)

        # 创建文件
        if not os.path.exists(file_path):
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write('/* Placeholder - actual styles in @duanxldragon/pantheon-base-ui */\n')
            print(f"[OK] {file_path}")
            created += 1
        else:
            skipped += 1

    print(f"\n[SUCCESS] 创建 {created} 个文件，跳过 {skipped} 个已存在文件")

if __name__ == '__main__':
    os.chdir(r'D:\workspace\go\pantheon-platform\pantheon-ops\frontend')
    create_all_placeholders()
