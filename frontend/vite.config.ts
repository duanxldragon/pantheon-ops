import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import istanbul from 'vite-plugin-istanbul';

function isLocalHttpProxyTarget(target: string, command: string) {
  if (command !== 'serve') {
    return false;
  }

  try {
    const url = new URL(target);
    return url.protocol === 'http:' && ['127.0.0.1', 'localhost', '::1'].includes(url.hostname);
  } catch {
    return false;
  }
}

// https://vite.dev/config/
export default defineConfig(({ command }) => {
  const apiProxyTarget = process.env.PANTHEON_API_PROXY_TARGET || 'http://127.0.0.1:8080';
  const localHttpProxy = isLocalHttpProxyTarget(apiProxyTarget, command);

  return {
    plugins: [
      ...(process.env.PANTHEON_COLLECT_COVERAGE === '1'
        ? [
            istanbul({
              include: 'src/*',
              exclude: ['src/modules/generated/**', 'node_modules/**'],
              extension: ['.ts', '.tsx'],
              cypress: true,
            }),
          ]
        : []),
      react(),
    ],
    server: {
      hmr: process.env.PANTHEON_SMOKE === '1' ? { overlay: false } : undefined,
      proxy: {
        '/api': {
          target: apiProxyTarget,
          changeOrigin: true,
          configure: localHttpProxy
            ? (proxy) => {
                // The backend keeps Secure cookies unconditionally. Strip only
                // that flag for the local HTTP dev proxy so browsers can keep
                // the session; production and HTTPS targets remain unchanged.
                proxy.on('proxyRes', (proxyResponse) => {
                  const cookies = proxyResponse.headers['set-cookie'];
                  if (!cookies) {
                    return;
                  }
                  proxyResponse.headers['set-cookie'] = cookies.map((cookie) =>
                    cookie.replace(/;\s*Secure/gi, ''),
                  );
                });
              }
            : undefined,
        },
      },
    },
    build: {
      rolldownOptions: {
        output: {
          codeSplitting: {
            minSize: 30000,
            groups: [
              {
                name: 'react-vendor',
                test: /node_modules[\\/](react|react-dom|react-router-dom|scheduler)[\\/]/,
              },
              {
                name: 'arco-icons',
                test: /node_modules[\\/]@arco-design[\\/]web-react[\\/]icon[\\/]/,
              },
              {
                name: 'arco-table',
                test: /node_modules[\\/]@arco-design[\\/]web-react[\\/](es|lib)[\\/](Table|Pagination)[\\/]/,
              },
              {
                name: 'arco-tree',
                test: /node_modules[\\/]@arco-design[\\/]web-react[\\/](es|lib)[\\/](Tree|TreeSelect)[\\/]/,
              },
              {
                name: 'arco-shell',
                test: /node_modules[\\/]@arco-design[\\/]web-react[\\/](es|lib)[\\/](Layout|Menu|Dropdown|Breadcrumb|Avatar|Tooltip)[\\/]/,
              },
              {
                name: 'arco-feedback',
                test: /node_modules[\\/]@arco-design[\\/]web-react[\\/](es|lib)[\\/](Alert|Empty|Message|Modal|Popconfirm|Progress|Result|Spin)[\\/]/,
              },
              {
                name: 'arco-form-base',
                test: /node_modules[\\/]@arco-design[\\/]web-react[\\/](es|lib)[\\/](AutoComplete|Form|Input|InputNumber|Switch|Tabs)[\\/]/,
              },
              {
                name: 'arco-content',
                test: /node_modules[\\/]@arco-design[\\/]web-react[\\/](es|lib)[\\/](Button|Card|Space|Tag|Typography|Grid|Descriptions|Statistic)[\\/]/,
              },
              {
                name: 'arco-form-heavy',
                test: /node_modules[\\/]@arco-design[\\/]web-react[\\/](es|lib)[\\/](Select|Trigger|Checkbox)[\\/]/,
              },
              {
                name: 'arco-vendor',
                test: /node_modules[\\/]@arco-design[\\/]/,
              },
              {
                name: 'app-vendor',
                test: /node_modules[\\/](axios|zustand|i18next|react-i18next)[\\/]/,
              },
              {
                name: 'platform-builder',
                test: /frontend[\\/]src[\\/](modules[\\/](generator|system[\\/]dynamicmodule)|generator[\\/])|node_modules[\\/]jszip[\\/]/,
              },
            ],
          },
        },
      },
    },
  };
});
