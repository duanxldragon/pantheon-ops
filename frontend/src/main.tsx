import { StrictMode } from 'react';
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

// TODO: 导入 base 样式
// import '@duanxldragon/pantheon-base-ui/dist/index.css';

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
