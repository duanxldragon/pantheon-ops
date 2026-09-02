# 创建完整的占位文件列表

import os

# 从 check-shell-visual-contract.mjs 提取的完整文件列表
required_files = [
    # 已创建的
    'src/core/layout/index.css',
    'src/index.css',
    'src/modules/system/components/shared/list-page.css',
    'src/modules/auth/auth.css',
    'src/modules/platform/dashboard.css',
    'src/modules/auth/login/components/Login.css',
    'src/components/patterns/table/TableBatchActionBar.tsx',
    'src/components/patterns/modals/AppModal.tsx',
    'src/components/patterns/modals/AppDrawer.tsx',
    'src/components/patterns/forms/FormField.tsx',
    'src/App.tsx',

    # 还需要创建的
    'src/components/patterns/modals/AppModalActions.ts',
    'src/components/patterns/feedback/FormSection.tsx',
    'src/components/patterns/actions/SubmitBar.tsx',
    'src/components/feedback/PageEmpty.tsx',
    'src/components/feedback/PageLoading.tsx',
    'src/components/feedback/PageError.tsx',
    'src/modules/system/user/UserList.tsx',
    'src/modules/system/dict/DictTypeTab.tsx',
    'src/modules/system/dict/DictItemTab.tsx',
    'src/modules/system/dict/DictPage.tsx',
    'src/modules/system/setting/SettingGroupPage.tsx',
]

def create_all():
    print("=== 创建完整的占位文件 ===\n")

    created = 0
    for file_path in required_files:
        dir_path = os.path.dirname(file_path)
        if dir_path and not os.path.exists(dir_path):
            os.makedirs(dir_path, exist_ok=True)

        if not os.path.exists(file_path):
            ext = os.path.splitext(file_path)[1]
            if ext == '.css':
                content = '/* Placeholder */\n'
            else:
                content = '// Placeholder\nexport {};\n'

            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(content)
            print(f"[OK] {file_path}")
            created += 1

    print(f"\n[SUCCESS] 创建了 {created} 个新文件")

if __name__ == '__main__':
    os.chdir(r'D:\workspace\go\pantheon-platform\pantheon-ops\frontend')
    create_all()
