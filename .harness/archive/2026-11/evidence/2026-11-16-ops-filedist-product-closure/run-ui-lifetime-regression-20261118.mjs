import path from 'node:path';
import { fileURLToPath } from 'node:url';

const frontendRoot = fileURLToPath(new URL('../../../frontend/', import.meta.url));
const { startVitest } = await import(new URL('../../../frontend/node_modules/vitest/dist/node.js', import.meta.url));
const context = await startVitest([], {
  root: path.dirname(frontendRoot.replace(/[\\/]$/, '')),
  config: path.join(frontendRoot, 'vitest.config.ts'),
  include: ['.harness/evidence/2026-11-16-ops-filedist-product-closure/ui-lifetime.regression-20261118.test.ts'],
  run: true, pool: 'threads', maxWorkers: 1, fileParallelism: false, reporters: ['verbose'],
}, {
  resolve: { alias: {
    vitest: path.join(frontendRoot, 'node_modules/vitest/dist/index.js'),
    '@testing-library/react': path.join(frontendRoot, 'node_modules/@testing-library/react/dist/index.js'),
    react: path.join(frontendRoot, 'node_modules/react'),
    'react-router-dom': path.join(frontendRoot, 'node_modules/react-router-dom/dist/index.mjs'),
    'react-i18next': path.join(frontendRoot, 'node_modules/react-i18next/dist/es/index.js'),
    '@arco-design/web-react': path.join(frontendRoot, 'node_modules/@arco-design/web-react'),
  } },
});
if (!context) process.exitCode = 1;
else await context.close();
