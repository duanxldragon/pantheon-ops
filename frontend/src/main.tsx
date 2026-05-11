import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import './core/arco/style';
import './index.css';
import App from './App';
import { initI18n } from './i18n';
import { initializePantheonTheme } from './core/theme/theme';
import { initializePublicSettings } from './core/settings/publicSettings';

const root = ReactDOM.createRoot(document.getElementById('root') as HTMLElement);

Promise.allSettled([initializePublicSettings(), initializePantheonTheme()]).finally(() => {
  initI18n().finally(() => {
    root.render(
      <React.StrictMode>
        <BrowserRouter>
          <App />
        </BrowserRouter>
      </React.StrictMode>,
    );
  });
});
