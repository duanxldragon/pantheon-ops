# 恢复 ops 特有代码

import os
import shutil

def restore_ops_code(backup_dir):
    print(f"=== 从 {backup_dir} 恢复 ops 代码 ===\n")

    if not os.path.exists(backup_dir):
        print(f"[ERROR] 备份目录不存在: {backup_dir}")
        return

    # 1. 恢复 business 模块
    print("恢复 business 模块...")
    if os.path.exists(f'{backup_dir}/business'):
        os.makedirs('src/modules', exist_ok=True)
        shutil.copytree(f'{backup_dir}/business', 'src/modules/business')
        print("  [OK] business 模块已恢复")

    # 2. 恢复 business 国际化
    print("恢复 business 国际化...")
    if os.path.exists(f'{backup_dir}/i18n/business'):
        os.makedirs('src/i18n/resources', exist_ok=True)
        shutil.copytree(f'{backup_dir}/i18n/business', 'src/i18n/resources/business')
        print("  [OK] business 国际化已恢复")

    # 3. 创建 business 模块入口
    print("创建 business 模块入口...")
    business_index = """import type { ModuleConfig } from '@duanxldragon/pantheon-base-ui';

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
"""
    with open('src/modules/business/index.ts', 'w', encoding='utf-8') as f:
        f.write(business_index)
    print("  [OK] src/modules/business/index.ts 已创建")

    # 4. 创建新的 main.tsx
    print("创建新的 main.tsx...")
    main_tsx = """import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { I18nextProvider } from 'react-i18next';
import { ConfigProvider } from '@arco-design/web-react';
import enUS from '@arco-design/web-react/es/locale/en-US';
import zhCN from '@arco-design/web-react/es/locale/zh-CN';

// 从 npm 包导入 base
import { App as BaseApp } from '@duanxldragon/pantheon-base-ui';

// 导入 ops 特有内容
import i18n from './i18n';
import { businessModules } from './modules/business';

// 导入样式（需要 base 发布 css）
// import '@duanxldragon/pantheon-base-ui/dist/style.css';

const locale = i18n.language === 'en-US' ? enUS : zhCN;

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <ConfigProvider locale={locale}>
        <I18nextProvider i18n={i18n}>
          <BaseApp additionalModules={businessModules} />
        </I18nextProvider>
      </ConfigProvider>
    </BrowserRouter>
  </StrictMode>
);
"""
    with open('src/main.tsx', 'w', encoding='utf-8') as f:
        f.write(main_tsx)
    print("  [OK] src/main.tsx 已创建")

    # 5. 创建临时 vite-env.d.ts
    print("创建 vite-env.d.ts...")
    vite_env = """/// <reference types="vite/client" />
"""
    with open('src/vite-env.d.ts', 'w', encoding='utf-8') as f:
        f.write(vite_env)
    print("  [OK] src/vite-env.d.ts 已创建")

    print("\n[SUCCESS] ops 代码恢复完成")
    print("\n下一步:")
    print("  1. 修改 business 模块导入路径: python fix-business-imports.py")
    print("  2. 验证构建: npm run build")

if __name__ == '__main__':
    os.chdir(r'D:\workspace\go\pantheon-platform\pantheon-ops\frontend')

    # 找到最新的备份目录
    backups = [d for d in os.listdir('.') if d.startswith('.frontend-backup-')]
    if not backups:
        print("[ERROR] 未找到备份目录")
        exit(1)

    latest_backup = sorted(backups)[-1]
    print(f"使用备份: {latest_backup}\n")

    restore_ops_code(latest_backup)
