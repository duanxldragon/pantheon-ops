import React from 'react';
import ReactDOM from 'react-dom/client';
import '@arco-design/web-react/es/_util/react-19-adapter.js';
import { BrowserRouter } from 'react-router-dom';
import './core/arco/style';
import './index.css';
import App from './App';
import { initI18n } from './i18n';
import { initializePantheonTheme } from './core/theme/theme';
import { initializePantheonColorMode } from './core/theme/colorMode';
import { initializePublicSettings } from './core/settings/publicSettings';

initializePantheonColorMode();

const root = ReactDOM.createRoot(document.getElementById('root') as HTMLElement);

await Promise.allSettled([initializePublicSettings(), initializePantheonTheme()]);
try {
  await initI18n();
} finally {
  root.render(
    <React.StrictMode>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </React.StrictMode>,
  );
}
